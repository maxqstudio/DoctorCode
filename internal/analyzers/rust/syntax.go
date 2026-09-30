package rustanalysis

import (
	"fmt"

	gotreesitter "github.com/odvcencio/gotreesitter"
	rustgrammar "github.com/odvcencio/gotreesitter/grammars/rust"
)

type syntaxDocument struct {
	tree     *gotreesitter.Tree
	language *gotreesitter.Language
}

func parseSyntax(source []byte) (*syntaxDocument, error) {
	language := rustgrammar.Language()
	if language == nil {
		return nil, fmt.Errorf("rust parser unavailable")
	}
	parser := gotreesitter.NewParser(language)
	result, err := parser.ParseWithStrict(source)
	if err != nil {
		return nil, fmt.Errorf("rust strict parse failed: %w", err)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("rust strict parse returned no tree")
	}
	root := result.Tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("rust strict parse returned no root node")
	}
	if root.HasErrorOrMissing() {
		return nil, fmt.Errorf("rust syntax tree contains ERROR or MISSING nodes")
	}
	return &syntaxDocument{tree: result.Tree, language: language}, nil
}
