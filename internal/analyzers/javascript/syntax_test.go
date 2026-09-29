package javascriptanalysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestM20ParserFoundationRecognizesJSXAndTSX(t *testing.T) {
	tests := []struct {
		name string
		file string
		source string
		wantRule string
	}{
		{
			name: "jsx simplify",
			file: "sample.jsx",
			source: "function decision(flag) {\n  if (flag) {\n    return true;\n  } else {\n    return false;\n  }\n}\nconst view = <div />;\n",
			wantRule: ruleSimplify,
		},
		{
			name: "tsx security",
			file: "sample.tsx",
			source: "const apiToken: string = \"hardcoded-production-token-12345\";\nexport function View(): JSX.Element { return <div />; }\n",
			wantRule: ruleSecurity,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, test.file), []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			findings, err := New().Analyze(context.Background(), root)
			if err != nil {
				t.Fatalf("Analyze(%s) error=%v", test.file, err)
			}
			for _, finding := range findings {
				if finding.RuleID == test.wantRule {
					return
				}
			}
			t.Fatalf("Analyze(%s) missing rule %s: %#v", test.file, test.wantRule, findings)
		})
	}
}

func TestM20ParserFoundationRejectsMalformedSyntax(t *testing.T) {
	tests := []struct {
		name string
		file string
		source string
	}{
		{name: "javascript unclosed block", file: "broken.js", source: "function broken(flag) { if (flag) { return true; }\n"},
		{name: "typescript missing type", file: "broken.ts", source: "const value: = 1;\n"},
		{name: "tsx malformed element", file: "broken.tsx", source: "export const view = <div>;\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, test.file), []byte(test.source), 0o600); err != nil {
				t.Fatal(err)
			}
			if findings, err := New().Analyze(context.Background(), root); err == nil {
				t.Fatalf("Analyze(%s) error=nil, want fail-closed syntax error; findings=%#v", test.file, findings)
			}
		})
	}
}
