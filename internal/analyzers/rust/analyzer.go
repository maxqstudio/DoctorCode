package rustanalysis

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

const (
	ruleDeadCode = "RS-DEADCODE-PRIVATE-ZERO-REF"
	ruleLogic    = "RS-LOGIC-DUPLICATE-CONDITION"
	ruleSecurity = "RS-SEC-HARDCODED-CREDENTIAL"
	ruleSimplify = "RS-SIMPLIFY-BOOL-RETURN"
)

type Analyzer struct{}

type parsedFile struct {
	rel       string
	source    []byte
	document  *syntaxDocument
	generated bool
}

type deadCodeCandidate struct {
	nameNode *gotreesitter.Node
	name     string
	file     *parsedFile
	node     *gotreesitter.Node
}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "rust/gotreesitter-v1" }

func (a *Analyzer) Descriptor() detector.Descriptor {
	return detector.Descriptor{
		ID:         a.Name(),
		Language:   "Rust",
		Extensions: []string{".rs"},
		Parser: detector.ParserContract{
			Kind:       detector.ParserEmbeddedAST,
			Provider:   "gotreesitter v0.55.1 Rust",
			FailClosed: true,
		},
		Availability: detector.AvailabilityContract{Mode: detector.AvailabilitySourceOnly},
		EvidenceBoundary: "Pure-Go gotreesitter Rust syntax/AST authority. M22 is bounded to direct literal SECURITY, opposite-boolean SIMPLIFY, duplicate bool-parameter LOGIC, and conservative top-level private zero-reference DEADCODE. Macros, attributes, public/exported functions, modified/extern functions, methods, nested modules, and unresolved dynamic linkage fail closed for DEADCODE. BLOAT remains unavailable.",
		Rules: []detector.RuleMetadata{
			{ID: ruleDeadCode, Category: model.CategoryDeadCode, SafeAutofix: false},
			{ID: ruleLogic, Category: model.CategoryLogic, SafeAutofix: false},
			{ID: ruleSecurity, Category: model.CategorySecurity, SafeAutofix: false},
			{ID: ruleSimplify, Category: model.CategorySimplify, SafeAutofix: false},
		},
		Benchmark: detector.BenchmarkContract{SchemaVersion: 1, Languages: []string{"Rust"}},
	}
}

func (a *Analyzer) Analyze(ctx context.Context, root string) ([]model.Finding, error) {
	files, err := parseRustFiles(ctx, root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w: no Rust source files", detector.ErrUnavailable)
	}

	var findings []model.Finding
	for _, file := range files {
		findings = append(findings, simplifyFindings(file)...)
		findings = append(findings, securityFindings(file)...)
		findings = append(findings, logicFindings(file)...)
	}
	findings = append(findings, deadCodeFindings(files)...)
	return findings, nil
}

func parseRustFiles(ctx context.Context, root string) ([]*parsedFile, error) {
	var paths []string
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
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".rs") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	files := make([]*parsedFile, 0, len(paths))
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read Rust source %s: %w", path, err)
		}
		if strings.IndexByte(string(source), 0) >= 0 {
			return nil, fmt.Errorf("rust parse incomplete: %s contains NUL byte", path)
		}
		document, err := parseSyntax(source)
		if err != nil {
			return nil, fmt.Errorf("rust syntax validation failed for %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		files = append(files, &parsedFile{
			rel:       filepath.ToSlash(rel),
			source:    source,
			document:  document,
			generated: generatedRustSource(source),
		})
	}
	return files, nil
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".idea", ".vscode", "target", "vendor", "node_modules", "dist", "build", "coverage", "testdata":
		return true
	default:
		return false
	}
}

func generatedRustSource(source []byte) bool {
	limit := len(source)
	if limit > 1024 {
		limit = 1024
	}
	prefix := strings.ToLower(string(source[:limit]))
	return strings.Contains(prefix, "code generated") || strings.Contains(prefix, "@generated")
}

