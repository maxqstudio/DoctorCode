package evidence

import (
	"os"
	"path/filepath"
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
