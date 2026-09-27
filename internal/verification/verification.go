package verification

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

const SchemaVersion = 1

type FindingCount struct {
	RuleID   string         `json:"rule_id"`
	Path     string         `json:"path"`
	Summary  string         `json:"summary"`
	Severity model.Severity `json:"severity"`
	Count    int            `json:"count"`
}

type Contract struct {
	SchemaVersion       int            `json:"schema_version"`
	Analyzers           []string       `json:"analyzers"`
	TargetID            string         `json:"target_id"`
	TargetRuleID        string         `json:"target_rule_id"`
	TargetPath          string         `json:"target_path"`
	TargetSummary       string         `json:"target_summary"`
	TargetSeverity      model.Severity `json:"target_severity"`
	TargetBaselineCount int            `json:"target_baseline_count"`
	BaselineFindings    []FindingCount `json:"baseline_findings"`
}

type FindingDelta struct {
	RuleID        string         `json:"rule_id"`
	Path          string         `json:"path"`
	Summary       string         `json:"summary"`
	Severity      model.Severity `json:"severity"`
	BaselineCount int            `json:"baseline_count"`
	CurrentCount  int            `json:"current_count"`
}

type Result struct {
	SchemaVersion        int            `json:"schema_version"`
	Passed               bool           `json:"passed"`
	TargetResolved       bool           `json:"target_resolved"`
	TargetBaselineCount  int            `json:"target_baseline_count"`
	TargetCurrentCount   int            `json:"target_current_count"`
	NewBlockingFindings  []FindingDelta `json:"new_blocking_findings,omitempty"`
}

type semanticKey struct {
	ruleID  string
	path    string
	summary string
}

func BuildContract(audit model.AuditResult, findingID string) (Contract, error) {
	var target model.Finding
	found := false
	for _, finding := range audit.Findings {
		if finding.ID == findingID {
			target = finding
			found = true
			break
		}
	}
	if !found {
		return Contract{}, fmt.Errorf("finding %q not found", findingID)
	}
	if strings.TrimSpace(target.RuleID) == "" || strings.TrimSpace(target.Path) == "" || strings.TrimSpace(target.Summary) == "" {
		return Contract{}, errors.New("target finding lacks stable verification fields")
	}

	counts := countFindingsOnPath(audit.Findings, target.Path)
	targetKey := keyFor(target)
	targetCount := counts[targetKey].Count
	if targetCount < 1 {
		return Contract{}, errors.New("target semantic key is absent from baseline")
	}

	baseline := make([]FindingCount, 0, len(counts))
	for _, item := range counts {
		baseline = append(baseline, item)
	}
	sortFindingCounts(baseline)

	return Contract{
		SchemaVersion:       SchemaVersion,
		Analyzers:           normalizedAnalyzers(audit.Analyzers),
		TargetID:            target.ID,
		TargetRuleID:        target.RuleID,
		TargetPath:          target.Path,
		TargetSummary:       target.Summary,
		TargetSeverity:      target.Severity,
		TargetBaselineCount: targetCount,
		BaselineFindings:    baseline,
	}, nil
}

