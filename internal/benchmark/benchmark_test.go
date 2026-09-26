package benchmark

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type fakeAnalyzer struct {
	findings []model.Finding
}

func (f fakeAnalyzer) Name() string {
	return "fake"
}

func (f fakeAnalyzer) Analyze(context.Context, string) ([]model.Finding, error) {
	return append([]model.Finding(nil), f.findings...), nil
}

func TestEvaluateCuratedGoCorpus(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	manifest := filepath.Join(filepath.Dir(current), "testdata", "manifest.json")

	report, err := Evaluate(context.Background(), manifest, goanalysis.New())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		t.Fatalf("benchmark failed: %#v", report.Failures)
	}
	if report.Metrics.TruePositive != 5 || report.Metrics.FalsePositive != 0 || report.Metrics.FalseNegative != 0 {
		t.Fatalf("unexpected aggregate metrics: %#v", report.Metrics)
	}
	if len(report.Rules) != 5 {
		t.Fatalf("expected five measured rules, got %d: %#v", len(report.Rules), report.Rules)
	}
}

func TestEvaluateReportsFalsePositive(t *testing.T) {
	root := t.TempDir()
	manifest := writeManifest(t, root, Manifest{
		SchemaVersion: 1,
		Language:      "Go",
		Thresholds:    Thresholds{MinPrecision: 1, MinRecall: 1},
		Cases:         []Case{{Name: "negative", Root: ".", Expected: []ExpectedFinding{}}},
	})
	analyzer := fakeAnalyzer{findings: []model.Finding{{
		RuleID: "RULE", Path: "sample.go", LineStart: 3, Summary: "unexpected",
	}}}

	report, err := Evaluate(context.Background(), manifest, analyzer)
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.Metrics.FalsePositive != 1 || report.Metrics.Precision != 0 {
		t.Fatalf("false positive did not fail benchmark: %#v", report)
	}
}

func TestEvaluateReportsFalseNegative(t *testing.T) {
	root := t.TempDir()
	manifest := writeManifest(t, root, Manifest{
		SchemaVersion: 1,
		Language:      "Go",
		Thresholds:    Thresholds{MinPrecision: 1, MinRecall: 1},
		Cases: []Case{{
			Name: "positive", Root: ".",
			Expected: []ExpectedFinding{{RuleID: "RULE", Path: "sample.go", LineStart: 3}},
		}},
	})

	report, err := Evaluate(context.Background(), manifest, fakeAnalyzer{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed || report.Metrics.FalseNegative != 1 || report.Metrics.Recall != 0 {
		t.Fatalf("false negative did not fail benchmark: %#v", report)
	}
}

func TestEvaluateRejectsEscapingCaseRoot(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "corpus")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := writeManifest(t, manifestDir, Manifest{
		SchemaVersion: 1,
		Language:      "Go",
		Thresholds:    Thresholds{MinPrecision: 1, MinRecall: 1},
		Cases:         []Case{{Name: "escape", Root: "../outside", Expected: []ExpectedFinding{}}},
	})

	_, err := Evaluate(context.Background(), manifest, fakeAnalyzer{})
	if err == nil || !strings.Contains(err.Error(), "escapes the manifest directory") {
		t.Fatalf("expected path escape rejection, got %v", err)
	}
}

func writeManifest(t *testing.T, root string, manifest Manifest) string {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
