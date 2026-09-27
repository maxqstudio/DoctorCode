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

type m10PythonManifest struct {
	SchemaVersion int               `json:"schema_version"`
	Sources       []m10PythonSource `json:"sources"`
}

type m10PythonSource struct {
	ID         string           `json:"id"`
	Repository string           `json:"repository"`
	SHA        string           `json:"sha"`
	License    string           `json:"license"`
	Env        string           `json:"env"`
	Labels     []m10PythonLabel `json:"labels"`
}

type m10PythonLabel struct {
	Status          string `json:"status"`
	RuleID          string `json:"rule_id"`
	Path            string `json:"path"`
	LineStart       int    `json:"line_start"`
	SummaryContains string `json:"summary_contains"`
	Rationale       string `json:"rationale"`
}

func TestM10LabeledPythonRealWorld(t *testing.T) {
	data, err := os.ReadFile("m10-python-labels.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest m10PythonManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SchemaVersion != 1 || len(manifest.Sources) == 0 {
		t.Fatalf("invalid M10 Python label manifest: %#v", manifest)
	}

	validTotal := 0
	invalidTotal := 0

	for _, src := range manifest.Sources {
		src := src
		t.Run(src.ID, func(t *testing.T) {
			root := strings.TrimSpace(os.Getenv(src.Env))
			if root == "" {
				t.Skipf("%s is not set; M10 labeled Python validation runs in its dedicated CI workflow", src.Env)
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

			for _, label := range src.Labels {
				actual, found := findM10PythonFinding(findings, label)
				switch label.Status {
				case "VALID_FINDING":
					validTotal++
					if !found {
						t.Errorf("%s: expected valid Python finding missing: %#v", src.Repository, label)
					}
				case "INVALID_FINDING":
					invalidTotal++
					if found {
						t.Errorf("%s: labeled Python false positive emitted: %#v actual=%#v", src.Repository, label, actual)
					}
				default:
					t.Fatalf("%s: unsupported M10 label status %q", src.Repository, label.Status)
				}
			}
		})
	}

	t.Logf("M10 bounded Python labels: valid=%d invalid=%d", validTotal, invalidTotal)
}

func findM10PythonFinding(findings []model.Finding, label m10PythonLabel) (model.Finding, bool) {
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
