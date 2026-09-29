package javascriptanalysis

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzerFindsBoundedRules(t *testing.T) {
	root := t.TempDir()
	source := `function decision(flag) {
  if (flag) {
    return true;
  } else {
    return false;
  }
}

const apiToken = "hardcoded-production-token-12345";
`
	if err := os.WriteFile(filepath.Join(root, "sample.js"), []byte(source), 0o600); err != nil { t.Fatal(err) }
	findings, err := New().Analyze(context.Background(), root)
	if err != nil { t.Fatal(err) }
	if len(findings) != 2 { t.Fatalf("findings=%d want=2: %#v", len(findings), findings) }
	for _, finding := range findings {
		if finding.SafeAutofix { t.Fatal("JS/TS findings must not authorize autofix") }
		for _, evidence := range finding.Evidence {
			if strings.Contains(evidence, "hardcoded-production-token-12345") { t.Fatal("secret leaked in evidence") }
		}
	}
}

func TestMaskStructuralRejectsUnterminatedLexicalConstructs(t *testing.T) {
	cases := []string{
		"const value = 'unterminated",
		"const value = \"unterminated",
		"const value = `unterminated",
		"/* unterminated",
	}
	for _, source := range cases {
		if _, err := maskStructural(source); err == nil {
			t.Fatalf("maskStructural(%q) error=nil, want fail-closed lexical error", source)
		}
	}
}

func TestMaskStructuralIgnoresCodeLikeNonCodeRegions(t *testing.T) {
	source := "// if (flag) { return true; } else { return false; }\n" +
		"const a = 'if (flag) { return true; } else { return false; }';\n" +
		"const b = `if (flag) { return true; } else { return false; }`;\n" +
		"/* if (flag) { return true; } else { return false; } */\n"
	masked, err := maskStructural(source)
	if err != nil {
		t.Fatal(err)
	}
	if boolReturn.MatchString(masked) {
		t.Fatalf("masked non-code region still matched simplify rule: %q", masked)
	}
}
