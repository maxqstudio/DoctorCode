package skilladapter

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skillPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "skills", "doctorcode", "SKILL.md"))
}

func TestDoctorCodeSkillContract(t *testing.T) {
	data, err := os.ReadFile(skillPath(t))
	if err != nil {
		t.Fatalf("read DoctorCode skill: %v", err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	required := []string{
		"---\nname: doctorcode\n",
		"description:",
		"Use when",
		"doctorcode audit",
		"doctorcode context",
		"doctorcode contract",
		"doctorcode verify",
		"safe_autofix",
		"repository-provided",
		"PASS",
	}
	for _, token := range required {
		if !strings.Contains(content, token) {
			t.Fatalf("skill missing required contract token %q", token)
		}
	}

	lines := strings.Count(content, "\n") + 1
	if lines > 500 {
		t.Fatalf("skill must stay context-efficient; got %d lines", lines)
	}
}

func TestDoctorCodeSkillDoesNotDuplicateProductLogic(t *testing.T) {
	data, err := os.ReadFile(skillPath(t))
	if err != nil {
		t.Fatalf("read DoctorCode skill: %v", err)
	}
	content := strings.ToLower(strings.ReplaceAll(string(data), "\r\n", "\n"))

	forbidden := []string{
		"implement your own detector",
		"reimplement the analyzer",
		"infer safe deletion",
		"run repository tests automatically",
	}
	for _, phrase := range forbidden {
		if strings.Contains(content, phrase) {
			t.Fatalf("skill contains forbidden duplicated-authority instruction %q", phrase)
		}
	}
}
