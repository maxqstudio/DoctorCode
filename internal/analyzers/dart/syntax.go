package dartanalysis

import (
	"fmt"

	gotreesitter "github.com/odvcencio/gotreesitter"
	dartgrammar "github.com/odvcencio/gotreesitter/grammars/dart"
)

type syntaxDocument struct {
	tree     *gotreesitter.Tree
	language *gotreesitter.Language
}

func parseSyntax(source []byte) (*syntaxDocument, error) {
	language := dartgrammar.Language()
	if language == nil {
		return nil, fmt.Errorf("Dart parser unavailable")
	}

	parser := gotreesitter.NewParser(language)
	result, err := parser.ParseWithStrict(source)
	if err != nil {
		return nil, fmt.Errorf("Dart strict parse failed: %w", err)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("Dart strict parse returned no tree")
	}
	root := result.Tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("Dart strict parse returned no root node")
	}
	if root.HasErrorOrMissing() {
		return nil, fmt.Errorf("Dart syntax tree contains ERROR or MISSING nodes")
	}

	return &syntaxDocument{tree: result.Tree, language: language}, nil
}
