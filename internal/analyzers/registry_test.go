package analyzers

import "testing"

func TestDefaultAnalyzerContractsValidate(t *testing.T) {
	if err := ValidateDefault(); err != nil {
		t.Fatal(err)
	}
	descriptors := Descriptors()
	if len(descriptors) != 3 {
		t.Fatalf("descriptors=%d want=3", len(descriptors))
	}
	want := []string{"go/builtin-v1", "python/stdlib-ast-v1", "javascript/parser-backed-v2"}
	for i, desc := range descriptors {
		if desc.ID != want[i] {
			t.Fatalf("descriptor[%d]=%q want=%q", i, desc.ID, want[i])
		}
	}
}
