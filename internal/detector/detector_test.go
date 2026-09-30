package detector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func fixtureDescriptor() Descriptor {
	return Descriptor{
		ID:         "fixture/v1",
		Language:   "Fixture",
		Extensions: []string{".fx"},
		Parser: ParserContract{
			Kind:       ParserBuiltinAST,
			Provider:   "fixture/parser",
			FailClosed: true,
		},
		Availability:     AvailabilityContract{Mode: AvailabilitySourceOnly},
		EvidenceBoundary: "fixture-only bounded evidence",
		Rules: []RuleMetadata{{
			ID:          "FX-LOGIC",
			Category:    model.CategoryLogic,
			SafeAutofix: false,
		}},
		Benchmark: BenchmarkContract{SchemaVersion: 1, Languages: []string{"Fixture"}},
	}
}

func TestDescriptorValidationAndRecognition(t *testing.T) {
	desc := fixtureDescriptor()
	if err := ValidateDescriptor(desc); err != nil {
		t.Fatal(err)
	}
	if !RecognizesPath(desc, "sample.FX") || RecognizesPath(desc, "sample.go") {
		t.Fatal("descriptor path recognition drift")
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "ignored.fx"), []byte("ignored"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := RecognizesRoot(context.Background(), root, desc)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("ignored directory must not make analyzer available")
	}
	if err := os.WriteFile(filepath.Join(root, "live.fx"), []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err = RecognizesRoot(context.Background(), root, desc)
	if err != nil || !ok {
		t.Fatalf("recognized source not detected: ok=%v err=%v", ok, err)
	}
}

func TestFindingIDCompatibilityAndValidation(t *testing.T) {
	if got, want := FindingID("RULE", "path/file.go", 7, "summary"), "RULE-080699986d"; got != want {
		t.Fatalf("FindingID()=%q want=%q", got, want)
	}

	desc := fixtureDescriptor()
	finding := model.Finding{
		ID:          FindingID("FX-LOGIC", "sample.fx", 3, "bounded"),
		RuleID:      "FX-LOGIC",
		Category:    model.CategoryLogic,
		Path:        "sample.fx",
		LineStart:   3,
		Summary:     "bounded",
		SafeAutofix: false,
	}
	if err := ValidateFindings(desc, []model.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	finding.RuleID = "FX-UNDECLARED"
	if err := ValidateFindings(desc, []model.Finding{finding}); err == nil {
		t.Fatal("undeclared rule must fail closed")
	}
}
