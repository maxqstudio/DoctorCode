package goanalysis

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestHighSignalRules(t *testing.T) {
	root := t.TempDir()
	source := `package demo

const apiToken = "sk_live_1234567890"

func live(v int) int { return wrapper(v) }
func target(v int) int { return v }
func wrapper(v int) int { return target(v) }

func staleHelper() {}

func classify(x int) string {
	if x == 1 {
		return "one"
	} else if x == 2 {
		return "two"
	} else if x == 1 {
		return "duplicate"
	}
	return "other"
}

func boolIdentity(ok bool) bool {
	if ok {
		return true
	} else {
		return false
	}
}

func sideEffect() bool { return true }
func impure() int {
	if sideEffect() {
		return 1
	} else if sideEffect() {
		return 2
	}
	return 3
}
`
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}

	assertRuleForSummary(t, findings, ruleDeadCode, "staleHelper")
	assertRule(t, findings, ruleLogic)
	assertRule(t, findings, ruleSimplify)
	security := assertRule(t, findings, ruleSecurity)
	if strings.Contains(strings.Join(security.Evidence, "\n"), "sk_live_1234567890") {
		t.Fatal("security evidence leaked the credential literal")
	}
	assertRuleForSummary(t, findings, ruleBloat, "wrapper")

	for _, finding := range findings {
		if finding.RuleID == ruleLogic && finding.LineStart > 20 {
			t.Fatalf("impure repeated call was incorrectly classified as duplicate pure logic: %#v", finding)
		}
		if finding.SafeAutofix {
			t.Fatalf("M01 finding unexpectedly allows safe autofix: %#v", finding)
		}
	}
}

func TestDeadCodeFailsClosedOnLinkageEscapeHatch(t *testing.T) {
	root := t.TempDir()
	source := `package demo

//go:linkname externallyLinked runtime.someSymbol
func externallyLinked() {}
`
	if err := os.WriteFile(filepath.Join(root, "linked.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.RuleID == ruleDeadCode {
			t.Fatalf("dead-code detector must fail closed when //go:linkname is present: %#v", finding)
		}
	}
}

func TestDeadCodeCountsTestReferences(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package demo\nfunc testOnlyHelper() int { return 1 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample_test.go"), []byte("package demo\nfunc useIt() int { return testOnlyHelper() }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.RuleID == ruleDeadCode && strings.Contains(finding.Summary, "testOnlyHelper") {
			t.Fatalf("function referenced by tests must not be zero-reference dead code: %#v", finding)
		}
	}
}

func assertRule(t *testing.T, findings []model.Finding, rule string) model.Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.RuleID == rule {
			return finding
		}
	}
	t.Fatalf("missing rule %s in findings: %#v", rule, findings)
	return model.Finding{}
}

func assertRuleForSummary(t *testing.T, findings []model.Finding, rule, needle string) model.Finding {
	t.Helper()
	for _, finding := range findings {
		if finding.RuleID == rule && strings.Contains(finding.Summary, needle) {
			return finding
		}
	}
	t.Fatalf("missing rule %s containing %q in findings: %#v", rule, needle, findings)
	return model.Finding{}
}
