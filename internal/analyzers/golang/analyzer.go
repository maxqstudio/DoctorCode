package goanalysis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

const (
	ruleDeadCode = "GO-DEADCODE-ZERO-REF"
	ruleLogic    = "GO-LOGIC-DUPLICATE-CONDITION"
	ruleSimplify = "GO-SIMPLIFY-BOOL-RETURN"
	ruleSecurity = "GO-SEC-HARDCODED-CREDENTIAL"
	ruleBloat    = "GO-BLOAT-PASSTHROUGH-WRAPPER"
)

type Analyzer struct{}

func New() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) Name() string {
	return "go/builtin-v1"
}

type parsedFile struct {
	abs    string
	rel    string
	dir    string
	pkg    string
	isTest bool
	file   *ast.File
}

type packageInfo struct {
	name  string
	dir   string
	files []*parsedFile
	risky bool
}

type candidate struct {
	name string
	decl *ast.FuncDecl
	file *parsedFile
}

func (a *Analyzer) Analyze(ctx context.Context, root string) ([]model.Finding, error) {
	fset := token.NewFileSet()
	files, asmDirs, err := parseGoFiles(ctx, root, fset)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	packages := map[string]*packageInfo{}
	for _, file := range files {
		key := file.dir + "\x00" + file.pkg
		pkg := packages[key]
		if pkg == nil {
			pkg = &packageInfo{name: file.pkg, dir: file.dir}
			packages[key] = pkg
		}
		pkg.files = append(pkg.files, file)
		if asmDirs[file.dir] || hasEscapeHatch(file.file) {
			pkg.risky = true
		}
	}

	var findings []model.Finding
	for _, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		findings = append(findings, deadCodeFindings(fset, pkg)...)
		findings = append(findings, bloatFindings(fset, pkg)...)
		for _, file := range pkg.files {
			if file.isTest {
				continue
			}
			findings = append(findings, logicFindings(fset, file)...)
			findings = append(findings, simplifyFindings(fset, file)...)
			findings = append(findings, securityFindings(fset, file)...)
		}
	}
	return findings, nil
}

func parseGoFiles(ctx context.Context, root string, fset *token.FileSet) ([]*parsedFile, map[string]bool, error) {
	var files []*parsedFile
	asmDirs := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".s" {
			asmDirs[filepath.Dir(path)] = true
			return nil
		}
		if ext != ".go" {
			return nil
		}

		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, &parsedFile{
			abs:    path,
			rel:    filepath.ToSlash(rel),
			dir:    filepath.Dir(path),
			pkg:    node.Name.Name,
			isTest: strings.HasSuffix(strings.ToLower(path), "_test.go"),
			file:   node,
		})
		return nil
	})
	return files, asmDirs, err
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", ".venv", "venv", "__pycache__":
		return true
	default:
		return false
	}
}

func hasEscapeHatch(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path != nil && imp.Path.Value == `"C"` {
			return true
		}
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			text := strings.TrimSpace(comment.Text)
			if strings.Contains(text, "//go:linkname") || strings.HasPrefix(text, "//export ") {
				return true
			}
		}
	}
	return false
}

func topLevelCandidates(pkg *packageInfo) []candidate {
	var out []candidate
	for _, file := range pkg.files {
		if file.isTest {
			continue
		}
		for _, decl := range file.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name == nil {
				continue
			}
			name := fn.Name.Name
			if ast.IsExported(name) || name == "init" || (pkg.name == "main" && name == "main") {
				continue
			}
			out = append(out, candidate{name: name, decl: fn, file: file})
		}
	}
	return out
}

func lexicalReferenceCounts(pkg *packageInfo, candidates []candidate) map[string]int {
	definitionPos := map[token.Pos]bool{}
	names := map[string]bool{}
	for _, item := range candidates {
		definitionPos[item.decl.Name.Pos()] = true
		names[item.name] = true
	}

	counts := map[string]int{}
	for _, file := range pkg.files {
		ast.Inspect(file.file, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok || !names[id.Name] || definitionPos[id.Pos()] {
				return true
			}
			counts[id.Name]++
			return true
		})
	}
	return counts
}

