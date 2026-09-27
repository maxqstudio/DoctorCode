package pythonanalysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestHighSignalPythonRules(t *testing.T) {
	root := t.TempDir()
	source := `api_token = "sk_live_python_123456"

def target(value):
    return value

def _wrapper(value):
    return target(value)

def use(value):
    return _wrapper(value)

def _stale():
    return 1

def classify(value):
    if value is None:
        return "first"
    elif value is None:
        return "duplicate"
    return "other"

def enabled(value):
    if value:
        return True
    else:
        return False
`
	if err := os.WriteFile(filepath.Join(root, "sample.py"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if errors.Is(err, detector.ErrUnavailable) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}

	for _, rule := range []string{ruleDeadCode, ruleLogic, ruleSimplify, ruleSecurity, ruleBloat} {
		if !hasRule(findings, rule) {
			t.Fatalf("missing %s in %#v", rule, findings)
		}
	}
	for _, finding := range findings {
		if finding.RuleID == ruleSecurity && strings.Contains(strings.Join(finding.Evidence, "\n"), "sk_live_python_123456") {
			t.Fatal("security evidence leaked credential literal")
		}
		if finding.SafeAutofix {
			t.Fatalf("Python finding unexpectedly enables safe autofix: %#v", finding)
		}
	}
}

func TestPythonConservativeNegatives(t *testing.T) {
	root := t.TempDir()
	source := `api_token = "your_token_here"

def public_unused():
    return 1

def register(fn):
    return fn

@register
def _hook():
    return 1

def ready():
    return True

def logic():
    if ready():
        return 1
    elif ready():
        return 2
    return 3

def same(value):
    if value:
        return True
    else:
        return True
`
	if err := os.WriteFile(filepath.Join(root, "negative.py"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if errors.Is(err, detector.ErrUnavailable) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.RuleID == ruleSecurity || finding.RuleID == ruleLogic || finding.RuleID == ruleSimplify {
			t.Fatalf("conservative negative produced %s: %#v", finding.RuleID, finding)
		}
		if strings.Contains(finding.Summary, "_hook") || strings.Contains(finding.Summary, "public_unused") {
			t.Fatalf("decorated/private or public API negative misclassified: %#v", finding)
		}
	}
}

func hasRule(findings []model.Finding, rule string) bool {
	for _, finding := range findings {
		if finding.RuleID == rule {
			return true
		}
	}
	return false
}
