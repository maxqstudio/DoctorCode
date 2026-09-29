package javascriptanalysis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

const ruleLogic = "JS-LOGIC-DUPLICATE-CONDITION"

var (
	logicIdentifier        = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	logicNegatedIdentifier = regexp.MustCompile(`^!\s*([A-Za-z_$][A-Za-z0-9_$]*)$`)
	logicStrictPrimitive   = regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)\s*(===|!==)\s*(true|false|null)$`)
	logicLeadingIdentifier = regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)`)
)

type logicCondition struct {
	start int
	end   int
}

type logicFunctionScope struct {
	bodyStart int
	bodyEnd   int
	params    map[string]bool
}

type logicSeenCondition struct {
	line int
	end  int
}

func logicFindings(source, structural, path string) []model.Finding {
	var out []model.Finding
	continuation := map[int]bool{}
	functionScopes := collectLogicFunctionScopes(structural)

	for search := 0; search < len(structural); {
		start := findLogicKeyword(structural, "if", search)
		if start < 0 {
			break
		}
		search = start + len("if")
		if continuation[start] {
			continue
		}

		conditions, continuationPositions := parseLogicIfChain(structural, start)
		for _, pos := range continuationPositions {
			continuation[pos] = true
		}
		if len(conditions) < 2 {
			continue
		}

		seen := map[string]logicSeenCondition{}
		for _, condition := range conditions {
			key, ok := normalizeLogicCondition(structural[condition.start:condition.end])
			if !ok {
				seen = map[string]logicSeenCondition{}
				continue
			}
			binding := logicConditionBindingName(key)
			if binding == "" || !logicNearestFunctionParameter(functionScopes, condition.start, binding) {
				// Parserless M19 does not infer globals, closures, arrow/method
				// parameters, or local declaration scopes. An unproven binding is
				// a barrier, not evidence for a duplicate-condition finding.
				seen = map[string]logicSeenCondition{}
				continue
			}

			line := lineAt(source, condition.start)
			if earlier, exists := seen[key]; exists {
				if logicBindingMutated(structural[earlier.end:condition.start], binding) {
					seen = map[string]logicSeenCondition{}
					continue
				}
				out = append(out, makeFinding(
					ruleLogic,
					model.CategoryLogic,
					model.SeverityMedium,
					model.ConfidenceHigh,
					path,
					line,
					"duplicate side-effect-free primitive parameter condition appears later in the same if/else-if chain",
					[]string{
						fmt.Sprintf("same normalized primitive parameter condition already appeared at line %d", earlier.line),
						"proof subset is limited to nearest function parameters used as bare identifiers, negated identifiers, or identifier ===/!== true/false/null",
						"unsupported, side-effect-capable, or unproven bindings reset duplicate tracking",
						"safe_autofix remains disabled",
					},
				))
				break
			}
			seen[key] = logicSeenCondition{line: line, end: condition.end}
		}
	}
	return out
}

func parseLogicIfChain(structural string, start int) ([]logicCondition, []int) {
	var conditions []logicCondition
	var continuations []int
	current := start

	for keywordAt(structural, current, "if") {
		open := skipLogicSpace(structural, current+len("if"))
		if open >= len(structural) || structural[open] != '(' {
			break
		}
		close := matchingLogicDelimiter(structural, open, '(', ')')
		if close < 0 {
			break
		}
		bodyOpen := skipLogicSpace(structural, close+1)
		if bodyOpen >= len(structural) || structural[bodyOpen] != '{' {
			break
		}
		bodyClose := matchingLogicDelimiter(structural, bodyOpen, '{', '}')
		if bodyClose < 0 {
			break
		}
		conditions = append(conditions, logicCondition{start: open + 1, end: close})

		next := skipLogicSpace(structural, bodyClose+1)
		if !keywordAt(structural, next, "else") {
			break
		}
		next = skipLogicSpace(structural, next+len("else"))
		if !keywordAt(structural, next, "if") {
			break
		}
		continuations = append(continuations, next)
		current = next
	}
	return conditions, continuations
}

func normalizeLogicCondition(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	for {
		unwrapped, ok := unwrapLogicParens(value)
		if !ok {
			break
		}
		value = strings.TrimSpace(unwrapped)
	}

	if logicIdentifier.MatchString(value) {
		return "id:" + value, true
	}
	if match := logicNegatedIdentifier.FindStringSubmatch(value); match != nil {
		return "not:id:" + match[1], true
	}
	if match := logicStrictPrimitive.FindStringSubmatch(value); match != nil {
		return "strict:" + match[1] + ":" + match[2] + ":" + match[3], true
	}
	return "", false
}

func logicConditionBindingName(key string) string {
	switch {
	case strings.HasPrefix(key, "id:"):
		return strings.TrimPrefix(key, "id:")
	case strings.HasPrefix(key, "not:id:"):
		return strings.TrimPrefix(key, "not:id:")
	case strings.HasPrefix(key, "strict:"):
		rest := strings.TrimPrefix(key, "strict:")
		if cut := strings.IndexByte(rest, ':'); cut >= 0 {
			return rest[:cut]
		}
	}
	return ""
}

