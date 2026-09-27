package realworld

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type m06Manifest struct {
	SchemaVersion int         `json:"schema_version"`
	Sources       []m06Source `json:"sources"`
}

type m06Source struct {
	ID         string     `json:"id"`
	Repository string     `json:"repository"`
	SHA        string     `json:"sha"`
	License    string     `json:"license"`
	Env        string     `json:"env"`
	Labels     []m06Label `json:"labels"`
}

type m06Label struct {
	Status          string `json:"status"`
	RuleID          string `json:"rule_id"`
	Path            string `json:"path"`
	LineStart       int    `json:"line_start"`
	SummaryContains string `json:"summary_contains"`
	Rationale       string `json:"rationale"`
}

func TestM06LabeledRealWorld(t *testing.T) {
	data, err := os.ReadFile("m06-labels.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest m06Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("unexpected schema version: %d", manifest.SchemaVersion)
	}

	validTotal := 0
	invalidTotal := 0
	ambiguousTotal := 0

	for _, src := range manifest.Sources {
		src := src
		t.Run(src.ID, func(t *testing.T) {
			root := strings.TrimSpace(os.Getenv(src.Env))
			if root == "" {
				t.Skipf("%s is not set; M06 labeled validation runs in its dedicated CI workflow", src.Env)
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
				assertFindingBoundary(t, source{Repository: src.Repository}, finding)
			}

			for _, label := range src.Labels {
				actual, found := findM06Finding(findings, label)
				switch label.Status {
				case "VALID_FINDING":
					validTotal++
					if !found {
						t.Errorf("%s: expected valid finding missing: %#v", src.Repository, label)
					}
				case "INVALID_FINDING":
					invalidTotal++
					if found {
						t.Errorf("%s: labeled false positive still emitted: %#v actual=%#v", src.Repository, label, actual)
					}
				case "AMBIGUOUS":
					ambiguousTotal++
					if found && actual.SafeAutofix {
						t.Errorf("%s: ambiguous finding must never authorize autofix: %#v", src.Repository, actual)
					}
				default:
					t.Fatalf("%s: unknown label status %q", src.Repository, label.Status)
				}
			}
		})
	}

	t.Logf("M06 bounded labels: valid=%d invalid=%d ambiguous=%d", validTotal, invalidTotal, ambiguousTotal)
}

func findM06Finding(findings []model.Finding, label m06Label) (model.Finding, bool) {
	wantPath := filepath.ToSlash(label.Path)
	for _, finding := range findings {
		if finding.RuleID != label.RuleID || filepath.ToSlash(finding.Path) != wantPath || finding.LineStart != label.LineStart {
			continue
		}
		if label.SummaryContains != "" && !strings.Contains(finding.Summary, label.SummaryContains) {
			continue
		}
		return finding, true
	}
	return model.Finding{}, false
}
