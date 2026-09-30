package detector

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

var ErrUnavailable = errors.New("analyzer unavailable")

type ParserKind string

const (
	ParserBuiltinAST  ParserKind = "BUILTIN_AST"
	ParserExternalAST ParserKind = "EXTERNAL_AST"
	ParserEmbeddedAST ParserKind = "EMBEDDED_AST"
)

type AvailabilityMode string

const (
	AvailabilitySourceOnly   AvailabilityMode = "SOURCE_ONLY"
	AvailabilityExternalTool AvailabilityMode = "EXTERNAL_TOOL"
)

type ParserContract struct {
	Kind       ParserKind `json:"kind"`
	Provider   string     `json:"provider"`
	FailClosed bool       `json:"fail_closed"`
}

type AvailabilityContract struct {
	Mode       AvailabilityMode `json:"mode"`
	Dependency string           `json:"dependency,omitempty"`
}

type RuleMetadata struct {
	ID          string         `json:"id"`
	Category    model.Category `json:"category"`
	SafeAutofix bool           `json:"safe_autofix"`
}

type BenchmarkContract struct {
	SchemaVersion int      `json:"schema_version"`
	Languages     []string `json:"languages"`
}

type Descriptor struct {
	ID               string               `json:"id"`
	Language         string               `json:"language"`
	Extensions       []string             `json:"extensions"`
	Parser           ParserContract       `json:"parser"`
	Availability     AvailabilityContract `json:"availability"`
	EvidenceBoundary string               `json:"evidence_boundary"`
	Rules            []RuleMetadata       `json:"rules"`
	Benchmark        BenchmarkContract    `json:"benchmark"`
}

type Analyzer interface {
	Name() string
	Descriptor() Descriptor
	Analyze(ctx context.Context, root string) ([]model.Finding, error)
}

func ValidateDescriptor(desc Descriptor) error {
	if strings.TrimSpace(desc.ID) == "" {
		return errors.New("descriptor id is required")
	}
	if strings.TrimSpace(desc.Language) == "" {
		return errors.New("descriptor language is required")
	}
	if len(desc.Extensions) == 0 {
		return errors.New("descriptor extensions are required")
	}
	if !sort.StringsAreSorted(desc.Extensions) {
		return errors.New("descriptor extensions must be sorted")
	}
	seenExt := map[string]bool{}
	for _, ext := range desc.Extensions {
		if ext == "" || !strings.HasPrefix(ext, ".") || ext != strings.ToLower(ext) {
			return fmt.Errorf("invalid descriptor extension %q", ext)
		}
		if seenExt[ext] {
			return fmt.Errorf("duplicate descriptor extension %q", ext)
		}
		seenExt[ext] = true
	}

	switch desc.Parser.Kind {
	case ParserBuiltinAST, ParserExternalAST, ParserEmbeddedAST:
	default:
		return fmt.Errorf("invalid parser kind %q", desc.Parser.Kind)
	}
	if strings.TrimSpace(desc.Parser.Provider) == "" {
		return errors.New("parser provider is required")
	}

	switch desc.Availability.Mode {
	case AvailabilitySourceOnly:
		if strings.TrimSpace(desc.Availability.Dependency) != "" {
			return errors.New("source-only availability must not declare an external dependency")
		}
	case AvailabilityExternalTool:
		if strings.TrimSpace(desc.Availability.Dependency) == "" {
			return errors.New("external-tool availability requires dependency")
		}
	default:
		return fmt.Errorf("invalid availability mode %q", desc.Availability.Mode)
	}

	if strings.TrimSpace(desc.EvidenceBoundary) == "" {
		return errors.New("evidence boundary is required")
	}
	if len(desc.Rules) == 0 {
		return errors.New("descriptor rules are required")
	}
	for i := 1; i < len(desc.Rules); i++ {
		if desc.Rules[i-1].ID > desc.Rules[i].ID {
			return errors.New("descriptor rules must be sorted by id")
		}
	}
	seenRules := map[string]bool{}
	for _, rule := range desc.Rules {
		if strings.TrimSpace(rule.ID) == "" {
			return errors.New("rule id is required")
		}
		if seenRules[rule.ID] {
			return fmt.Errorf("duplicate rule id %q", rule.ID)
		}
		seenRules[rule.ID] = true
		if !validCategory(rule.Category) {
			return fmt.Errorf("rule %q has invalid category %q", rule.ID, rule.Category)
		}
	}

	if desc.Benchmark.SchemaVersion != 1 {
		return fmt.Errorf("unsupported benchmark schema_version=%d", desc.Benchmark.SchemaVersion)
	}
	if len(desc.Benchmark.Languages) == 0 {
		return errors.New("benchmark languages are required")
	}
	if !sort.StringsAreSorted(desc.Benchmark.Languages) {
		return errors.New("benchmark languages must be sorted")
	}
	seenLanguages := map[string]bool{}
	for _, language := range desc.Benchmark.Languages {
		if strings.TrimSpace(language) == "" {
			return errors.New("benchmark language must not be empty")
		}
		if seenLanguages[language] {
			return fmt.Errorf("duplicate benchmark language %q", language)
		}
		seenLanguages[language] = true
	}
	return nil
}

func validCategory(category model.Category) bool {
	switch category {
	case model.CategoryBloat, model.CategorySecurity, model.CategorySimplify, model.CategoryLogic, model.CategoryDeadCode:
		return true
	default:
		return false
	}
}

func RecognizesPath(desc Descriptor, path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	index := sort.SearchStrings(desc.Extensions, ext)
	return index < len(desc.Extensions) && desc.Extensions[index] == ext
}

func RecognizesRoot(ctx context.Context, root string, desc Descriptor) (bool, error) {
	found := false
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
		if RecognizesPath(desc, path) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found, err
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", "coverage", ".venv", "venv", "__pycache__", "testdata":
		return true
	default:
		return false
	}
}

func FindingID(rule, path string, line int, summary string) string {
	key := fmt.Sprintf("%s|%s|%d|%s", rule, path, line, summary)
	digest := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%s-%x", rule, digest[:5])
}

func ValidateFindings(desc Descriptor, findings []model.Finding) error {
	rules := make(map[string]RuleMetadata, len(desc.Rules))
	for _, rule := range desc.Rules {
		rules[rule.ID] = rule
	}
	for _, finding := range findings {
		rule, ok := rules[finding.RuleID]
		if !ok {
			return fmt.Errorf("analyzer %q emitted undeclared rule %q", desc.ID, finding.RuleID)
		}
		if finding.Category != rule.Category {
			return fmt.Errorf("rule %q category=%q want=%q", finding.RuleID, finding.Category, rule.Category)
		}
		if finding.SafeAutofix != rule.SafeAutofix {
			return fmt.Errorf("rule %q safe_autofix=%v want=%v", finding.RuleID, finding.SafeAutofix, rule.SafeAutofix)
		}
		wantID := FindingID(finding.RuleID, finding.Path, finding.LineStart, finding.Summary)
		if finding.ID != wantID {
			return fmt.Errorf("rule %q finding id=%q want=%q", finding.RuleID, finding.ID, wantID)
		}
	}
	return nil
}

func SupportsBenchmarkLanguage(desc Descriptor, language string) bool {
	index := sort.SearchStrings(desc.Benchmark.Languages, language)
	return index < len(desc.Benchmark.Languages) && desc.Benchmark.Languages[index] == language
}
