package javascriptanalysis

import (
	"fmt"
	"strings"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

const ruleLogic = "JS-LOGIC-DUPLICATE-CONDITION"

type logicASTCondition struct {
	node      *gotreesitter.Node
	condition *gotreesitter.Node
}

type logicSeenCondition struct {
	line int
	end  int
}

type logicASTFunctionScope struct {
	bodyStart uint32
	bodyEnd   uint32
	params    map[string]bool
	supported bool
}

func logicFindingsAST(source, structural, path string, root *gotreesitter.Node, language *gotreesitter.Language) []model.Finding {
	if root == nil || language == nil {
		return nil
	}
	var out []model.Finding
	functionScopes := collectLogicASTFunctionScopes(root, language, source)
	continuations := map[uint32]bool{}
	gotreesitter.Walk(root, func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		if node.Type(language) != "if_statement" || continuations[node.StartByte()] {
			return gotreesitter.WalkContinue
		}
		chain := logicASTIfChain(node, language)
		for i := 1; i < len(chain); i++ {
			continuations[chain[i].node.StartByte()] = true
		}
		if finding, ok := logicFindingForASTChain(source, structural, path, chain, language, functionScopes); ok {
			out = append(out, finding)
		}
		return gotreesitter.WalkContinue
	})
	return out
}

func logicFindingForASTChain(source, structural, path string, conditions []logicASTCondition, language *gotreesitter.Language, functionScopes []logicASTFunctionScope) (model.Finding, bool) {
	if len(conditions) < 2 {
		return model.Finding{}, false
	}

	seen := map[string]logicSeenCondition{}
	for _, item := range conditions {
		key, binding, ok := normalizeLogicASTCondition(source, item.condition, language)
		if !ok || !logicASTNearestTraditionalParameter(functionScopes, item.condition.StartByte(), binding) {
			seen = map[string]logicSeenCondition{}
			continue
		}

		line := 1 + int(item.condition.StartPoint().Row)
		if earlier, exists := seen[key]; exists {
			startByte := earlier.end
			endByte := int(item.condition.StartByte())
			if startByte < 0 || endByte < startByte || endByte > len(structural) {
				seen = map[string]logicSeenCondition{}
				continue
			}
			if logicBindingMutated(structural[startByte:endByte], binding) {
				seen = map[string]logicSeenCondition{}
				continue
			}
			return makeFinding(
				ruleLogic,
				model.CategoryLogic,
				model.SeverityMedium,
				model.ConfidenceHigh,
				path,
				line,
				"duplicate side-effect-free primitive parameter condition appears later in the same if/else-if chain",
				[]string{
					fmt.Sprintf("AST-proven same primitive parameter condition already appeared at line %d", earlier.line),
					"proof subset is limited to nearest traditional function parameters used as bare identifiers, negated identifiers, or identifier ===/!== true/false/null",
					"unsupported, side-effect-capable, unproven, or textually mutated bindings reset duplicate tracking",
					"safe_autofix remains disabled",
				},
			), true
		}
		seen[key] = logicSeenCondition{line: line, end: int(item.condition.EndByte())}
	}
	return model.Finding{}, false
}

func logicASTIfChain(start *gotreesitter.Node, language *gotreesitter.Language) []logicASTCondition {
	var out []logicASTCondition
	for current := start; current != nil && current.Type(language) == "if_statement"; {
		consequence := current.ChildByFieldName("consequence", language)
		condition := current.ChildByFieldName("condition", language)
		if consequence == nil || consequence.Type(language) != "statement_block" || condition == nil {
			break
		}
		out = append(out, logicASTCondition{node: current, condition: condition})

		alternative := current.ChildByFieldName("alternative", language)
		if alternative == nil || alternative.Type(language) != "else_clause" {
			break
		}
		current = nil
		for i := 0; i < alternative.NamedChildCount(); i++ {
			child := alternative.NamedChild(i)
			if child != nil && child.Type(language) == "if_statement" {
				current = child
				break
			}
		}
	}
	return out
}

func normalizeLogicASTCondition(source string, node *gotreesitter.Node, language *gotreesitter.Language) (string, string, bool) {
	node = unwrapLogicASTParens(node, language)
	if node == nil {
		return "", "", false
	}
	switch node.Type(language) {
	case "identifier":
		name := logicNodeText(source, node)
		if name == "" {
			return "", "", false
		}
		return "id:" + name, name, true
	case "unary_expression":
		operator := node.ChildByFieldName("operator", language)
		argument := unwrapLogicASTParens(node.ChildByFieldName("argument", language), language)
		if operator == nil || argument == nil || logicNodeText(source, operator) != "!" || argument.Type(language) != "identifier" {
			return "", "", false
		}
		name := logicNodeText(source, argument)
		if name == "" {
			return "", "", false
		}
		return "not:id:" + name, name, true
	case "binary_expression":
		left := node.ChildByFieldName("left", language)
		operator := node.ChildByFieldName("operator", language)
		right := node.ChildByFieldName("right", language)
		if left == nil || operator == nil || right == nil || left.Type(language) != "identifier" {
			return "", "", false
		}
		op := logicNodeText(source, operator)
		if op != "===" && op != "!==" {
			return "", "", false
		}
		switch right.Type(language) {
		case "true", "false", "null":
		default:
			return "", "", false
		}
		name := logicNodeText(source, left)
		if name == "" {
			return "", "", false
		}
		return "strict:" + name + ":" + op + ":" + right.Type(language), name, true
	default:
		return "", "", false
	}
}

