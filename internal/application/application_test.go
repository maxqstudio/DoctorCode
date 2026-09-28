package application

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepairWorkflowUsesOneSharedApplicationAuthority(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	before := "package sample\n\nfunc decision(flag bool) bool {\n\tif flag {\n\t\treturn true\n\t} else {\n\t\treturn false\n\t}\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	audit, err := Audit(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	var findingID string
	for _, finding := range audit.Findings {
		if finding.RuleID == "GO-SIMPLIFY-BOOL-RETURN" {
			findingID = finding.ID
			break
		}
	}
	if findingID == "" {
		t.Fatalf("expected simplify finding, got %#v", audit.Findings)
	}

	packet, err := Context(ctx, root, findingID, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Finding.ID != findingID {
		t.Fatalf("context returned wrong finding: %s", packet.Finding.ID)
	}

	contract, err := Contract(ctx, root, findingID)
	if err != nil {
		t.Fatal(err)
	}
	if contract.TargetID != findingID || contract.TargetBaselineCount < 1 {
		t.Fatalf("unexpected contract: %#v", contract)
	}

	after := "package sample\n\nfunc decision(flag bool) bool {\n\treturn flag\n}\n"
	if err := os.WriteFile(path, []byte(after), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := Verify(ctx, root, contract)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed || !result.TargetResolved {
		t.Fatalf("expected verified repair, got %#v", result)
	}
}

func TestContextFailsClosedForUnknownFinding(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Context(context.Background(), root, "UNKNOWN-FINDING", 4096); err == nil {
		t.Fatal("unknown finding id must fail closed")
	}
}
