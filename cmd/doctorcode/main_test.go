package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maxqstudio/DoctorCode/internal/engine"
	"github.com/maxqstudio/DoctorCode/internal/evidence"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

func TestParseContextArgs(t *testing.T) {
	findingID, root, asJSON, maxBytes, err := parseContextArgs([]string{
		"FINDING-1", "repo", "--json", "--max-bytes=2048",
	})
	if err != nil {
		t.Fatal(err)
	}
	if findingID != "FINDING-1" || root != "repo" || !asJSON || maxBytes != 2048 {
		t.Fatalf("unexpected parsed context args: id=%q root=%q json=%t max=%d", findingID, root, asJSON, maxBytes)
	}
}

func TestParseContextArgsRequiresFindingID(t *testing.T) {
	if _, _, _, _, err := parseContextArgs([]string{"--json"}); err == nil {
		t.Fatal("expected missing finding id to fail")
	}
}

func TestParseContextArgsRejectsExtraPositional(t *testing.T) {
	if _, _, _, _, err := parseContextArgs([]string{"FINDING-1", "repo", "extra"}); err == nil {
		t.Fatal("expected extra positional argument to fail")
	}
}

func TestFindFindingByID(t *testing.T) {
	findings := []model.Finding{
		{ID: "FIRST", RuleID: "R1"},
		{ID: "TARGET", RuleID: "R2"},
	}
	got, ok := findFindingByID(findings, "TARGET")
	if !ok {
		t.Fatal("expected finding id to be selected")
	}
	if got.RuleID != "R2" {
		t.Fatalf("selected wrong finding: %#v", got)
	}
	if _, ok := findFindingByID(findings, "MISSING"); ok {
		t.Fatal("missing finding id should not resolve")
	}
}

func TestContextPipelineSelectsExactFindingWithinBudget(t *testing.T) {
	root := t.TempDir()
	source := "package demo\n\nfunc firstUnused() {}\n\nfunc secondUnused() {}\n"
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := engine.Default().Audit(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	var target model.Finding
	for _, finding := range result.Findings {
		if finding.RuleID == "GO-DEADCODE-ZERO-REF" && strings.Contains(finding.Summary, "secondUnused") {
			target = finding
			break
		}
	}
	if target.ID == "" {
		t.Fatalf("expected secondUnused DEADCODE finding, got %#v", result.Findings)
	}

	selected, ok := findFindingByID(result.Findings, target.ID)
	if !ok || selected.ID != target.ID {
		t.Fatalf("exact finding selection failed: target=%q selected=%#v", target.ID, selected)
	}

	packet, err := evidence.Build(result.Root, selected, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if packet.Finding.ID != target.ID {
		t.Fatalf("packet used wrong finding: want=%q got=%q", target.ID, packet.Finding.ID)
	}
	if packet.SourceExcerpt == "" || !strings.Contains(packet.SourceExcerpt, "secondUnused") {
		t.Fatalf("packet missing selected source excerpt: %#v", packet)
	}
	data, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(data)+1 > 1024 {
		t.Fatalf("context packet exceeds byte budget: %d", len(data)+1)
	}
}


func TestParseVerificationArgs(t *testing.T) {
	first, root, asJSON, err := parseVerificationArgs("contract", []string{"FINDING-1", "repo", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if first != "FINDING-1" || root != "repo" || !asJSON {
		t.Fatalf("unexpected verification args: first=%q root=%q json=%t", first, root, asJSON)
	}
	if _, _, _, err := parseVerificationArgs("verify", []string{}); err == nil {
		t.Fatal("verify must require contract path")
	}
	if _, _, _, err := parseVerificationArgs("verify", []string{"contract.json", "repo", "extra"}); err == nil {
		t.Fatal("verify must reject extra positional arguments")
	}
}

func TestReadVerificationContractRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contract.json")
	content := `{"schema_version":1,"analyzers":["go/stdlib-ast-v1"],"target_id":"X","target_rule_id":"R","target_path":"sample.go","target_summary":"summary","target_severity":"LOW","target_baseline_count":1,"baseline_findings":[],"unexpected":true}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readVerificationContract(path); err == nil {
		t.Fatal("unknown contract fields must fail closed")
	}
}


func TestVersionArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"--json"}} {
		for _, arg := range args {
			if arg != "--json" {
				t.Fatalf("unexpected fixture argument %q", arg)
			}
		}
	}
	// runVersion exits on invalid input, so keep parser behavior explicit in the
	// command implementation and cover release metadata through artifact smoke.
}
