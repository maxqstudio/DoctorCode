package goanalysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestRelatedLocationsFindPackageAndTestReferencesWithoutShadowNoise(t *testing.T) {
	root := t.TempDir()
	source := "package demo\n\nfunc target(flag bool) bool {\n\tif flag {\n\t\treturn true\n\t} else {\n\t\treturn false\n\t}\n}\n\nfunc caller() bool { return target(true) }\n\nfunc shadow() {\n\ttarget := func() {}\n\ttarget()\n}\n"
	testSource := "package demo\n\nimport \"testing\"\n\nfunc TestTarget(t *testing.T) {\n\t_ = target(false)\n}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}

	finding := model.Finding{Path: "sample.go", LineStart: 4, LineEnd: 8}
	got, err := RelatedLocations(root, finding)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected exactly package + test references, got %#v", got)
	}
	if got[0].Path != "sample.go" || got[0].Line != 11 || got[0].Kind != "reference" {
		t.Fatalf("unexpected production reference: %#v", got[0])
	}
	if got[1].Path != "sample_test.go" || got[1].Line != 6 || got[1].Kind != "test_reference" {
		t.Fatalf("unexpected test reference: %#v", got[1])
	}
}

func TestRelatedLocationsIgnoreExternalTestPackage(t *testing.T) {
	root := t.TempDir()
	source := "package demo\n\nfunc target() bool { return true }\n"
	externalTest := "package demo_test\n\nfunc TestTarget() { target() }\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "external_test.go"), []byte(externalTest), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := RelatedLocations(root, model.Finding{Path: "sample.go", LineStart: 3, LineEnd: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("external test package must not be guessed as a related binding: %#v", got)
	}
}