func simplifyFindings(file *parsedFile) []model.Finding {
	var out []model.Finding
	lang := file.document.language
	gotreesitter.Walk(file.document.tree.RootNode(), func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		if node.Type(lang) != "if_expression" {
			return gotreesitter.WalkContinue
		}
		consequence := node.ChildByFieldName("consequence", lang)
		alternative := node.ChildByFieldName("alternative", lang)
		if consequence == nil || alternative == nil || alternative.Type(lang) != "else_clause" {
			return gotreesitter.WalkContinue
		}
		elseBlock := directNamedChildOfType(alternative, lang, "block")
		if elseBlock == nil {
			return gotreesitter.WalkContinue
		}
		left, leftOK := blockBoolean(file.source, consequence, lang)
		right, rightOK := blockBoolean(file.source, elseBlock, lang)
		if !leftOK || !rightOK || left == right {
			return gotreesitter.WalkContinue
		}
		line := 1 + int(node.StartPoint().Row)
		out = append(out, newFinding(
			ruleSimplify,
			model.CategorySimplify,
			model.SeverityLow,
			model.ConfidenceHigh,
			file.rel,
			line,
			"opposite Rust boolean branches can be reduced to the condition or its negation",
			[]string{
				"AST-proven if/else branches each contain exactly one opposite boolean result",
				"safe_autofix remains disabled",
			},
		))
		return gotreesitter.WalkSkipChildren
	})
	return out
}

func blockBoolean(source []byte, block *gotreesitter.Node, lang *gotreesitter.Language) (bool, bool) {
	if block == nil || block.NamedChildCount() != 1 {
		return false, false
	}
	node := block.NamedChild(0)
	if node == nil {
		return false, false
	}
	if node.Type(lang) == "boolean_literal" {
		return booleanNode(source, node)
	}
	if node.Type(lang) != "expression_statement" || node.NamedChildCount() != 1 {
		return false, false
	}
	ret := node.NamedChild(0)
	if ret == nil || ret.Type(lang) != "return_expression" || ret.NamedChildCount() != 1 {
		return false, false
	}
	return booleanNode(source, ret.NamedChild(0))
}

func booleanNode(source []byte, node *gotreesitter.Node) (bool, bool) {
	switch nodeText(source, node) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

func securityFindings(file *parsedFile) []model.Finding {
	var out []model.Finding
	lang := file.document.language
	gotreesitter.Walk(file.document.tree.RootNode(), func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		var nameNode, valueNode *gotreesitter.Node
		switch node.Type(lang) {
		case "const_item", "static_item":
			nameNode = node.ChildByFieldName("name", lang)
			valueNode = node.ChildByFieldName("value", lang)
		case "let_declaration":
			pattern := node.ChildByFieldName("pattern", lang)
			if pattern != nil && pattern.Type(lang) == "identifier" {
				nameNode = pattern
				valueNode = node.ChildByFieldName("value", lang)
			}
		}
		if nameNode == nil || valueNode == nil {
			return gotreesitter.WalkContinue
		}
		name := nodeText(file.source, nameNode)
		if !credentialLike(name) {
			return gotreesitter.WalkContinue
		}
		value, ok := rustStringValue(file.source, valueNode, lang)
		if !ok || obviousPlaceholder(value) {
			return gotreesitter.WalkContinue
		}
		digest := sha256.Sum256([]byte(value))
		line := 1 + int(node.StartPoint().Row)
		out = append(out, newFinding(
			ruleSecurity,
			model.CategorySecurity,
			model.SeverityHigh,
			model.ConfidenceSuspicious,
			file.rel,
			line,
			"credential-like Rust binding is assigned a hardcoded string literal",
			[]string{
				fmt.Sprintf("identifier=%s literal_bytes=%d sha256_prefix=%x", name, len(value), digest[:6]),
				"literal value is redacted",
			},
		))
		return gotreesitter.WalkSkipChildren
	})
	return out
}

