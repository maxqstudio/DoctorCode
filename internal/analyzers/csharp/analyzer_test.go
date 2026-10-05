package csharpanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/benchmark"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestM25Corpus(t *testing.T) {
	manifest := filepath.Join("..", "..", "benchmark", "testdata", "m25-csharp.json")
	report, err := benchmark.Evaluate(context.Background(), manifest, New())
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed {
		t.Fatalf("M25 corpus failed: metrics=%+v failures=%v", report.Metrics, report.Failures)
	}
	if report.Metrics.TruePositive != 4 || report.Metrics.FalsePositive != 0 || report.Metrics.FalseNegative != 0 {
		t.Fatalf("metrics=%+v want 4 TP / 0 FP / 0 FN", report.Metrics)
	}
}

func TestDescriptorStaysInsideM25Authority(t *testing.T) {
	desc := New().Descriptor()
	if err := detector.ValidateDescriptor(desc); err != nil {
		t.Fatal(err)
	}
	for _, rule := range desc.Rules {
		if rule.Category == model.CategoryDeadCode || rule.Category == model.CategoryBloat {
			t.Fatalf("M25 must not declare category %s", rule.Category)
		}
	}
}

func TestMalformedSourceFailsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.cs"), []byte("class Broken { void Run("), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Analyze(context.Background(), root); err == nil {
		t.Fatal("malformed C# source unexpectedly analyzed without error")
	}
}
