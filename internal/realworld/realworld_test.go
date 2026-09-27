package realworld

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type sourceManifest struct {
	SchemaVersion int      `json:"schema_version"`
	Sources       []source `json:"sources"`
}

type source struct {
	ID                         string   `json:"id"`
	Repository                 string   `json:"repository"`
	SHA                        string   `json:"sha"`
	License                    string   `json:"license"`
	Env                        string   `json:"env"`
	Observations               []string `json:"observations"`
	NotDeadcodeSummaryContains []string `json:"not_deadcode_summary_contains"`
}

func TestPinnedPublicRepositories(t *testing.T) {
	data, err := os.ReadFile("sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest sourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Sources) == 0 {
		t.Fatalf("invalid real-world source manifest: %#v", manifest)
	}

	for _, src := range manifest.Sources {
		src := src
		t.Run(src.ID, func(t *testing.T) {
			root := strings.TrimSpace(os.Getenv(src.Env))
			if root == "" {
				t.Skipf("%s is not set; pinned public-repository validation runs in the dedicated CI job", src.Env)
			}

			abs, err := resolveSourceRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyCheckoutSHA(abs, src.SHA); err != nil {
				t.Fatalf("%s provenance check failed: %v", src.Repository, err)
			}

			findings, err := goanalysis.New().Analyze(context.Background(), abs)
			if err != nil {
				t.Fatalf("%s analysis failed: %v", src.Repository, err)
			}

			for _, finding := range findings {
				assertFindingBoundary(t, src, finding)
			}
			for _, name := range src.NotDeadcodeSummaryContains {
				for _, finding := range findings {
					if finding.Category == model.CategoryDeadCode && strings.Contains(finding.Summary, name) {
						t.Fatalf("%s: known-live function %q was reported as DEADCODE: %#v", src.Repository, name, finding)
					}
				}
			}

			t.Logf("%s@%s analyzed with %d findings; results are compatibility evidence, not labeled precision", src.Repository, src.SHA[:12], len(findings))
		})
	}
}

func resolveSourceRoot(value string) (string, error) {
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	if workspace := strings.TrimSpace(os.Getenv("GITHUB_WORKSPACE")); workspace != "" {
		return filepath.Abs(filepath.Join(workspace, value))
	}
	return filepath.Abs(value)
}

func verifyCheckoutSHA(root, expected string) error {
	data, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return err
	}
	actual := strings.TrimSpace(string(data))
	if strings.HasPrefix(actual, "ref:") {
		return fmt.Errorf("expected detached exact-SHA checkout, got %q", actual)
	}
	if actual != expected {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}
	return nil
}

func assertFindingBoundary(t *testing.T, src source, finding model.Finding) {
	t.Helper()
	if finding.SafeAutofix {
		t.Fatalf("%s: real-world finding unexpectedly enables autofix: %#v", src.Repository, finding)
	}
	if filepath.IsAbs(finding.Path) {
		t.Fatalf("%s: finding path must remain repository-relative: %#v", src.Repository, finding)
	}
	clean := filepath.Clean(filepath.FromSlash(finding.Path))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		t.Fatalf("%s: finding path escaped repository root: %#v", src.Repository, finding)
	}
	switch finding.Category {
	case model.CategoryBloat, model.CategorySecurity, model.CategorySimplify, model.CategoryLogic, model.CategoryDeadCode:
	default:
		t.Fatalf("%s: unknown finding category: %#v", src.Repository, finding)
	}
	switch finding.Confidence {
	case model.ConfidenceProven, model.ConfidenceHigh, model.ConfidenceSuspicious, model.ConfidenceUnknown:
	default:
		t.Fatalf("%s: unknown finding confidence: %#v", src.Repository, finding)
	}
}
