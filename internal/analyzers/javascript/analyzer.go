package javascriptanalysis

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

const (
	ruleSimplify = "JS-SIMPLIFY-BOOL-RETURN"
	ruleSecurity = "JS-SEC-HARDCODED-CREDENTIAL"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }
func (a *Analyzer) Name() string { return "javascript/parser-backed-v2" }

func (a *Analyzer) Descriptor() detector.Descriptor {
	return detector.Descriptor{
		ID:         a.Name(),
		Language:   "JavaScript / TypeScript",
		Extensions: []string{".cjs", ".cts", ".js", ".jsx", ".mjs", ".mts", ".ts", ".tsx"},
		Parser: detector.ParserContract{
			Kind:       detector.ParserEmbeddedAST,
			Provider:   "gotreesitter v0.55.1 JavaScript/TypeScript/TSX",
			FailClosed: true,
		},
		Availability:     detector.AvailabilityContract{Mode: detector.AvailabilitySourceOnly},
		EvidenceBoundary: "Embedded gotreesitter syntax/AST authority with fail-closed ERROR/MISSING rejection; semantic authority remains bounded to declared LOGIC, SECURITY, and SIMPLIFY rules without JS/TS DEADCODE, BLOAT, autofix, or whole-program claims.",
		Rules: []detector.RuleMetadata{
			{ID: ruleLogic, Category: model.CategoryLogic, SafeAutofix: false},
			{ID: ruleSecurity, Category: model.CategorySecurity, SafeAutofix: false},
			{ID: ruleSimplify, Category: model.CategorySimplify, SafeAutofix: false},
		},
		Benchmark: detector.BenchmarkContract{
			SchemaVersion: 1,
			Languages: []string{
				"JavaScript / TypeScript",
				"JavaScript / TypeScript parser-backed JSX/TSX",
			},
		},
	}
}

var boolReturn = regexp.MustCompile("(?s)if\\s*\\([^{}]+\\)\\s*\\{\\s*return\\s+(true|false)\\s*;?\\s*\\}\\s*else\\s*\\{\\s*return\\s+(true|false)\\s*;?\\s*\\}")
var credential = regexp.MustCompile("(?im)^\\s*(?:const|let|var)\\s+([A-Za-z_$][\\w$]*)(?:\\s*:\\s*[^=;]+)?\\s*=\\s*[\\\"\x27]([^\\\"\x27\\r\\n]+)[\\\"\x27]\\s*;?")
var credentialName = regexp.MustCompile("(?i)(?:api[_-]?key|api[_-]?token|access[_-]?token|auth[_-]?token|secret|password|passwd|credential)")

func (a *Analyzer) Analyze(ctx context.Context, root string) ([]model.Finding, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil { return walkErr }
		if err := ctx.Err(); err != nil { return err }
		if entry.IsDir() { if path != root && ignoredDirectory(entry.Name()) { return filepath.SkipDir }; return nil }
		if entry.Type()&fs.ModeSymlink != 0 { return nil }
		if supported(path) { files = append(files, path) }
		return nil
	})
	if err != nil { return nil, err }
	if len(files) == 0 { return nil, fmt.Errorf("%w: no supported JavaScript or TypeScript source files", detector.ErrUnavailable) }
	sort.Strings(files)
	var findings []model.Finding
	for _, path := range files {
		data, err := os.ReadFile(path); if err != nil { return nil, fmt.Errorf("read JavaScript/TypeScript source %s: %w", path, err) }
		source := string(data); if strings.IndexByte(source, 0) >= 0 { return nil, fmt.Errorf("javascript/typescript parse incomplete: %s contains NUL byte", path) }
		document, err := parseSyntax(path, data); if err != nil { return nil, fmt.Errorf("javascript/typescript syntax validation failed for %s: %w", path, err) }
		structural, credentialSource := document.lexicalViews(source)
		rel, err := filepath.Rel(root, path); if err != nil { return nil, err }; rel = filepath.ToSlash(rel)
		findings = append(findings, logicFindingsAST(source, structural, rel, document.tree.RootNode(), document.language)...)
		for _, m := range boolReturn.FindAllStringSubmatchIndex(structural, -1) {
			left, right := structural[m[2]:m[3]], structural[m[4]:m[5]]; if left == right { continue }
			findings = append(findings, makeFinding(ruleSimplify, model.CategorySimplify, model.SeverityLow, model.ConfidenceHigh, rel, lineAt(source, m[0]), "opposite boolean-return branches can be reduced to the condition or its negation", []string{"structurally matched boolean return branches", "safe_autofix remains disabled"}))
		}
		for _, m := range credential.FindAllStringSubmatchIndex(credentialSource, -1) {
			name, value := source[m[2]:m[3]], source[m[4]:m[5]]; if !credentialName.MatchString(name) || placeholder(value) { continue }
			digest := sha256.Sum256([]byte(value))
			findings = append(findings, makeFinding(ruleSecurity, model.CategorySecurity, model.SeverityHigh, model.ConfidenceSuspicious, rel, lineAt(source, m[0]), "credential-like identifier is assigned a hardcoded string literal", []string{fmt.Sprintf("identifier=%s literal_bytes=%d sha256_prefix=%x", name, len(value), digest[:6]), "literal value is redacted"}))
		}
	}
	return findings, nil
}


