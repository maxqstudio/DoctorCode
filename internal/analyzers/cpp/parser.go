package cppanalysis

import (
	"fmt"

	gotreesitter "github.com/odvcencio/gotreesitter"
	cgrammar "github.com/odvcencio/gotreesitter/grammars/c"
	cppgrammar "github.com/odvcencio/gotreesitter/grammars/cpp"
)

type sourceKind string

const (
	sourceC   sourceKind = "C"
	sourceCPP sourceKind = "C++"
)

type syntaxDocument struct {
	tree     *gotreesitter.Tree
	language *gotreesitter.Language
	kind     sourceKind
}

func parseTranslationUnit(source []byte, kind sourceKind) (*syntaxDocument, error) {
	var language *gotreesitter.Language
	switch kind {
	case sourceC:
		language = cgrammar.Language()
	case sourceCPP:
		language = cppgrammar.Language()
	default:
		return nil, fmt.Errorf("unsupported C/C++ source kind %q", kind)
	}
	if language == nil {
		return nil, fmt.Errorf("%s parser unavailable", kind)
	}

	parser := gotreesitter.NewParser(language)
	result, err := parser.ParseWithStrict(source)
	if err != nil {
		return nil, fmt.Errorf("%s strict parse failed: %w", kind, err)
	}
	if result.Tree == nil {
		return nil, fmt.Errorf("%s strict parse returned no tree", kind)
	}
	root := result.Tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("%s strict parse returned no root node", kind)
	}
	if root.HasErrorOrMissing() {
		return nil, fmt.Errorf("%s syntax tree contains ERROR or MISSING nodes", kind)
	}
	return &syntaxDocument{tree: result.Tree, language: language, kind: kind}, nil
}
