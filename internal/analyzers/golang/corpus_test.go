package goanalysis

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCuratedCorpusExpectedRules(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(current), "testdata", "positive")
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []string{ruleDeadCode, ruleLogic, ruleSimplify, ruleSecurity, ruleBloat} {
		assertRule(t, findings, rule)
	}
}

func TestCuratedNegativeCorpusAvoidsTargetedFalsePositives(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(current), "testdata", "negative")
	findings, err := New().Analyze(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		switch finding.RuleID {
		case ruleLogic:
			t.Fatalf("side-effecting repeated condition must not be reported: %#v", finding)
		case ruleSecurity:
			t.Fatalf("obvious example credential must not be reported: %#v", finding)
		}
	}
}
