package javascriptanalysis

import (
	"fmt"
	"path/filepath"
	"strings"

	gotreesitter "github.com/odvcencio/gotreesitter"
	javascriptgrammar "github.com/odvcencio/gotreesitter/grammars/javascript"
	tsxgrammar "github.com/odvcencio/gotreesitter/grammars/tsx"
	typescriptgrammar "github.com/odvcencio/gotreesitter/grammars/typescript"
)

func validateSyntax(path string, source []byte) error {
	language, languageName, err := syntaxLanguageForPath(path)
	if err != nil {
		return err
	}
	if language == nil {
		return fmt.Errorf("javascript/typescript parser unavailable for %s", path)
	}

	parser := gotreesitter.NewParser(language)
	result, err := parser.ParseWithStrict(source)
	if err != nil {
		return fmt.Errorf("%s strict parse failed: %w", languageName, err)
	}
	if result.Tree == nil {
		return fmt.Errorf("%s strict parse returned no tree", languageName)
	}
	root := result.Tree.RootNode()
	if root == nil {
		return fmt.Errorf("%s strict parse returned no root node", languageName)
	}
	if root.HasErrorOrMissing() {
		return fmt.Errorf("%s syntax tree contains ERROR or MISSING nodes", languageName)
	}
	return nil
}

func syntaxLanguageForPath(path string) (*gotreesitter.Language, string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs", ".cjs", ".jsx":
		return javascriptgrammar.Language(), "javascript", nil
	case ".ts", ".mts", ".cts":
		return typescriptgrammar.Language(), "typescript", nil
	case ".tsx":
		return tsxgrammar.Language(), "tsx", nil
	default:
		return nil, "", fmt.Errorf("unsupported JavaScript/TypeScript extension %q", filepath.Ext(path))
	}
}
