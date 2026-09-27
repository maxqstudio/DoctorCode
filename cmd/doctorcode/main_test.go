package main

import (
	"testing"

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