func collectLogicFunctionScopes(structural string) []logicFunctionScope {
	var scopes []logicFunctionScope
	for search := 0; search < len(structural); {
		start := findLogicKeyword(structural, "function", search)
		if start < 0 {
			break
		}
		search = start + len("function")
		cursor := skipLogicSpace(structural, search)
		if cursor < len(structural) && structural[cursor] == '*' {
			cursor = skipLogicSpace(structural, cursor+1)
		}
		if cursor < len(structural) && logicIdentifierStart(structural[cursor]) {
			for cursor < len(structural) && logicIdentifierByte(structural[cursor]) {
				cursor++
			}
			cursor = skipLogicSpace(structural, cursor)
		}
		if cursor >= len(structural) || structural[cursor] != '(' {
			continue
		}
		paramsClose := matchingLogicDelimiter(structural, cursor, '(', ')')
		if paramsClose < 0 {
			continue
		}
		bodyOpen := logicFunctionBodyOpen(structural, paramsClose+1)
		if bodyOpen < 0 {
			continue
		}
		bodyClose := matchingLogicDelimiter(structural, bodyOpen, '{', '}')
		if bodyClose < 0 {
			continue
		}
		scopes = append(scopes, logicFunctionScope{
			bodyStart: bodyOpen,
			bodyEnd:   bodyClose,
			params:    logicParameterNames(structural[cursor+1 : paramsClose]),
		})
	}
	return scopes
}

func logicFunctionBodyOpen(structural string, pos int) int {
	pos = skipLogicSpace(structural, pos)
	if pos < len(structural) && structural[pos] == '{' {
		return pos
	}
	if pos >= len(structural) || structural[pos] != ':' {
		return -1
	}
	// TypeScript return annotations are supported only when the first structural
	// brace after ':' is the function body. Complex object-return annotations
	// therefore fail conservative scope proof instead of being guessed.
	for i := pos + 1; i < len(structural); i++ {
		switch structural[i] {
		case '{':
			return i
		case ';', '=':
			return -1
		}
	}
	return -1
}

func logicParameterNames(raw string) map[string]bool {
	out := map[string]bool{}
	for _, segment := range splitLogicParameters(raw) {
		value := strings.TrimSpace(segment)
		if value == "" || strings.HasPrefix(value, "...") || strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
			continue
		}
		match := logicLeadingIdentifier.FindStringSubmatch(value)
		if match == nil {
			continue
		}
		name := match[1]
		rest := strings.TrimSpace(value[len(name):])
		if rest == "" || strings.HasPrefix(rest, ":") || strings.HasPrefix(rest, "?") || strings.HasPrefix(rest, "=") {
			out[name] = true
		}
	}
	return out
}

func splitLogicParameters(raw string) []string {
	var out []string
	start := 0
	paren, bracket, brace, angle := 0, 0, 0, 0
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		case '{':
			brace++
		case '}':
			if brace > 0 {
				brace--
			}
		case '<':
			angle++
		case '>':
			if angle > 0 {
				angle--
			}
		case ',':
			if paren == 0 && bracket == 0 && brace == 0 && angle == 0 {
				out = append(out, raw[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, raw[start:])
	return out
}

func logicNearestFunctionParameter(scopes []logicFunctionScope, pos int, name string) bool {
	best := -1
	bestSpan := 0
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
	return best >= 0 && scopes[best].params[name]
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

func isLogicSpace(ch byte) bool {
	switch ch {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func unwrapLogicParens(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '(' {
		return "", false
	}
	close := matchingLogicDelimiter(value, 0, '(', ')')
	if close != len(value)-1 {
		return "", false
	}
	return value[1:close], true
}

func findLogicKeyword(source, keyword string, start int) int {
	for i := start; i+len(keyword) <= len(source); i++ {
		if keywordAt(source, i, keyword) {
			return i
		}
	}
	return -1
}

func keywordAt(source string, pos int, keyword string) bool {
	if pos < 0 || pos+len(keyword) > len(source) || source[pos:pos+len(keyword)] != keyword {
		return false
	}
	if pos > 0 && (logicIdentifierByte(source[pos-1]) || source[pos-1] == '.') {
		return false
	}
	end := pos + len(keyword)
	return end == len(source) || !logicIdentifierByte(source[end])
}

func logicIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_' || ch == '$'
}

func logicIdentifierByte(ch byte) bool {
	return logicIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func skipLogicSpace(source string, pos int) int {
	for pos < len(source) {
		switch source[pos] {
		case ' ', '\t', '\r', '\n':
			pos++
		default:
			return pos
		}
	}
	return pos
}

func matchingLogicDelimiter(source string, open int, left, right byte) int {
	if open < 0 || open >= len(source) || source[open] != left {
		return -1
	}
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case left:
			depth++
		case right:
			depth--
			if depth == 0 {
				return i
			}
			if depth < 0 {
				return -1
			}
		}
	}
	return -1
}
