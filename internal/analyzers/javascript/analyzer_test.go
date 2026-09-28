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
	source := "function decision(flag) {\\n  if (flag) {\\n    return true;\\n  } else {\\n    return false;\\n  }\\n}\\n\\nconst apiToken = \\"hardcoded-production-token-12345\\";\\n"
	if err := os.WriteFile(filepath.Join(root, "sample.js"), []byte(source), 0o600); err != nil { t.Fatal(err) }
	findings, err := New().Analyze(context.Background(), root); if err != nil { t.Fatal(err) }
	if len(findings) != 2 { t.Fatalf("findings=%d want=2: %#v", len(findings), findings) }
	for _, finding := range findings { if finding.SafeAutofix { t.Fatal("JS/TS findings must not authorize autofix") }; for _, evidence := range finding.Evidence { if strings.Contains(evidence, "hardcoded-production-token-12345") { t.Fatal("secret leaked in evidence") } } }
}