func rustStringValue(source []byte, node *gotreesitter.Node, lang *gotreesitter.Language) (string, bool) {
	if node == nil {
		return "", false
	}
	switch node.Type(lang) {
	case "string_literal":
		text := nodeText(source, node)
		if strings.HasPrefix(text, "\"") && strings.HasSuffix(text, "\"") {
			value, err := strconv.Unquote(text)
			if err == nil {
				return value, true
			}
		}
	case "raw_string_literal":
		text := nodeText(source, node)
		firstQuote := strings.IndexByte(text, '"')
		lastQuote := strings.LastIndexByte(text, '"')
		if strings.HasPrefix(text, "r") && firstQuote >= 1 && lastQuote > firstQuote {
			hashes := text[1:firstQuote]
			if strings.Trim(hashes, "#") == "" && text[lastQuote+1:] == hashes {
				return text[firstQuote+1 : lastQuote], true
			}
		}
	}
	return "", false
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

func obviousPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" || strings.HasPrefix(lower, "$"+"{") || strings.HasPrefix(lower, "$(") {
		return true
	}
	for _, marker := range []string{"example", "dummy", "placeholder", "changeme", "change-me", "notasecret", "not-a-secret", "your_", "your-", "<"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func logicFindings(file *parsedFile) []model.Finding {
	var out []model.Finding
	lang := file.document.language
	gotreesitter.Walk(file.document.tree.RootNode(), func(fn *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		if fn.Type(lang) != "function_item" {
			return gotreesitter.WalkContinue
		}
		params := rustBoolParameters(file.source, fn, lang)
		if len(params) == 0 {
			return gotreesitter.WalkSkipChildren
		}
		body := fn.ChildByFieldName("body", lang)
		if body == nil {
			return gotreesitter.WalkSkipChildren
		}
		continuations := map[uint32]bool{}
		gotreesitter.Walk(body, func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
			if node != body && node.Type(lang) == "function_item" {
				return gotreesitter.WalkSkipChildren
			}
			if node.Type(lang) != "if_expression" || continuations[node.StartByte()] {
				return gotreesitter.WalkContinue
			}
			chain := rustIfChain(node, lang)
			for i := 1; i < len(chain); i++ {
				continuations[chain[i].StartByte()] = true
			}
			seen := map[string]int{}
			for _, item := range chain {
				condition := item.ChildByFieldName("condition", lang)
				key, ok := rustBoolCondition(file.source, condition, lang, params)
				if !ok {
					seen = map[string]int{}
					continue
				}
				line := 1 + int(item.StartPoint().Row)
				if earlier, exists := seen[key]; exists {
					out = append(out, newFinding(
						ruleLogic,
						model.CategoryLogic,
						model.SeverityMedium,
						model.ConfidenceHigh,
						file.rel,
						line,
						"duplicate side-effect-free Rust bool-parameter condition appears later in the same if/else-if chain",
						[]string{
							fmt.Sprintf("AST-proven same bool-parameter condition already appeared at line %d", earlier),
							"proof subset is limited to bare or negated bool parameters of the nearest function item",
							"safe_autofix remains disabled",
						},
					))
					break
				}
				seen[key] = line
			}
			return gotreesitter.WalkSkipChildren
		})
		return gotreesitter.WalkSkipChildren
	})
	return out
}

func rustBoolParameters(source []byte, fn *gotreesitter.Node, lang *gotreesitter.Language) map[string]bool {
	out := map[string]bool{}
	params := fn.ChildByFieldName("parameters", lang)
	if params == nil {
		return out
	}
	for i := 0; i < params.NamedChildCount(); i++ {
		param := params.NamedChild(i)
		if param == nil || param.Type(lang) != "parameter" {
			continue
		}
		pattern := param.ChildByFieldName("pattern", lang)
		typ := param.ChildByFieldName("type", lang)
		if pattern == nil || typ == nil || pattern.Type(lang) != "identifier" || nodeText(source, typ) != "bool" {
			continue
		}
		out[nodeText(source, pattern)] = true
	}
	return out
}

func rustIfChain(start *gotreesitter.Node, lang *gotreesitter.Language) []*gotreesitter.Node {
	var out []*gotreesitter.Node
	for current := start; current != nil && current.Type(lang) == "if_expression"; {
		out = append(out, current)
		alternative := current.ChildByFieldName("alternative", lang)
		if alternative == nil || alternative.Type(lang) != "else_clause" {
			break
		}
		current = directNamedChildOfType(alternative, lang, "if_expression")
	}
	return out
}

func rustBoolCondition(source []byte, node *gotreesitter.Node, lang *gotreesitter.Language, params map[string]bool) (string, bool) {
	if node == nil {
		return "", false
	}
	if node.Type(lang) == "identifier" {
		name := nodeText(source, node)
		if params[name] {
			return name, true
		}
		return "", false
	}
	if node.Type(lang) == "unary_expression" && node.NamedChildCount() == 1 {
		child := node.NamedChild(0)
		if child != nil && child.Type(lang) == "identifier" {
			name := nodeText(source, child)
			if params[name] && strings.TrimSpace(nodeText(source, node)) == "!"+name {
				return "!" + name, true
			}
		}
	}
	return "", false
}

func deadCodeFindings(files []*parsedFile) []model.Finding {
	var candidates []deadCodeCandidate
	nameCounts := map[string]int{}
	for _, file := range files {
		if file.generated {
			continue
		}
		root := file.document.tree.RootNode()
		lang := file.document.language
		pendingAttribute := false
		for i := 0; i < root.NamedChildCount(); i++ {
			node := root.NamedChild(i)
			if node == nil {
				continue
			}
			switch node.Type(lang) {
			case "attribute_item", "inner_attribute_item":
				pendingAttribute = true
				continue
			case "function_item":
				if pendingAttribute {
					pendingAttribute = false
					continue
				}
				if directNamedChildOfType(node, lang, "visibility_modifier") != nil ||
					directNamedChildOfType(node, lang, "function_modifiers") != nil {
					continue
				}
				nameNode := node.ChildByFieldName("name", lang)
				if nameNode == nil || nameNode.Type(lang) != "identifier" {
					continue
				}
				name := nodeText(file.source, nameNode)
				if name == "" || name == "main" {
					continue
				}
				candidates = append(candidates, deadCodeCandidate{nameNode: nameNode, name: name, file: file, node: node})
				nameCounts[name]++
			default:
				pendingAttribute = false
			}
		}
	}

	var out []model.Finding
	for _, candidate := range candidates {
		if nameCounts[candidate.name] != 1 || rustIdentifierReferenced(files, candidate) {
			continue
		}
		line := 1 + int(candidate.node.StartPoint().Row)
		out = append(out, newFinding(
			ruleDeadCode,
			model.CategoryDeadCode,
			model.SeverityLow,
			model.ConfidenceHigh,
			candidate.file.rel,
			line,
			fmt.Sprintf("private top-level Rust function %s has zero repository references", candidate.name),
			[]string{
				"AST-proven private top-level free function with no attributes or function modifiers",
				"no same-name identifier reference was found in visible Rust source",
				"public, attributed, modified, method, nested-module, generated, duplicate-name, and main entry functions fail closed",
				"safe_autofix remains disabled",
			},
		))
	}
	return out
}

func rustIdentifierReferenced(files []*parsedFile, candidate deadCodeCandidate) bool {
	for _, file := range files {
		lang := file.document.language
		found := false
		gotreesitter.Walk(file.document.tree.RootNode(), func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
			if node.Type(lang) != "identifier" || nodeText(file.source, node) != candidate.name {
				return gotreesitter.WalkContinue
			}
			if file == candidate.file &&
				node.StartByte() == candidate.nameNode.StartByte() &&
				node.EndByte() == candidate.nameNode.EndByte() {
				return gotreesitter.WalkContinue
			}
			found = true
			return gotreesitter.WalkStop
		})
		if found {
			return true
		}
	}
	return false
}

func directNamedChildOfType(node *gotreesitter.Node, lang *gotreesitter.Language, typ string) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child != nil && child.Type(lang) == typ {
			return child
		}
	}
	return nil
}

func nodeText(source []byte, node *gotreesitter.Node) string {
	if node == nil {
		return ""
	}
	start, end := int(node.StartByte()), int(node.EndByte())
	if start < 0 || end < start || start >= len(source) {
		return ""
	}
	if end > len(source) {
		end = len(source)
	}
	return string(source[start:end])
}

func newFinding(rule string, category model.Category, severity model.Severity, confidence model.Confidence, path string, line int, summary string, evidence []string) model.Finding {
	return model.Finding{
		ID:           detector.FindingID(rule, path, line, summary),
		RuleID:       rule,
		Category:     category,
		Severity:     severity,
		Confidence:   confidence,
		Path:         path,
		LineStart:    line,
		LineEnd:      line,
		Summary:      summary,
		Evidence:     evidence,
		Verification: []string{"cargo check", "cargo test"},
		SafeAutofix:  false,
	}
}
