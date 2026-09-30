package rustanalysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/detector"
)

func TestDescriptorValid(t *testing.T) {
	if err := detector.ValidateDescriptor(New().Descriptor()); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzerFindsBoundedRustRules(t *testing.T) {
	root := t.TempDir()
	source := "const API_TOKEN: &str = \"hardcoded-production-token-12345\";\n\n" +
		"fn stale_helper() -> i32 { 1 }\n\n" +
		"pub fn enabled(flag: bool) -> bool {\n    if flag { true } else { false }\n}\n\n" +
		"pub fn classify(flag: bool) -> i32 {\n    if flag {\n        1\n    } else if flag {\n        2\n    } else {\n        3\n    }\n}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.rs"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{ruleDeadCode, ruleLogic, ruleSecurity, ruleSimplify} {
		found := false
		for _, finding := range findings {
			if finding.RuleID == rule {
				found = true
			}
			if finding.SafeAutofix {
				t.Fatalf("Rust finding unexpectedly enables safe autofix: %#v", finding)
			}
			if strings.Contains(strings.Join(finding.Evidence, "\\n"), "hardcoded-production-token-12345") {
				t.Fatal("security evidence leaked credential literal")
			}
		}
		if !found {
			t.Fatalf("missing %s in %#v", rule, findings)
		}
	}
}

func TestRustSyntaxFailureIsBlocking(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.rs"), []byte("fn broken( {\\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New().Analyze(context.Background(), root); err == nil || !strings.Contains(err.Error(), "rust syntax validation failed") {
		t.Fatalf("malformed Rust must fail closed, got %v", err)
	}
}

func TestRustUnavailableWithoutSource(t *testing.T) {
	_, err := New().Analyze(context.Background(), t.TempDir())
	if !errors.Is(err, detector.ErrUnavailable) {
		t.Fatalf("empty root error=%v want ErrUnavailable", err)
	}
}
