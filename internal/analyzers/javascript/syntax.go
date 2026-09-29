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

type syntaxDocument struct {
	tree         *gotreesitter.Tree
	language     *gotreesitter.Language
	languageName string
}

func parseSyntax(path string, source []byte) (*syntaxDocument, error) {
	language, languageName, err := syntaxLanguageForPath(path)
	if err != nil {
		return nil, err
	}
	if language == nil {
		return nil, fmt.Errorf("javascript/typescript parser unavailable for %s", path)
	}

	parser := gotreesitter.NewParser(language)
	result, err := parser.ParseWithStrict(source)
	if err != nil {
		return nil, fmt.Errorf("%s strict parse failed: %w", languageName, err)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("%s strict parse returned no tree", languageName)
	}
	root := result.Tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("%s strict parse returned no root node", languageName)
	}
	if root.HasErrorOrMissing() {
		return nil, fmt.Errorf("%s syntax tree contains ERROR or MISSING nodes", languageName)
	}
	return &syntaxDocument{tree: result.Tree, language: language, languageName: languageName}, nil
}

func validateSyntax(path string, source []byte) error {
	_, err := parseSyntax(path, source)
	return err
}

func (d *syntaxDocument) lexicalViews(source string) (string, string) {
	structural := []byte(source)
	credentialSource := []byte(source)
	if d == nil || d.tree == nil || d.language == nil {
		return string(structural), string(credentialSource)
	}

	gotreesitter.Walk(d.tree.RootNode(), func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		switch node.Type(d.language) {
		case "comment":
			maskSyntaxRange(structural, node.StartByte(), node.EndByte())
			maskSyntaxRange(credentialSource, node.StartByte(), node.EndByte())
			return gotreesitter.WalkSkipChildren
		case "string":
			maskSyntaxRange(structural, node.StartByte(), node.EndByte())
			return gotreesitter.WalkSkipChildren
		case "template_string", "regex", "jsx_text":
			maskSyntaxRange(structural, node.StartByte(), node.EndByte())
			maskSyntaxRange(credentialSource, node.StartByte(), node.EndByte())
			return gotreesitter.WalkSkipChildren
		default:
			return gotreesitter.WalkContinue
		}
	})
	return string(structural), string(credentialSource)
}

func maskSyntaxRange(target []byte, start, end uint32) {
	if start >= end || int(start) >= len(target) {
		return
	}
	if int(end) > len(target) {
		end = uint32(len(target))
	}
	for i := int(start); i < int(end); i++ {
		if target[i] != '\n' && target[i] != '\r' {
			target[i] = ' '
		}
	}
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
