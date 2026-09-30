package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/maxqstudio/DoctorCode/internal/analyzers"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Engine struct {
	analyzers []detector.Analyzer
}

func Default() *Engine {
	return &Engine{
		analyzers: analyzers.Default(),
	}
}

func (e *Engine) Audit(ctx context.Context, root string) (model.AuditResult, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return model.AuditResult{}, err
	}

	result := model.AuditResult{Root: absoluteRoot}
	for _, analyzer := range e.analyzers {
		desc := analyzer.Descriptor()
		if desc.ID != analyzer.Name() {
			return model.AuditResult{}, fmt.Errorf("analyzer descriptor id %q does not match name %q", desc.ID, analyzer.Name())
		}
		if err := detector.ValidateDescriptor(desc); err != nil {
			return model.AuditResult{}, fmt.Errorf("analyzer %q descriptor: %w", analyzer.Name(), err)
		}
		recognized, err := detector.RecognizesRoot(ctx, absoluteRoot, desc)
		if err != nil {
			return model.AuditResult{}, fmt.Errorf("analyzer %q recognition: %w", analyzer.Name(), err)
		}
		if !recognized {
			continue
		}
		findings, analyzeErr := analyzer.Analyze(ctx, absoluteRoot)
		if analyzeErr != nil {
			if errors.Is(analyzeErr, detector.ErrUnavailable) {
				continue
			}
			return model.AuditResult{}, analyzeErr
		}
		if err := detector.ValidateFindings(desc, findings); err != nil {
			return model.AuditResult{}, fmt.Errorf("analyzer %q findings: %w", analyzer.Name(), err)
		}
		result.Analyzers = append(result.Analyzers, desc.ID)
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
