package analyzers

import "testing"

func TestDefaultAnalyzerContractsValidate(t *testing.T) {
	if err := ValidateDefault(); err != nil {
		t.Fatal(err)
	}
	descriptors := Descriptors()
	if len(descriptors) != 5 {
		t.Fatalf("descriptors=%d want=5", len(descriptors))
	}
	want := []string{"go/builtin-v1", "python/stdlib-ast-v1", "javascript/parser-backed-v2", "rust/gotreesitter-v1", "jvm/gotreesitter-v1"}
	for i, desc := range descriptors {
		if desc.ID != want[i] {
			t.Fatalf("descriptor[%d]=%q want=%q", i, desc.ID, want[i])
		}
	}
}
