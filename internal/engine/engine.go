package engine

import (
	"context"
	"path/filepath"
	"sort"

	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Engine struct {
	analyzers []detector.Analyzer
}

func Default() *Engine {
	return &Engine{
		analyzers: []detector.Analyzer{
			goanalysis.New(),
		},
	}
}

func (e *Engine) Audit(ctx context.Context, root string) (model.AuditResult, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return model.AuditResult{}, err
	}

	result := model.AuditResult{Root: absoluteRoot}
	for _, analyzer := range e.analyzers {
		findings, analyzeErr := analyzer.Analyze(ctx, absoluteRoot)
		if analyzeErr != nil {
			return model.AuditResult{}, analyzeErr
		}
		result.Analyzers = append(result.Analyzers, analyzer.Name())
		result.Findings = append(result.Findings, findings...)
	}

	sort.Slice(result.Findings, func(i, j int) bool {
		left := result.Findings[i]
		right := result.Findings[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) < severityRank(right.Severity)
		}
		if confidenceRank(left.Confidence) != confidenceRank(right.Confidence) {
			return confidenceRank(left.Confidence) < confidenceRank(right.Confidence)
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.LineStart != right.LineStart {
			return left.LineStart < right.LineStart
		}
		return left.ID < right.ID
	})
	return result, nil
}

func severityRank(value model.Severity) int {
	switch value {
	case model.SeverityCritical:
		return 0
	case model.SeverityHigh:
		return 1
	case model.SeverityMedium:
		return 2
	case model.SeverityLow:
		return 3
	default:
		return 4
	}
}

func confidenceRank(value model.Confidence) int {
	switch value {
	case model.ConfidenceProven:
		return 0
	case model.ConfidenceHigh:
		return 1
	case model.ConfidenceSuspicious:
		return 2
	default:
		return 3
	}
}
