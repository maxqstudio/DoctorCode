package doctorcodemcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerExposesOnlyBoundedDoctorCodeTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	session := connectTestClient(t, root)
	defer session.Close()

	var names []string
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Name)
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		schema := strings.ToLower(string(raw))
		for _, forbidden := range []string{`"root"`, `"path"`, `"command"`, `"shell"`} {
			if strings.Contains(schema, forbidden) {
				t.Fatalf("tool %s exposes forbidden transport authority %s in schema: %s", tool.Name, forbidden, schema)
			}
		}
	}
	sort.Strings(names)
	want := []string{"doctorcode_audit", "doctorcode_context", "doctorcode_contract", "doctorcode_verify"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected MCP tool surface: got=%v want=%v", names, want)
	}
}

func TestMCPRepairWorkflowUsesBoundStartupRoot(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "sample.go")
	before := "package sample\n\nfunc decision(flag bool) bool {\n\tif flag {\n\t\treturn true\n\t} else {\n\t\treturn false\n\t}\n}\n"
	if err := os.WriteFile(sourcePath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	session := connectTestClient(t, root)
	defer session.Close()

	audit := callTool(t, session, "doctorcode_audit", map[string]any{"max_findings": 20})
	if audit.IsError {
		t.Fatalf("audit tool error: %#v", audit.Content)
	}
	var auditOut struct {
		Findings []struct {
			ID     string `json:"id"`
			RuleID string `json:"rule_id"`
		} `json:"findings"`
	}
	decodeStructured(t, audit.StructuredContent, &auditOut)
	var findingID string
	for _, finding := range auditOut.Findings {
		if finding.RuleID == "GO-SIMPLIFY-BOOL-RETURN" {
			findingID = finding.ID
			break
		}
	}
	if findingID == "" {
		t.Fatalf("expected simplify finding: %#v", auditOut)
	}

	contextResult := callTool(t, session, "doctorcode_context", map[string]any{
		"finding_id": findingID,
		"max_bytes":  4096,
	})
	if contextResult.IsError {
		t.Fatalf("context tool error: %#v", contextResult.Content)
	}

	contractResult := callTool(t, session, "doctorcode_contract", map[string]any{"finding_id": findingID})
	if contractResult.IsError {
		t.Fatalf("contract tool error: %#v", contractResult.Content)
	}

	after := "package sample\n\nfunc decision(flag bool) bool {\n\treturn flag\n}\n"
	if err := os.WriteFile(sourcePath, []byte(after), 0o644); err != nil {
		t.Fatal(err)
	}
	verifyResult := callTool(t, session, "doctorcode_verify", map[string]any{
		"contract": contractResult.StructuredContent,
	})
	if verifyResult.IsError {
		t.Fatalf("clean repair should verify: %#v", verifyResult.Content)
	}
	var verified struct {
		Passed         bool `json:"passed"`
		TargetResolved bool `json:"target_resolved"`
	}
	decodeStructured(t, verifyResult.StructuredContent, &verified)
	if !verified.Passed || !verified.TargetResolved {
		t.Fatalf("unexpected verify output: %#v", verified)
	}
}

func TestMCPVerifyMarksDomainFailureAsToolError(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "sample.go")
	before := "package sample\n\nfunc decision(flag bool) bool {\n\tif flag {\n\t\treturn true\n\t} else {\n\t\treturn false\n\t}\n}\n"
	if err := os.WriteFile(sourcePath, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	session := connectTestClient(t, root)
	defer session.Close()

	audit := callTool(t, session, "doctorcode_audit", map[string]any{"max_findings": 20})
	var auditOut struct {
		Findings []struct {
			ID     string `json:"id"`
			RuleID string `json:"rule_id"`
		} `json:"findings"`
	}
	decodeStructured(t, audit.StructuredContent, &auditOut)
	var findingID string
	for _, finding := range auditOut.Findings {
		if finding.RuleID == "GO-SIMPLIFY-BOOL-RETURN" {
			findingID = finding.ID
			break
		}
	}
	if findingID == "" {
		t.Fatal("missing simplify finding")
	}
	contractResult := callTool(t, session, "doctorcode_contract", map[string]any{"finding_id": findingID})

	regression := "package sample\n\nconst apiToken = \"hardcoded-production-token-12345\"\n\nfunc decision(flag bool) bool { return flag }\n"
	if err := os.WriteFile(sourcePath, []byte(regression), 0o644); err != nil {
		t.Fatal(err)
	}
	verifyResult := callTool(t, session, "doctorcode_verify", map[string]any{
		"contract": contractResult.StructuredContent,
	})
	if !verifyResult.IsError {
		t.Fatalf("verification domain failure must be visible as MCP tool error: %#v", verifyResult)
	}
	var failed struct {
		Passed              bool  `json:"passed"`
		NewBlockingFindings []any `json:"new_blocking_findings"`
	}
	decodeStructured(t, verifyResult.StructuredContent, &failed)
	if failed.Passed || len(failed.NewBlockingFindings) != 1 {
		t.Fatalf("unexpected failed verification output: %#v", failed)
	}
}

func TestNewServerRejectsInvalidRoot(t *testing.T) {
	if _, err := NewServer(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing startup root must fail closed")
	}
}

func connectTestClient(t *testing.T, root string) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	server, err := NewServer(root)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "doctorcode-mcp-test", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s protocol call failed: %v", name, err)
	}
	return result
}

func decodeStructured(t *testing.T, value any, target any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}