func deadCodeFindings(fset *token.FileSet, pkg *packageInfo) []model.Finding {
	if pkg.risky {
		return nil
	}
	candidates := topLevelCandidates(pkg)
	counts := lexicalReferenceCounts(pkg, candidates)
	var out []model.Finding
	for _, item := range candidates {
		if counts[item.name] != 0 {
			continue
		}
		line := fset.Position(item.decl.Name.Pos()).Line
		out = append(out, newFinding(
			ruleDeadCode,
			model.CategoryDeadCode,
			model.SeverityMedium,
			model.ConfidenceHigh,
			item.file.rel,
			line,
			line,
			fmt.Sprintf("unexported package-level function %s has no lexical references in its package", item.name),
			[]string{
				"0 identifier references found across production and test files in the same package",
				"package has no assembly, cgo import, //go:linkname, or //export escape hatch detected",
				"HIGH is intentionally below PROVEN because external build tooling and generated linkage can exist outside the visible tree",
			},
		))
	}
	return out
}

func bloatFindings(fset *token.FileSet, pkg *packageInfo) []model.Finding {
	candidates := topLevelCandidates(pkg)
	counts := lexicalReferenceCounts(pkg, candidates)
	var out []model.Finding
	for _, item := range candidates {
		if counts[item.name] != 1 || item.decl.Body == nil || len(item.decl.Body.List) != 1 {
			continue
		}
		ret, ok := item.decl.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		call, ok := ret.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		params, ok := parameterNames(item.decl.Type.Params)
		if !ok || len(params) != len(call.Args) || !argsForwardExactly(params, call.Args) {
			continue
		}
		if id, ok := call.Fun.(*ast.Ident); ok && id.Name == item.name {
			continue
		}
		line := fset.Position(item.decl.Name.Pos()).Line
		out = append(out, newFinding(
			ruleBloat,
			model.CategoryBloat,
			model.SeverityLow,
			model.ConfidenceSuspicious,
			item.file.rel,
			line,
			fset.Position(item.decl.End()).Line,
			fmt.Sprintf("unexported one-call pass-through wrapper %s has one lexical reference", item.name),
			[]string{
				"function body is a single return of another call",
				"all named parameters are forwarded unchanged and in order",
				"wrapper has exactly one lexical reference in the package",
				"SUSPICIOUS only: naming, policy boundaries, instrumentation, or future API intent may justify the wrapper",
			},
		))
	}
	return out
}

func parameterNames(fields *ast.FieldList) ([]string, bool) {
	if fields == nil {
		return nil, true
	}
	var names []string
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			return nil, false
		}
		for _, name := range field.Names {
			names = append(names, name.Name)
		}
	}
	return names, true
}

func argsForwardExactly(params []string, args []ast.Expr) bool {
	for i, arg := range args {
		id, ok := arg.(*ast.Ident)
		if !ok || id.Name != params[i] {
			return false
		}
	}
	return true
}

func logicFindings(fset *token.FileSet, file *parsedFile) []model.Finding {
	var out []model.Finding
	ast.Inspect(file.file, func(node ast.Node) bool {
		start, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		seen := map[string]int{}
		for current := start; current != nil; {
			if pureCondition(current.Cond) {
				key := expressionText(fset, current.Cond)
				line := fset.Position(current.Cond.Pos()).Line
				if earlier, exists := seen[key]; exists {
					out = append(out, newFinding(
						ruleLogic,
						model.CategoryLogic,
						model.SeverityMedium,
						model.ConfidenceHigh,
						file.rel,
						line,
						line,
						"duplicate side-effect-free condition appears later in the same if/else-if chain",
						[]string{
							fmt.Sprintf("same normalized condition already appeared at line %d", earlier),
							"condition is restricted to literals, identifiers, selectors, unary expressions, and binary expressions without calls",
						},
					))
					break
				}
				seen[key] = line
			}
			next, ok := current.Else.(*ast.IfStmt)
			if !ok {
				break
			}
			current = next
		}
		return true
	})
	return out
}

func pureCondition(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return pureCondition(value.X)
	case *ast.SelectorExpr:
		return pureCondition(value.X)
	case *ast.UnaryExpr:
		return pureCondition(value.X)
	case *ast.BinaryExpr:
		return pureCondition(value.X) && pureCondition(value.Y)
	default:
		return false
	}
}

