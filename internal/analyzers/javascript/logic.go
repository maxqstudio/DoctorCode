package javascriptanalysis

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

const ruleLogic = "JS-LOGIC-DUPLICATE-CONDITION"

var (
	logicIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	logicNegatedIdentifier = regexp.MustCompile(`^!\s*([A-Za-z_$][A-Za-z0-9_$]*)$`)
	logicStrictPrimitive = regexp.MustCompile(`^([A-Za-z_$][A-Za-z0-9_$]*)\s*(===|!==)\s*(true|false|null)$`)
)

type logicCondition struct {
	start int
	end   int
}

func logicFindings(source, structural, path string) []model.Finding {
	var out []model.Finding
	continuation := map[int]bool{}

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

		seen := map[string]int{}
		for _, condition := range conditions {
			key, ok := normalizeLogicCondition(structural[condition.start:condition.end])
			if !ok {
				// Unsupported/coercive/side-effect-capable conditions are a proof barrier:
				// an intervening evaluation could change state before a later duplicate.
				seen = map[string]int{}
				continue
			}
			line := lineAt(source, condition.start)
			if earlier, exists := seen[key]; exists {
				out = append(out, makeFinding(
					ruleLogic,
					model.CategoryLogic,
					model.SeverityMedium,
					model.ConfidenceHigh,
					path,
					line,
					"duplicate side-effect-free primitive condition appears later in the same if/else-if chain",
					[]string{
						fmt.Sprintf("same normalized primitive condition already appeared at line %d", earlier),
						"proof subset is limited to bare identifiers, negated identifiers, and identifier ===/!== true/false/null",
						"unsupported or side-effect-capable conditions reset duplicate tracking",
						"safe_autofix remains disabled",
					},
				))
				break
			}
			seen[key] = line
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

func logicIdentifierByte(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '$'
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
