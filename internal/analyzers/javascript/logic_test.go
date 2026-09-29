package javascriptanalysis

import "testing"

func TestNormalizeLogicConditionBoundaries(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"flag", "id:flag", true},
		{"! flag", "not:id:flag", true},
		{"((flag))", "id:flag", true},
		{"flag === null", "strict:flag:===:null", true},
		{"flag !== false", "strict:flag:!==:false", true},
		{"ready()", "", false},
		{"state.ready", "", false},
		{"flag == true", "", false},
		{"flag = true", "", false},
		{"true === flag", "", false},
	}
	for _, test := range tests {
		got, ok := normalizeLogicCondition(test.input)
		if ok != test.ok || got != test.want {
			t.Fatalf("normalizeLogicCondition(%q)=(%q,%v) want=(%q,%v)", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestLogicFindingsDuplicatePureCondition(t *testing.T) {
	source := `function choose(flag, other) {
  if (flag) {
    return 1;
  } else if (other) {
    return 2;
  } else if (flag) {
    return 3;
  }
}`
	structural, err := maskStructural(source)
	if err != nil {
		t.Fatal(err)
	}
	findings := logicFindings(source, structural, "sample.js")
	if len(findings) != 1 {
		t.Fatalf("findings=%d want=1: %#v", len(findings), findings)
	}
	if findings[0].RuleID != ruleLogic || findings[0].LineStart != 6 {
		t.Fatalf("finding=%#v", findings[0])
	}
	if findings[0].SafeAutofix {
		t.Fatal("M19 LOGIC finding must not authorize autofix")
	}
}

func TestLogicFindingsUnsupportedConditionIsBarrier(t *testing.T) {
	source := `function choose(flag) {
  if (flag) {
    return 1;
  } else if (mutate()) {
    return 2;
  } else if (flag) {
    return 3;
  }
}`
	structural, err := maskStructural(source)
	if err != nil {
		t.Fatal(err)
	}
	if findings := logicFindings(source, structural, "sample.js"); len(findings) != 0 {
		t.Fatalf("unsupported condition must break duplicate proof: %#v", findings)
	}
}

func TestLogicFindingsRejectsUnprovenGlobalBinding(t *testing.T) {
	source := `Object.defineProperty(globalThis, "flag", { get() { return Math.random() > 0.5 } })
function choose(other) {
  if (flag) {
    return 1
  } else if (other) {
    return 2
  } else if (flag) {
    return 3
  }
}`
	structural, err := maskStructural(source)
	if err != nil {
		t.Fatal(err)
	}
	if findings := logicFindings(source, structural, "sample.js"); len(findings) != 0 {
		t.Fatalf("unproven global binding must not produce M19 logic finding: %#v", findings)
	}
}

func TestLogicNearestFunctionParameterUsesInnermostFunction(t *testing.T) {
	source := `function outer(flag) {
  function inner(other) {
    if (flag) {
      return 1
    } else if (other) {
      return 2
    } else if (flag) {
      return 3
    }
  }
}`
	structural, err := maskStructural(source)
	if err != nil {
		t.Fatal(err)
	}
	if findings := logicFindings(source, structural, "sample.js"); len(findings) != 0 {
		t.Fatalf("closure binding is outside M19 proof and must not produce a finding: %#v", findings)
	}
}

func TestLogicFindingsParameterMutationIsBarrier(t *testing.T) {
	cases := []string{
		`function choose(flag, other) {
  if (flag) {
    return 1
  } else if (other) {
    flag = false
    return 2
  } else if (flag) {
    return 3
  }
}`,
		`function choose(flag, other) {
  if (flag) {
    return 1
  } else if (other) {
    flag++
    return 2
  } else if (flag) {
    return 3
  }
}`,
	}
	for _, source := range cases {
		structural, err := maskStructural(source)
		if err != nil {
			t.Fatal(err)
		}
		if findings := logicFindings(source, structural, "sample.js"); len(findings) != 0 {
			t.Fatalf("parameter mutation must invalidate duplicate-condition proof: %#v", findings)
		}
	}
}