func unwrapLogicASTParens(node *gotreesitter.Node, language *gotreesitter.Language) *gotreesitter.Node {
	for node != nil && node.Type(language) == "parenthesized_expression" {
		if node.NamedChildCount() != 1 {
			return nil
		}
		node = node.NamedChild(0)
	}
	return node
}

func collectLogicASTFunctionScopes(root *gotreesitter.Node, language *gotreesitter.Language, source string) []logicASTFunctionScope {
	var scopes []logicASTFunctionScope
	gotreesitter.Walk(root, func(node *gotreesitter.Node, _ int) gotreesitter.WalkAction {
		kind := node.Type(language)
		supported := kind == "function_declaration" || kind == "function_expression"
		switch kind {
		case "function_declaration", "function_expression", "arrow_function", "method_definition", "generator_function", "generator_function_declaration":
		default:
			return gotreesitter.WalkContinue
		}
		body := node.ChildByFieldName("body", language)
		if body == nil {
			return gotreesitter.WalkContinue
		}
		scope := logicASTFunctionScope{
			bodyStart: body.StartByte(),
			bodyEnd:   body.EndByte(),
			params:    map[string]bool{},
			supported: supported,
		}
		if supported {
			params := node.ChildByFieldName("parameters", language)
			if params != nil {
				for i := 0; i < params.NamedChildCount(); i++ {
					if name := logicASTSimpleParameterName(params.NamedChild(i), language, source); name != "" {
						scope.params[name] = true
					}
				}
			}
		}
		scopes = append(scopes, scope)
		return gotreesitter.WalkContinue
	})
	return scopes
}

func logicASTNearestTraditionalParameter(scopes []logicASTFunctionScope, pos uint32, name string) bool {
	best := -1
	var bestSpan uint32
	for i := range scopes {
		scope := scopes[i]
		if pos <= scope.bodyStart || pos >= scope.bodyEnd {
			continue
		}
		span := scope.bodyEnd - scope.bodyStart
		if best < 0 || span < bestSpan {
			best = i
			bestSpan = span
		}
	}
	return best >= 0 && scopes[best].supported && scopes[best].params[name]
}

func logicASTHasSimpleParameter(params *gotreesitter.Node, language *gotreesitter.Language, source, name string) bool {
	if params == nil {
		return false
	}
	for i := 0; i < params.NamedChildCount(); i++ {
		if logicASTSimpleParameterName(params.NamedChild(i), language, source) == name {
			return true
		}
	}
	return false
}

func logicASTSimpleParameterName(node *gotreesitter.Node, language *gotreesitter.Language, source string) string {
	if node == nil {
		return ""
	}
	switch node.Type(language) {
	case "identifier":
		return logicNodeText(source, node)
	case "assignment_pattern":
		left := node.ChildByFieldName("left", language)
		if left != nil && left.Type(language) == "identifier" {
			return logicNodeText(source, left)
		}
	case "required_parameter", "optional_parameter":
		pattern := node.ChildByFieldName("pattern", language)
		if pattern != nil && pattern.Type(language) == "identifier" {
			return logicNodeText(source, pattern)
		}
	}
	return ""
}

func logicNodeText(source string, node *gotreesitter.Node) string {
	if node == nil {
		return ""
	}
	start, end := int(node.StartByte()), int(node.EndByte())
	if start < 0 || end < start || end > len(source) {
		return ""
	}
	return source[start:end]
}

func logicBindingMutated(segment, name string) bool {
	for search := 0; search < len(segment); {
		idx := strings.Index(segment[search:], name)
		if idx < 0 {
			return false
		}
		idx += search
		end := idx + len(name)
		if (idx == 0 || !logicIdentifierByte(segment[idx-1])) &&
			(end == len(segment) || !logicIdentifierByte(segment[end])) {
			before := idx - 1
			for before >= 0 && isLogicSpace(segment[before]) {
				before--
			}
			if before >= 1 {
				prefix := segment[before-1 : before+1]
				if prefix == "++" || prefix == "--" {
					return true
				}
			}

			after := end
			for after < len(segment) && isLogicSpace(segment[after]) {
				after++
			}
			for _, op := range []string{"++", "--", "**=", "&&=", "||=", "??=", "+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "<<=", ">>=", ">>>="} {
				if strings.HasPrefix(segment[after:], op) {
					return true
				}
			}
			if after < len(segment) && segment[after] == '=' {
				next := byte(0)
				if after+1 < len(segment) {
					next = segment[after+1]
				}
				if next != '=' && next != '>' {
					return true
				}
			}
		}
		search = end
	}
	return false
}

func logicIdentifierByte(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '$'
}

func isLogicSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}
