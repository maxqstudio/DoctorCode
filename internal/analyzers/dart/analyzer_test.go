package dartanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/benchmark"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestM26FrozenCorpus(t *testing.T) {
	manifest := filepath.Join("..", "..", "benchmark", "testdata", "m26-dart-flutter.json")
	report, err := benchmark.Evaluate(context.Background(), manifest, New())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Metrics.TruePositive != 4 || report.Metrics.FalsePositive != 0 || report.Metrics.FalseNegative != 0 {
		t.Fatalf("M26 frozen corpus mismatch: %+v failures=%v", report.Metrics, report.Failures)
	}
}

func TestDescriptorM26Authority(t *testing.T) {
	desc := New().Descriptor()
	if err := detector.ValidateDescriptor(desc); err != nil {
		t.Fatal(err)
	}
	for _, rule := range desc.Rules {
		if rule.Category == model.CategoryDeadCode || rule.Category == model.CategoryBloat || rule.SafeAutofix {
			t.Fatalf("M26 disallowed rule: %+v", rule)
		}
	}
}

func TestMalformedDartFailsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.dart"), []byte("class Broken { bool run("), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Analyze(context.Background(), root); err == nil {
		t.Fatal("malformed Dart unexpectedly analyzed without error")
	}
}

func TestGeneratedDartExcludedWithoutParsing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "model.g.dart"), []byte("class Broken { bool run("), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("generated source yielded %d findings", len(findings))
	}
}
