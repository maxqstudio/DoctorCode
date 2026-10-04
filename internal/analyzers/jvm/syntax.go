package jvmanalysis

import (
	"fmt"

	gotreesitter "github.com/odvcencio/gotreesitter"
	javagrammar "github.com/odvcencio/gotreesitter/grammars/java"
	kotlingrammar "github.com/odvcencio/gotreesitter/grammars/kotlin"
)

type sourceKind string

const (
	sourceJava   sourceKind = "java"
	sourceKotlin sourceKind = "kotlin"
)

type syntaxDocument struct {
	tree     *gotreesitter.Tree
	language *gotreesitter.Language
	kind     sourceKind
}

func parseSyntax(source []byte, kind sourceKind) (*syntaxDocument, error) {
	var language *gotreesitter.Language
	switch kind {
	case sourceJava:
		language = javagrammar.Language()
	case sourceKotlin:
		language = kotlingrammar.Language()
	default:
		return nil, fmt.Errorf("unsupported JVM source kind %q", kind)
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