func expressionText(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func simplifyFindings(fset *token.FileSet, file *parsedFile) []model.Finding {
	var out []model.Finding
	ast.Inspect(file.file, func(node ast.Node) bool {
		stmt, ok := node.(*ast.IfStmt)
		if !ok || stmt.Else == nil || len(stmt.Body.List) != 1 {
			return true
		}
		elseBlock, ok := stmt.Else.(*ast.BlockStmt)
		if !ok || len(elseBlock.List) != 1 {
			return true
		}
		left, leftOK := boolReturn(stmt.Body.List[0])
		right, rightOK := boolReturn(elseBlock.List[0])
		if !leftOK || !rightOK || left == right {
			return true
		}
		line := fset.Position(stmt.If).Line
		replacement := "return condition"
		if !left {
			replacement = "return !condition"
		}
		out = append(out, newFinding(
			ruleSimplify,
			model.CategorySimplify,
			model.SeverityLow,
			model.ConfidenceProven,
			file.rel,
			line,
			fset.Position(stmt.End()).Line,
			"boolean if/else returns opposite literals and can be represented directly",
			[]string{
				"both branches contain exactly one return statement",
				"branches return opposite boolean literals",
				"behavior-preserving shape: " + replacement,
			},
		))
		return true
	})
	return out
}

func boolReturn(stmt ast.Stmt) (bool, bool) {
	ret, ok := stmt.(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false, false
	}
	lit, ok := ret.Results[0].(*ast.Ident)
	if !ok {
		return false, false
	}
	switch lit.Name {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func securityFindings(fset *token.FileSet, file *parsedFile) []model.Finding {
	if file.isTest || strings.Contains("/"+file.rel, "/testdata/") {
		return nil
	}
	var out []model.Finding
	ast.Inspect(file.file, func(node ast.Node) bool {
		switch value := node.(type) {
		case *ast.ValueSpec:
			for i, name := range value.Names {
				if i >= len(value.Values) {
					continue
				}
				out = append(out, credentialFinding(fset, file.rel, name.Name, value.Values[i])...)
			}
		case *ast.AssignStmt:
			for i, lhs := range value.Lhs {
				if i >= len(value.Rhs) {
					continue
				}
				id, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}
				out = append(out, credentialFinding(fset, file.rel, id.Name, value.Rhs[i])...)
			}
		}
		return true
	})
	return out
}

func credentialFinding(fset *token.FileSet, path, name string, expr ast.Expr) []model.Finding {
	if !credentialLike(name) {
		return nil
	}
	value, ok := stringLiteral(expr)
	if !ok || len(value) < 8 || obviousPlaceholder(value) {
		return nil
	}
	sum := sha256.Sum256([]byte(value))
	line := fset.Position(expr.Pos()).Line
	return []model.Finding{newFinding(
		ruleSecurity,
		model.CategorySecurity,
		model.SeverityHigh,
		model.ConfidenceSuspicious,
		path,
		line,
		line,
		fmt.Sprintf("credential-like identifier %s is assigned a hardcoded string literal", name),
		[]string{
			fmt.Sprintf("literal value redacted; byte_length=%d; sha256_prefix=%x", len(value), sum[:4]),
			"SUSPICIOUS only: the literal may be a non-secret fixture or placeholder not recognized by the conservative filter",
		},
	)}
}

func credentialLike(name string) bool {
	var normalized strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			normalized.WriteRune(r)
		}
	}
	value := normalized.String()
	for _, suffix := range []string{"password", "passwd", "secret", "apikey", "token", "privatekey", "accesskey"} {
		if value == suffix || strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func obviousPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "$"+"{") || strings.HasPrefix(lower, "$(") {
		return true
	}
	for _, marker := range []string{"example", "dummy", "placeholder", "changeme", "change-me", "notasecret", "not-a-secret", "your_", "your-"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func newFinding(rule string, category model.Category, severity model.Severity, confidence model.Confidence, path string, lineStart, lineEnd int, summary string, evidence []string) model.Finding {
	key := fmt.Sprintf("%s|%s|%d|%s", rule, path, lineStart, summary)
	digest := sha256.Sum256([]byte(key))
	return model.Finding{
		ID:           fmt.Sprintf("%s-%x", rule, digest[:5]),
		RuleID:       rule,
		Category:     category,
		Severity:     severity,
		Confidence:   confidence,
		Path:         path,
		LineStart:    lineStart,
		LineEnd:      lineEnd,
		Summary:      summary,
		Evidence:     evidence,
		Verification: []string{"go test ./...", "go vet ./..."},
		SafeAutofix:  false,
	}
}
