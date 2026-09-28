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
func (a *Analyzer) Name() string { return "javascript/conservative-structural-v1" }

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
		rel, err := filepath.Rel(root, path); if err != nil { return nil, err }; rel = filepath.ToSlash(rel)
		for _, m := range boolReturn.FindAllStringSubmatchIndex(source, -1) {
			left, right := source[m[2]:m[3]], source[m[4]:m[5]]; if left == right { continue }
			findings = append(findings, makeFinding(ruleSimplify, model.CategorySimplify, model.SeverityLow, model.ConfidenceHigh, rel, lineAt(source, m[0]), "opposite boolean-return branches can be reduced to the condition or its negation", []string{"structurally matched boolean return branches", "safe_autofix remains disabled"}))
		}
		for _, m := range credential.FindAllStringSubmatchIndex(source, -1) {
			name, value := source[m[2]:m[3]], source[m[4]:m[5]]; if !credentialName.MatchString(name) || placeholder(value) { continue }
			digest := sha256.Sum256([]byte(value))
			findings = append(findings, makeFinding(ruleSecurity, model.CategorySecurity, model.SeverityHigh, model.ConfidenceSuspicious, rel, lineAt(source, m[0]), "credential-like identifier is assigned a hardcoded string literal", []string{fmt.Sprintf("identifier=%s literal_bytes=%d sha256_prefix=%x", name, len(value), digest[:6]), "literal value is redacted"}))
		}
	}
	return findings, nil
}

func makeFinding(rule string, category model.Category, severity model.Severity, confidence model.Confidence, path string, line int, summary string, evidence []string) model.Finding {
	key := fmt.Sprintf("%s|%s|%d|%s", rule, path, line, summary); digest := sha256.Sum256([]byte(key))
	return model.Finding{ID: fmt.Sprintf("%s-%x", rule, digest[:5]), RuleID: rule, Category: category, Severity: severity, Confidence: confidence, Path: path, LineStart: line, LineEnd: line, Summary: summary, Evidence: evidence, Verification: []string{"project parser/typecheck", "project tests"}, SafeAutofix: false}
}

func lineAt(source string, offset int) int { return 1 + strings.Count(source[:offset], "\\n") }
func supported(path string) bool { switch strings.ToLower(filepath.Ext(path)) { case ".js", ".mjs", ".cjs", ".ts", ".mts", ".cts": return true; default: return false } }
func ignoredDirectory(name string) bool { switch name { case ".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", "coverage", "testdata": return true; default: return false } }
func placeholder(value string) bool { v := strings.ToLower(strings.TrimSpace(value)); if v == "" { return true }; for _, token := range []string{"example", "placeholder", "changeme", "change-me", "dummy", "test", "your_", "your-", "<", "${"} { if strings.Contains(v, token) { return true } }; return false }
