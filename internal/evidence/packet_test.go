package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestPacketHonorsByteBudget(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	content := "package demo\n\nfunc example() {\n\tprintln(\"hello\")\n}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	finding := model.Finding{
		ID: "X", RuleID: "R", Category: model.CategoryLogic,
		Severity: model.SeverityMedium, Confidence: model.ConfidenceHigh,
		Path: "sample.go", LineStart: 3, LineEnd: 5, Summary: "example",
	}
	packet, err := Build(root, finding, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if packet.SourceExcerpt == "" {
		t.Fatal("expected bounded source excerpt")
	}
	if encodedSize(packet) > 1024 {
		t.Fatalf("packet exceeds budget: %d", encodedSize(packet))
	}
}

func TestSecurityPacketOmitsSourceExcerpt(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "secret.go"), []byte("package demo\nconst apiToken = \"real-secret-value\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	finding := model.Finding{
		ID: "S", RuleID: "SEC", Category: model.CategorySecurity,
		Severity: model.SeverityHigh, Confidence: model.ConfidenceSuspicious,
		Path: "secret.go", LineStart: 2, LineEnd: 2, Summary: "credential",
	}
	packet, err := Build(root, finding, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if packet.SourceExcerpt != "" || !packet.SensitiveExcerptOmitted {
		t.Fatalf("security packet leaked or failed to mark omission: %#v", packet)
	}
}

func TestPacketRejectsPathTraversal(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.go")
	if err := os.WriteFile(outside, []byte("package outside\nconst secret = \"do-not-read\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	finding := model.Finding{
		ID: "ESCAPE", RuleID: "R", Category: model.CategoryLogic,
		Severity: model.SeverityMedium, Confidence: model.ConfidenceHigh,
		Path: "../outside.go", LineStart: 1, LineEnd: 2, Summary: "escape",
	}
	_, err := Build(root, finding, 1024)
	if err == nil {
		t.Fatal("expected path traversal outside repository root to be rejected")
	}
	if !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("unexpected traversal error: %v", err)
	}
}

func TestPacketRejectsSymlinkEscape(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.go")
	if err := os.WriteFile(outside, []byte("package outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}

	finding := model.Finding{
		ID: "LINK", RuleID: "R", Category: model.CategoryLogic,
		Severity: model.SeverityMedium, Confidence: model.ConfidenceHigh,
		Path: "link.go", LineStart: 1, LineEnd: 1, Summary: "link escape",
	}
	_, err := Build(root, finding, 1024)
	if err == nil {
		t.Fatal("expected symlink escape outside repository root to be rejected")
	}
	if !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("unexpected symlink error: %v", err)
	}
}
