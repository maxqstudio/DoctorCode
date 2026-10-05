package csharpanalysis

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type projectScope struct {
	projects  []string
	solutions []string
}

// discoverProjectScope represents only visible .csproj/.sln manifest presence.
// It intentionally does not parse or evaluate MSBuild, NuGet, project references,
// solution membership, source generators, target frameworks, or conditional items.
func discoverProjectScope(ctx context.Context, root string) (projectScope, error) {
	var scope projectScope
	err := filepath.WalkDir(root, func(candidate string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if candidate != root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(candidate))
		if ext != ".csproj" && ext != ".sln" {
			return nil
		}
		rel, err := filepath.Rel(root, candidate)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if ext == ".csproj" {
			scope.projects = append(scope.projects, rel)
		} else {
			scope.solutions = append(scope.solutions, rel)
		}
		return nil
	})
	if err != nil {
		return projectScope{}, fmt.Errorf("discover C# project scope: %w", err)
	}
	sort.Strings(scope.projects)
	sort.Strings(scope.solutions)
	return scope, nil
}

func (scope projectScope) evidenceFor(sourceRel string) []string {
	sourceDir := path.Dir(filepath.ToSlash(sourceRel))
	bestProject := ""
	bestDirLen := -1
	for _, projectFile := range scope.projects {
		projectDir := path.Dir(projectFile)
		if !withinProjectDir(sourceDir, projectDir) || len(projectDir) <= bestDirLen {
			continue
		}
		bestProject = projectFile
		bestDirLen = len(projectDir)
	}

	var evidence []string
	if bestProject != "" {
		evidence = append(evidence, "project_manifest="+bestProject+" presence_only=true")
	}
	if len(scope.solutions) > 0 {
		evidence = append(evidence, fmt.Sprintf("solution_manifests=%d presence_only=true", len(scope.solutions)))
	}
	return evidence
}

func withinProjectDir(child, parent string) bool {
	if parent == "." || parent == "" {
		return true
	}
	return child == parent || strings.HasPrefix(child, parent+"/")
}