func maskStructural(source string) (string, error) {
	structural, _, err := maskLexicalViews(source)
	return structural, err
}

func maskLexicalViews(source string) (string, string, error) {
	structural := []byte(source)
	credentialSource := []byte(source)
	const (
		stateCode = iota
		stateSingleQuote
		stateDoubleQuote
		stateTemplate
		stateLineComment
		stateBlockComment
	)
	state := stateCode
	escaped := false
	for i := 0; i < len(structural); i++ {
		ch := source[i]
		switch state {
		case stateCode:
			if ch == '/' && i+1 < len(structural) && source[i+1] == '/' {
				structural[i], structural[i+1] = ' ', ' '
				credentialSource[i], credentialSource[i+1] = ' ', ' '
				i++
				state = stateLineComment
				continue
			}
			if ch == '/' && i+1 < len(structural) && source[i+1] == '*' {
				structural[i], structural[i+1] = ' ', ' '
				credentialSource[i], credentialSource[i+1] = ' ', ' '
				i++
				state = stateBlockComment
				continue
			}
			switch ch {
			case '\'':
				structural[i] = ' '
				state = stateSingleQuote
				escaped = false
			case '"':
				structural[i] = ' '
				state = stateDoubleQuote
				escaped = false
			case '`':
				structural[i] = ' '
				credentialSource[i] = ' '
				state = stateTemplate
				escaped = false
			}
		case stateLineComment:
			if ch == '\n' {
				state = stateCode
			} else {
				structural[i] = ' '
				credentialSource[i] = ' '
			}
		case stateBlockComment:
			if ch == '*' && i+1 < len(structural) && source[i+1] == '/' {
				structural[i], structural[i+1] = ' ', ' '
				credentialSource[i], credentialSource[i+1] = ' ', ' '
				i++
				state = stateCode
			} else if ch != '\n' {
				structural[i] = ' '
				credentialSource[i] = ' '
			}
		case stateSingleQuote, stateDoubleQuote:
			if ch == '\n' {
				if !escaped {
					return "", "", fmt.Errorf("unterminated quoted string before line break")
				}
				escaped = false
				continue
			}
			structural[i] = ' '
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if (state == stateSingleQuote && ch == '\'') ||
				(state == stateDoubleQuote && ch == '"') {
				state = stateCode
			}
		case stateTemplate:
			if ch == '\n' {
				escaped = false
				continue
			}
			structural[i] = ' '
			credentialSource[i] = ' '
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '`' {
				state = stateCode
			}
		}
	}
	switch state {
	case stateCode, stateLineComment:
		return string(structural), string(credentialSource), nil
	case stateBlockComment:
		return "", "", fmt.Errorf("unterminated block comment")
	case stateSingleQuote, stateDoubleQuote:
		return "", "", fmt.Errorf("unterminated quoted string")
	case stateTemplate:
		return "", "", fmt.Errorf("unterminated template literal")
	default:
		return "", "", fmt.Errorf("unknown lexical state")
	}
}

func makeFinding(rule string, category model.Category, severity model.Severity, confidence model.Confidence, path string, line int, summary string, evidence []string) model.Finding {
	return model.Finding{ID: detector.FindingID(rule, path, line, summary), RuleID: rule, Category: category, Severity: severity, Confidence: confidence, Path: path, LineStart: line, LineEnd: line, Summary: summary, Evidence: evidence, Verification: []string{"project parser/typecheck", "project tests"}, SafeAutofix: false}
}

func lineAt(source string, offset int) int { return 1 + strings.Count(source[:offset], "\n") }
func supported(path string) bool { switch strings.ToLower(filepath.Ext(path)) { case ".js", ".mjs", ".cjs", ".jsx", ".ts", ".mts", ".cts", ".tsx": return true; default: return false } }
func ignoredDirectory(name string) bool { switch name { case ".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", "coverage", "testdata": return true; default: return false } }
func placeholder(value string) bool { v := strings.ToLower(strings.TrimSpace(value)); if v == "" { return true }; for _, token := range []string{"example", "placeholder", "changeme", "change-me", "dummy", "test", "your_", "your-", "<", "${"} { if strings.Contains(v, token) { return true } }; return false }