func Verify(contract Contract, current model.AuditResult) (Result, error) {
	if err := validateContract(contract); err != nil {
		return Result{}, err
	}
	if !equalStrings(contract.Analyzers, normalizedAnalyzers(current.Analyzers)) {
		return Result{}, fmt.Errorf(
			"verification analyzer set changed: baseline=%v current=%v",
			contract.Analyzers,
			normalizedAnalyzers(current.Analyzers),
		)
	}

	baseline := make(map[semanticKey]FindingCount, len(contract.BaselineFindings))
	for _, item := range contract.BaselineFindings {
		key := semanticKey{ruleID: item.RuleID, path: item.Path, summary: item.Summary}
		if key.path != contract.TargetPath {
			return Result{}, fmt.Errorf("verification contract baseline escapes target path: %s", key.path)
		}
		if item.Count < 1 {
			return Result{}, errors.New("verification contract contains non-positive baseline count")
		}
		if _, exists := baseline[key]; exists {
			return Result{}, errors.New("verification contract contains duplicate semantic keys")
		}
		baseline[key] = item
	}

	targetKey := semanticKey{
		ruleID:  contract.TargetRuleID,
		path:    contract.TargetPath,
		summary: contract.TargetSummary,
	}
	targetBaseline, ok := baseline[targetKey]
	if !ok {
		return Result{}, errors.New("verification contract baseline does not contain target semantic key")
	}
	if targetBaseline.Count != contract.TargetBaselineCount {
		return Result{}, fmt.Errorf(
			"verification contract target baseline mismatch: metadata=%d baseline=%d",
			contract.TargetBaselineCount,
			targetBaseline.Count,
		)
	}
	if targetBaseline.Severity != contract.TargetSeverity {
		return Result{}, fmt.Errorf(
			"verification contract target severity mismatch: metadata=%s baseline=%s",
			contract.TargetSeverity,
			targetBaseline.Severity,
		)
	}

	currentCounts := countFindingsOnPath(current.Findings, contract.TargetPath)
	targetCurrent := 0
	if item, ok := currentCounts[targetKey]; ok {
		targetCurrent = item.Count
	}
	targetResolved := targetCurrent < contract.TargetBaselineCount

	var blocking []FindingDelta
	for key, item := range currentCounts {
		base := baseline[key]
		if item.Count <= base.Count {
			continue
		}
		if severityRank(item.Severity) < severityRank(contract.TargetSeverity) {
			continue
		}
		blocking = append(blocking, FindingDelta{
			RuleID:        item.RuleID,
			Path:          item.Path,
			Summary:       item.Summary,
			Severity:      item.Severity,
			BaselineCount: base.Count,
			CurrentCount:  item.Count,
		})
	}
	sort.Slice(blocking, func(i, j int) bool {
		if blocking[i].RuleID != blocking[j].RuleID {
			return blocking[i].RuleID < blocking[j].RuleID
		}
		if blocking[i].Summary != blocking[j].Summary {
			return blocking[i].Summary < blocking[j].Summary
		}
		return blocking[i].Severity < blocking[j].Severity
	})

	return Result{
		SchemaVersion:       SchemaVersion,
		Passed:              targetResolved && len(blocking) == 0,
		TargetResolved:      targetResolved,
		TargetBaselineCount: contract.TargetBaselineCount,
		TargetCurrentCount:  targetCurrent,
		NewBlockingFindings: blocking,
	}, nil
}

func validateContract(contract Contract) error {
	if contract.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported verification contract schema %d", contract.SchemaVersion)
	}
	if len(contract.Analyzers) == 0 {
		return errors.New("verification contract analyzer set is empty")
	}
	if strings.TrimSpace(contract.TargetID) == "" ||
		strings.TrimSpace(contract.TargetRuleID) == "" ||
		strings.TrimSpace(contract.TargetPath) == "" ||
		strings.TrimSpace(contract.TargetSummary) == "" {
		return errors.New("verification contract target is incomplete")
	}
	if contract.TargetBaselineCount < 1 {
		return errors.New("verification contract target baseline count must be positive")
	}
	return nil
}

func countFindingsOnPath(findings []model.Finding, path string) map[semanticKey]FindingCount {
	out := map[semanticKey]FindingCount{}
	for _, finding := range findings {
		if finding.Path != path {
			continue
		}
		key := keyFor(finding)
		item := out[key]
		if item.Count == 0 {
			item = FindingCount{
				RuleID:   finding.RuleID,
				Path:     finding.Path,
				Summary:  finding.Summary,
				Severity: finding.Severity,
			}
		}
		item.Count++
		if severityRank(finding.Severity) > severityRank(item.Severity) {
			item.Severity = finding.Severity
		}
		out[key] = item
	}
	return out
}

func keyFor(finding model.Finding) semanticKey {
	return semanticKey{
		ruleID:  finding.RuleID,
		path:    finding.Path,
		summary: finding.Summary,
	}
}

func sortFindingCounts(items []FindingCount) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].RuleID != items[j].RuleID {
			return items[i].RuleID < items[j].RuleID
		}
		if items[i].Path != items[j].Path {
			return items[i].Path < items[j].Path
		}
		if items[i].Summary != items[j].Summary {
			return items[i].Summary < items[j].Summary
		}
		return items[i].Severity < items[j].Severity
	})
}

func normalizedAnalyzers(items []string) []string {
	out := append([]string(nil), items...)
	sort.Strings(out)
	return out
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func severityRank(severity model.Severity) int {
	switch severity {
	case model.SeverityCritical:
		return 5
	case model.SeverityHigh:
		return 4
	case model.SeverityMedium:
		return 3
	case model.SeverityLow:
		return 2
	case model.SeverityInfo:
		return 1
	default:
		return 0
	}
}
