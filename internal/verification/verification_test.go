package verification

import (
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestVerifyPassesWhenSemanticTargetCountDropsAfterLineShift(t *testing.T) {
	baseline := model.AuditResult{
		Analyzers: []string{"go/stdlib-ast-v1"},
		Findings: []model.Finding{
			{ID: "OLD", RuleID: "GO-SIMPLIFY-BOOL-RETURN", Path: "sample.go", LineStart: 10, Summary: "boolean if/else returns opposite literals and can be represented directly", Severity: model.SeverityLow},
			{ID: "OTHER", RuleID: "GO-SIMPLIFY-BOOL-RETURN", Path: "sample.go", LineStart: 30, Summary: "boolean if/else returns opposite literals and can be represented directly", Severity: model.SeverityLow},
		},
	}
	contract, err := BuildContract(baseline, "OLD")
	if err != nil {
		t.Fatal(err)
	}
	if contract.TargetBaselineCount != 2 {
		t.Fatalf("expected semantic-key baseline count 2, got %d", contract.TargetBaselineCount)
	}

	current := model.AuditResult{
		Analyzers: []string{"go/stdlib-ast-v1"},
		Findings: []model.Finding{
			{ID: "SHIFTED", RuleID: "GO-SIMPLIFY-BOOL-RETURN", Path: "sample.go", LineStart: 35, Summary: "boolean if/else returns opposite literals and can be represented directly", Severity: model.SeverityLow},
		},
	}
	result, err := Verify(contract, current)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || !result.TargetResolved {
		t.Fatalf("expected semantic target count decrease to pass: %#v", result)
	}
	if result.TargetCurrentCount != 1 {
		t.Fatalf("expected one matching finding after repair, got %d", result.TargetCurrentCount)
	}
}

func TestVerifyFailsOnNewSameOrHigherSeverityFindingInTargetPath(t *testing.T) {
	baseline := model.AuditResult{
		Analyzers: []string{"python/stdlib-ast-v1"},
		Findings: []model.Finding{
			{ID: "TARGET", RuleID: "PY-SIMPLIFY-BOOL-RETURN", Path: "module.py", LineStart: 4, Summary: "boolean if/else returns opposite literals and can be represented directly", Severity: model.SeverityLow},
		},
	}
	contract, err := BuildContract(baseline, "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	current := model.AuditResult{
		Analyzers: []string{"python/stdlib-ast-v1"},
		Findings: []model.Finding{
			{ID: "NEW", RuleID: "PY-SEC-HARDCODED-CREDENTIAL", Path: "module.py", LineStart: 2, Summary: "credential-like identifier api_token is assigned a hardcoded string literal", Severity: model.SeverityHigh},
		},
	}
	result, err := Verify(contract, current)
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed {
		t.Fatalf("new higher-severity finding must block PASS: %#v", result)
	}
	if !result.TargetResolved || len(result.NewBlockingFindings) != 1 {
		t.Fatalf("expected resolved target plus one blocking regression: %#v", result)
	}
}

func TestVerifyFailsClosedWhenAnalyzerSetChanges(t *testing.T) {
	baseline := model.AuditResult{
		Analyzers: []string{"go/stdlib-ast-v1", "python/stdlib-ast-v1"},
		Findings: []model.Finding{
			{ID: "TARGET", RuleID: "GO-DEADCODE-ZERO-REF", Path: "sample.go", LineStart: 3, Summary: "unexported package-level function stale has no lexical references in its package", Severity: model.SeverityMedium},
		},
	}
	contract, err := BuildContract(baseline, "TARGET")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Verify(contract, model.AuditResult{Analyzers: []string{"go/stdlib-ast-v1"}})
	if err == nil {
		t.Fatal("analyzer-set drift must fail closed")
	}
}

func TestBuildContractRejectsMissingFinding(t *testing.T) {
	_, err := BuildContract(model.AuditResult{Analyzers: []string{"go/stdlib-ast-v1"}}, "MISSING")
	if err == nil {
		t.Fatal("missing finding id must fail")
	}
}
