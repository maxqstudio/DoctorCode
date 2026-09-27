package realworld

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pythonanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/python"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type m07PythonManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Sources       []m07PythonSource `json:"sources"`
}

type m07PythonSource struct {
	ID         string              `json:"id"`
	Repository string              `json:"repository"`
	SHA        string              `json:"sha"`
	License    string              `json:"license"`
	Env        string              `json:"env"`
	Expected   []m07PythonExpected `json:"expected"`
}

type m07PythonExpected struct {
	RuleID          string `json:"rule_id"`
	Path            string `json:"path"`
	LineStart       int    `json:"line_start"`
	SummaryContains string `json:"summary_contains"`
}

func TestM07PinnedPythonRepositories(t *testing.T) {
	data, err := os.ReadFile("m07-python-sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest m07PythonManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Sources) == 0 {
		t.Fatalf("invalid M07 Python source manifest: %#v", manifest)
	}

	for _, src := range manifest.Sources {
		src := src
		t.Run(src.ID, func(t *testing.T) {
			root := strings.TrimSpace(os.Getenv(src.Env))
			if root == "" {
				t.Skipf("%s is not set; M07 public Python validation runs in its dedicated CI workflow", src.Env)
			}
			abs, err := resolveSourceRoot(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := verifyCheckoutSHA(abs, src.SHA); err != nil {
				t.Fatalf("%s provenance check failed: %v", src.Repository, err)
			}

			findings, err := pythonanalysis.New().Analyze(context.Background(), abs)
			if err != nil {
				t.Fatalf("%s Python analysis failed: %v", src.Repository, err)
			}
			for _, finding := range findings {
				assertFindingBoundary(t, source{Repository: src.Repository}, finding)
			}
			for _, expected := range src.Expected {
				if !hasM07PythonFinding(findings, expected) {
					t.Fatalf("%s expected bounded finding missing: %#v findings=%#v", src.Repository, expected, findings)
				}
			}

			t.Logf("%s@%s parsed and analyzed with %d findings; only %d exact anchors are labeled", src.Repository, src.SHA[:12], len(findings), len(src.Expected))
		})
	}
}

func hasM07PythonFinding(findings []model.Finding, expected m07PythonExpected) bool {
	wantPath := filepath.ToSlash(expected.Path)
	for _, finding := range findings {
		if finding.RuleID != expected.RuleID || filepath.ToSlash(finding.Path) != wantPath {
			continue
		}
		if expected.LineStart > 0 && finding.LineStart != expected.LineStart {
			continue
		}
		if expected.SummaryContains != "" && !strings.Contains(finding.Summary, expected.SummaryContains) {
			continue
		}
		return true
	}
	return false
}
