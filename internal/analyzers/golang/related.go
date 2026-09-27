package goanalysis

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

type RelatedLocation struct {
	Path string
	Line int
	Kind string
}

type relatedParsedFile struct {
	path   string
	rel    string
	isTest bool
	file   *ast.File
}

func RelatedLocations(root string, finding model.Finding) ([]RelatedLocation, error) {
	if strings.ToLower(filepath.Ext(finding.Path)) != ".go" {
		return nil, nil
	}

	sourcePath := filepath.Join(root, filepath.FromSlash(finding.Path))
	sourceInfo, err := os.Lstat(sourcePath)
	if err != nil {
		return nil, err
	}
	if sourceInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, nil
	}

	fset := token.NewFileSet()
	sourceFile, err := parser.ParseFile(fset, sourcePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", finding.Path, err)
	}

	selected := enclosingTopLevelFunction(fset, sourceFile, finding.LineStart, finding.LineEnd)
	if selected == nil || selected.Name == nil || selected.Name.Obj == nil {
		return nil, nil
	}
	name := selected.Name.Name
	selectedObject := selected.Name.Obj
	definitionPos := selected.Name.Pos()
	sourceClean := filepath.Clean(sourcePath)

	entries, err := os.ReadDir(filepath.Dir(sourcePath))
	if err != nil {
		return nil, err
	}

	var files []relatedParsedFile
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&fs.ModeSymlink != 0 || strings.ToLower(filepath.Ext(entry.Name())) != ".go" {
			continue
		}
		path := filepath.Join(filepath.Dir(sourcePath), entry.Name())
		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		if node.Name == nil || node.Name.Name != sourceFile.Name.Name {
			continue
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil, relErr
		}
		files = append(files, relatedParsedFile{
			path:   filepath.Clean(path),
			rel:    filepath.ToSlash(rel),
			isTest: strings.HasSuffix(strings.ToLower(entry.Name()), "_test.go"),
			file:   node,
		})
	}

	var out []RelatedLocation
	seen := map[string]bool{}
	for _, file := range files {
		nonPackageRef := map[token.Pos]bool{}
		ast.Inspect(file.file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.FuncDecl:
				if value.Name != nil {
					nonPackageRef[value.Name.Pos()] = true
				}
			case *ast.SelectorExpr:
				nonPackageRef[value.Sel.Pos()] = true
			case *ast.KeyValueExpr:
				if id, ok := value.Key.(*ast.Ident); ok {
					nonPackageRef[id.Pos()] = true
				}
			}
			return true
		})

		ast.Inspect(file.file, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok || id.Name != name || id.Pos() == definitionPos || nonPackageRef[id.Pos()] {
				return true
			}

			isSelectedReference := false
			if id.Obj != nil {
				isSelectedReference = file.path == sourceClean && id.Obj == selectedObject
			} else {
				isSelectedReference = file.path != sourceClean
			}
			if !isSelectedReference {
				return true
			}

			line := fset.Position(id.Pos()).Line
			kind := "reference"
			if file.isTest {
				kind = "test_reference"
			}
			key := fmt.Sprintf("%s:%d:%s", file.rel, line, kind)
			if !seen[key] {
				seen[key] = true
				out = append(out, RelatedLocation{Path: file.rel, Line: line, Kind: kind})
			}
			return true
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Kind < out[j].Kind
	})
	return out, nil
}

func enclosingTopLevelFunction(fset *token.FileSet, file *ast.File, lineStart, lineEnd int) *ast.FuncDecl {
	if lineStart < 1 {
		return nil
	}
	if lineEnd < lineStart {
		lineEnd = lineStart
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Body == nil {
			continue
		}
		start := fset.Position(fn.Pos()).Line
		end := fset.Position(fn.End()).Line
		if lineStart >= start && lineEnd <= end {
			return fn
		}
	}
	return nil
}
