package javascriptanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func analyzeLogicFixture(t *testing.T, filename, source string) []model.Finding {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, filename), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var logic []model.Finding
	for _, finding := range findings {
		if finding.RuleID == ruleLogic {
			logic = append(logic, finding)
		}
	}
	return logic
}

func TestASTLogicFindsSupportedPrimitiveConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition string
	}{
		{name: "identifier", condition: "flag"},
		{name: "negated identifier", condition: "!flag"},
		{name: "nested parens", condition: "((flag))"},
		{name: "strict null", condition: "flag === null"},
		{name: "strict false", condition: "flag !== false"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := "function choose(flag, other) {\n" +
				"  if (" + test.condition + ") {\n" +
				"    return 1;\n" +
				"  } else if (other) {\n" +
				"    return 2;\n" +
				"  } else if (" + test.condition + ") {\n" +
				"    return 3;\n" +
				"  }\n" +
				"}\n"
			findings := analyzeLogicFixture(t, "sample.js", source)
			if len(findings) != 1 {
				document, parseErr := parseSyntax("sample.js", []byte(source))
				if parseErr != nil {
					t.Fatalf("findings=%d want=1 parseErr=%v: %#v", len(findings), parseErr, findings)
				}
				t.Fatalf("findings=%d want=1 tree=%s: %#v", len(findings), document.tree.RootNode().SExpr(document.language), findings)
			}
			if findings[0].LineStart != 6 || findings[0].SafeAutofix {
				t.Fatalf("unexpected finding: %#v", findings[0])
			}
		})
	}
}

func TestASTLogicRejectsUnsupportedConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition string
	}{
		{name: "call", condition: "ready()"},
		{name: "member", condition: "state.ready"},
		{name: "loose equality", condition: "flag == true"},
		{name: "assignment", condition: "flag = true"},
		{name: "reversed strict", condition: "true === flag"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := "function choose(flag, other, state) {\n" +
				"  if (flag) {\n" +
				"    return 1;\n" +
				"  } else if (" + test.condition + ") {\n" +
				"    return 2;\n" +
				"  } else if (flag) {\n" +
				"    return 3;\n" +
				"  }\n" +
				"}\n"
			if findings := analyzeLogicFixture(t, "sample.js", source); len(findings) != 0 {
				t.Fatalf("unsupported condition must break duplicate proof: %#v", findings)
			}
		})
	}
}

func TestASTLogicRejectsUnprovenGlobalBinding(t *testing.T) {
	source := `Object.defineProperty(globalThis, "flag", { get() { return Math.random() > 0.5 } })
function choose(other) {
  if (flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag) {
    return 3;
  }
}
`
	if findings := analyzeLogicFixture(t, "sample.js", source); len(findings) != 0 {
		t.Fatalf("unproven global binding must not produce M20 logic finding: %#v", findings)
	}
}

func TestASTLogicUsesNearestFunctionBoundary(t *testing.T) {
	source := `function outer(flag) {
  function inner(other) {
    if (flag) {
      return 1;
    } else if (other) {
      return 2;
    } else if (flag) {
      return 3;
    }
  }
}
`
	if findings := analyzeLogicFixture(t, "sample.js", source); len(findings) != 0 {
		t.Fatalf("closure binding is outside M20 proof and must not produce a finding: %#v", findings)
	}
}

func TestASTLogicRejectsArrowAndMethodParameterWidening(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "arrow",
			source: `const choose = (flag, other) => {
  if (flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag) {
    return 3;
  }
};
`,
		},
		{
			name: "method",
			source: `class Picker {
  choose(flag, other) {
    if (flag) {
      return 1;
    } else if (other) {
      return 2;
    } else if (flag) {
      return 3;
    }
  }
}
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if findings := analyzeLogicFixture(t, "sample.js", test.source); len(findings) != 0 {
				t.Fatalf("M20 must not widen M19 parameter scope: %#v", findings)
			}
		})
	}
}
