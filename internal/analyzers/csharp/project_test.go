package csharpanalysis

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverProjectScopeRepresentsManifestPresenceOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Demo.sln"), []byte("not evaluated as a solution"), 0o600); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(root, "src", "App")
	if err := os.MkdirAll(projectDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "App.csproj"), []byte("<Project><intentionally-unclosed>"), 0o600); err != nil {
		t.Fatal(err)
	}
	ignoredDir := filepath.Join(root, "obj")
	if err := os.MkdirAll(ignoredDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignoredDir, "Ignored.csproj"), []byte("<Project />"), 0o600); err != nil {
		t.Fatal(err)
	}

	scope, err := discoverProjectScope(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scope.projects, []string{"src/App/App.csproj"}) {
		t.Fatalf("projects=%v", scope.projects)
	}
	if !reflect.DeepEqual(scope.solutions, []string{"Demo.sln"}) {
		t.Fatalf("solutions=%v", scope.solutions)
	}
	wantEvidence := []string{
		"project_manifest=src/App/App.csproj presence_only=true",
		"solution_manifests=1 presence_only=true",
	}
	if got := scope.evidenceFor("src/App/Feature.cs"); !reflect.DeepEqual(got, wantEvidence) {
		t.Fatalf("evidence=%v want=%v", got, wantEvidence)
	}
}

func TestProjectScopeDoesNotInferUnrelatedProjectMembership(t *testing.T) {
	scope := projectScope{
		projects:  []string{"src/App/App.csproj"},
		solutions: []string{"Demo.sln"},
	}
	want := []string{"solution_manifests=1 presence_only=true"}
	if got := scope.evidenceFor("tools/Worker.cs"); !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence=%v want=%v", got, want)
	}
}
