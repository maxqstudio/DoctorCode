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

func TestPacketIncludesBoundedGoRelatedExcerpts(t *testing.T) {
	root := t.TempDir()
	source := "package demo\n\nfunc target(flag bool) bool {\n\tif flag {\n\t\treturn true\n\t} else {\n\t\treturn false\n\t}\n}\n\nfunc caller() bool { return target(true) }\n"
	testSource := "package demo\n\nimport \"testing\"\n\nfunc TestTarget(t *testing.T) {\n\t_ = target(false)\n}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatal(err)
	}

	finding := model.Finding{
		ID: "RELATED", RuleID: "GO-SIMPLIFY-BOOL-RETURN", Category: model.CategorySimplify,
		Severity: model.SeverityLow, Confidence: model.ConfidenceProven,
		Path: "sample.go", LineStart: 4, LineEnd: 8, Summary: "simplify",
	}
	packet, err := Build(root, finding, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if packet.SchemaVersion != 2 {
		t.Fatalf("expected context packet schema v2, got %d", packet.SchemaVersion)
	}
	if packet.RelatedTotal != 2 || len(packet.RelatedExcerpts) != 2 {
		t.Fatalf("expected two related excerpts, got total=%d excerpts=%#v", packet.RelatedTotal, packet.RelatedExcerpts)
	}
	if packet.RelatedExcerpts[0].Kind != "reference" || !strings.Contains(packet.RelatedExcerpts[0].Excerpt, "target(true)") {
		t.Fatalf("missing production reference excerpt: %#v", packet.RelatedExcerpts[0])
	}
	if packet.RelatedExcerpts[1].Kind != "test_reference" || !strings.Contains(packet.RelatedExcerpts[1].Excerpt, "target(false)") {
		t.Fatalf("missing test reference excerpt: %#v", packet.RelatedExcerpts[1])
	}
	if encodedSize(packet) > 4096 {
		t.Fatalf("packet exceeds budget: %d", encodedSize(packet))
	}
}

func TestPacketRelatedContextTruncatesInsideBudget(t *testing.T) {
	root := t.TempDir()
	source := "package demo\n\nfunc target() bool { return true }\n\nfunc a() bool { return target() }\nfunc b() bool { return target() }\nfunc c() bool { return target() }\nfunc d() bool { return target() }\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	finding := model.Finding{
		ID: "BUDGET", RuleID: "R", Category: model.CategoryLogic,
		Severity: model.SeverityMedium, Confidence: model.ConfidenceHigh,
		Path: "sample.go", LineStart: 3, LineEnd: 3, Summary: strings.Repeat("metadata-", 24),
	}
	packet, err := Build(root, finding, 900)
	if err != nil {
		t.Fatal(err)
	}
	if packet.RelatedTotal != 4 {
		t.Fatalf("expected all four related locations to be counted, got %d", packet.RelatedTotal)
	}
	if len(packet.RelatedExcerpts) >= packet.RelatedTotal {
		t.Fatalf("expected related excerpts to truncate under tight budget: %#v", packet.RelatedExcerpts)
	}
	if !packet.Truncated {
		t.Fatal("expected packet to mark truncation")
	}
	if encodedSize(packet) > 900 {
		t.Fatalf("packet exceeds budget: %d", encodedSize(packet))
	}
}

func TestPacketRejectsTraversalBeforeRelatedProviderReadsOutside(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.go")
	if err := os.WriteFile(outside, []byte("this is deliberately invalid go syntax"), 0o644); err != nil {
		t.Fatal(err)
	}
	finding := model.Finding{
		ID: "EARLY-BOUNDARY", RuleID: "R", Category: model.CategoryLogic,
		Severity: model.SeverityMedium, Confidence: model.ConfidenceHigh,
		Path: "../outside.go", LineStart: 1, LineEnd: 1, Summary: "escape",
	}
	_, err := Build(root, finding, 1024)
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if !strings.Contains(err.Error(), "repository root") {
		t.Fatalf("related provider read outside root before boundary rejection: %v", err)
	}
}
