package cppanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/benchmark"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestM24Corpus(t *testing.T) {
	manifest := filepath.Join("..", "..", "benchmark", "testdata", "m24-c-cpp.json")
	report, err := benchmark.Evaluate(context.Background(), manifest, New())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		t.Fatalf("M24 corpus failed: metrics=%+v failures=%v", report.Metrics, report.Failures)
	}
	if report.Metrics.TruePositive != 8 || report.Metrics.FalsePositive != 0 || report.Metrics.FalseNegative != 0 {
		t.Fatalf("metrics=%+v want 8 TP / 0 FP / 0 FN", report.Metrics)
	}
}

func TestDescriptorExcludesTargetDependentClaims(t *testing.T) {
	desc := New().Descriptor()
	if err := detector.ValidateDescriptor(desc); err != nil {
		t.Fatal(err)
	}
	for _, rule := range desc.Rules {
		if rule.Category == model.CategoryDeadCode || rule.Category == model.CategoryBloat {
			t.Fatalf("M24 must not declare target-dependent category %s", rule.Category)
		}
	}
}

func TestMalformedSourceFailsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.cpp"), []byte("int main("), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Analyze(context.Background(), root); err == nil {
		t.Fatal("malformed C++ source unexpectedly analyzed without error")
	}
}
