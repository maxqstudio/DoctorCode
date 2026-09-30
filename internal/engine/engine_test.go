package engine

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAuditReportsOnlyAnalyzersRecognizingSource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.js"), []byte("const value = 1;\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Default().Audit(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"javascript/parser-backed-v2"}
	if !reflect.DeepEqual(result.Analyzers, want) {
		t.Fatalf("analyzers=%v want=%v", result.Analyzers, want)
	}
}

func TestAuditReportsNoAnalyzerForUnrecognizedRoot(t *testing.T) {
	result, err := Default().Audit(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Analyzers) != 0 {
		t.Fatalf("analyzers=%v want empty", result.Analyzers)
	}
}
