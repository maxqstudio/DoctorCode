package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

type Thresholds struct {
	MinPrecision float64 `json:"min_precision"`
	MinRecall    float64 `json:"min_recall"`
}

type ExpectedFinding struct {
	RuleID          string `json:"rule_id"`
	Path            string `json:"path"`
	LineStart       int    `json:"line_start,omitempty"`
	SummaryContains string `json:"summary_contains,omitempty"`
}

type Case struct {
	Name     string            `json:"name"`
	Root     string            `json:"root"`
	Expected []ExpectedFinding `json:"expected"`
}

type Manifest struct {
	SchemaVersion int        `json:"schema_version"`
	Language      string     `json:"language"`
	Thresholds    Thresholds `json:"thresholds"`
	Cases         []Case     `json:"cases"`
}

type Metrics struct {
	TruePositive  int     `json:"true_positive"`
	FalsePositive int     `json:"false_positive"`
	FalseNegative int     `json:"false_negative"`
	Precision     float64 `json:"precision"`
	Recall        float64 `json:"recall"`
}

type Report struct {
	SchemaVersion int                `json:"schema_version"`
	Manifest      string             `json:"manifest"`
	Analyzer      string             `json:"analyzer"`
	Language      string             `json:"language"`
	Cases         int                `json:"cases"`
	Metrics       Metrics            `json:"metrics"`
	Rules         map[string]Metrics `json:"rules"`
	Failures      []string           `json:"failures,omitempty"`
	Passed        bool               `json:"passed"`
}

type counts struct {
	tp int
	fp int
	fn int
}

func Evaluate(ctx context.Context, manifestPath string, analyzer detector.Analyzer) (Report, error) {
	if analyzer == nil {
		return Report{}, errors.New("analyzer is required")
	}

	manifest, manifestAbs, err := loadManifest(manifestPath)
	if err != nil {
		return Report{}, err
	}
	desc := analyzer.Descriptor()
	if desc.ID != analyzer.Name() {
		return Report{}, fmt.Errorf("analyzer descriptor id %q does not match name %q", desc.ID, analyzer.Name())
	}
	if err := detector.ValidateDescriptor(desc); err != nil {
		return Report{}, fmt.Errorf("analyzer descriptor: %w", err)
	}
	if !detector.SupportsBenchmarkLanguage(desc, manifest.Language) {
		return Report{}, fmt.Errorf("benchmark language %q is not declared by analyzer %q", manifest.Language, desc.ID)
	}

	report := Report{
		SchemaVersion: 1,
		Manifest:      filepath.ToSlash(manifestAbs),
		Analyzer:      analyzer.Name(),
		Language:      manifest.Language,
		Cases:         len(manifest.Cases),
		Rules:         map[string]Metrics{},
	}
	baseDir := filepath.Dir(manifestAbs)
	byRule := map[string]*counts{}
	total := counts{}

	for _, benchmarkCase := range manifest.Cases {
		caseRoot, err := secureCaseRoot(baseDir, benchmarkCase.Root)
		if err != nil {
			return Report{}, fmt.Errorf("case %q: %w", benchmarkCase.Name, err)
		}

		actual, err := analyzer.Analyze(ctx, caseRoot)
		if err != nil {
			return Report{}, fmt.Errorf("case %q analyze: %w", benchmarkCase.Name, err)
		}
		if err := detector.ValidateFindings(desc, actual); err != nil {
			return Report{}, fmt.Errorf("case %q findings: %w", benchmarkCase.Name, err)
		}
		sortFindings(actual)

		matched := make([]bool, len(actual))
		for _, expected := range benchmarkCase.Expected {
			ruleCounts := ensureCounts(byRule, expected.RuleID)
			index := matchExpected(expected, actual, matched)
			if index >= 0 {
				matched[index] = true
				ruleCounts.tp++
				total.tp++
				continue
			}
			ruleCounts.fn++
			total.fn++
			report.Failures = append(report.Failures,
				fmt.Sprintf("%s: missing %s at %s:%d", benchmarkCase.Name, expected.RuleID, expected.Path, expected.LineStart))
		}

		for index, finding := range actual {
			if matched[index] {
				continue
			}
			ruleCounts := ensureCounts(byRule, finding.RuleID)
			ruleCounts.fp++
			total.fp++
			report.Failures = append(report.Failures,
				fmt.Sprintf("%s: unexpected %s at %s:%d", benchmarkCase.Name, finding.RuleID, finding.Path, finding.LineStart))
		}
	}

	report.Metrics = calculateMetrics(total)
	for rule, value := range byRule {
		report.Rules[rule] = calculateMetrics(*value)
	}

	for rule, value := range report.Rules {
		if value.Precision < manifest.Thresholds.MinPrecision {
			report.Failures = append(report.Failures,
				fmt.Sprintf("%s precision %.4f below %.4f", rule, value.Precision, manifest.Thresholds.MinPrecision))
		}
		if value.Recall < manifest.Thresholds.MinRecall {
			report.Failures = append(report.Failures,
				fmt.Sprintf("%s recall %.4f below %.4f", rule, value.Recall, manifest.Thresholds.MinRecall))
		}
	}
	if report.Metrics.Precision < manifest.Thresholds.MinPrecision {
		report.Failures = append(report.Failures,
			fmt.Sprintf("aggregate precision %.4f below %.4f", report.Metrics.Precision, manifest.Thresholds.MinPrecision))
	}
	if report.Metrics.Recall < manifest.Thresholds.MinRecall {
		report.Failures = append(report.Failures,
			fmt.Sprintf("aggregate recall %.4f below %.4f", report.Metrics.Recall, manifest.Thresholds.MinRecall))
	}

	sort.Strings(report.Failures)
	report.Passed = len(report.Failures) == 0
	return report, nil
}

func loadManifest(path string) (Manifest, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return Manifest{}, "", err
	}
	data, err := os.ReadFile(absolute)
	if err != nil {
		return Manifest{}, "", err
	}

	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, "", fmt.Errorf("decode manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Manifest{}, "", errors.New("decode manifest: trailing JSON content")
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, "", err
	}
	return manifest, absolute, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("unsupported schema_version=%d", manifest.SchemaVersion)
	}
	if strings.TrimSpace(manifest.Language) == "" {
		return errors.New("language is required")
	}
	if manifest.Thresholds.MinPrecision < 0 || manifest.Thresholds.MinPrecision > 1 {
		return errors.New("min_precision must be between 0 and 1")
	}
	if manifest.Thresholds.MinRecall < 0 || manifest.Thresholds.MinRecall > 1 {
		return errors.New("min_recall must be between 0 and 1")
	}
	if len(manifest.Cases) == 0 {
		return errors.New("at least one case is required")
	}

	names := map[string]bool{}
	for _, benchmarkCase := range manifest.Cases {
		if strings.TrimSpace(benchmarkCase.Name) == "" {
			return errors.New("case name is required")
		}
		if names[benchmarkCase.Name] {
			return fmt.Errorf("duplicate case name %q", benchmarkCase.Name)
		}
		names[benchmarkCase.Name] = true
		if strings.TrimSpace(benchmarkCase.Root) == "" {
			return fmt.Errorf("case %q root is required", benchmarkCase.Name)
		}
		for _, expected := range benchmarkCase.Expected {
			if strings.TrimSpace(expected.RuleID) == "" || strings.TrimSpace(expected.Path) == "" {
				return fmt.Errorf("case %q expected finding requires rule_id and path", benchmarkCase.Name)
			}
		}
	}
	return nil
}

func secureCaseRoot(baseDir, relative string) (string, error) {
	if filepath.IsAbs(relative) {
		return "", errors.New("case root must be relative to the manifest")
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("case root escapes the manifest directory")
	}

	baseReal, err := filepath.EvalSymlinks(baseDir)
	if err != nil {
		return "", err
	}
	rootReal, err := filepath.EvalSymlinks(filepath.Join(baseDir, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(baseReal, rootReal)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("case root resolves outside the manifest directory")
	}
	info, err := os.Stat(rootReal)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("case root is not a directory")
	}
	return rootReal, nil
}

func matchExpected(expected ExpectedFinding, actual []model.Finding, matched []bool) int {
	for index, finding := range actual {
		if matched[index] || finding.RuleID != expected.RuleID {
			continue
		}
		if filepath.ToSlash(finding.Path) != filepath.ToSlash(expected.Path) {
			continue
		}
		if expected.LineStart > 0 && finding.LineStart != expected.LineStart {
			continue
		}
		if expected.SummaryContains != "" && !strings.Contains(finding.Summary, expected.SummaryContains) {
			continue
		}
		return index
	}
	return -1
}

func sortFindings(findings []model.Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].LineStart != findings[j].LineStart {
			return findings[i].LineStart < findings[j].LineStart
		}
		return findings[i].Summary < findings[j].Summary
	})
}

func ensureCounts(values map[string]*counts, rule string) *counts {
	value := values[rule]
	if value == nil {
		value = &counts{}
		values[rule] = value
	}
	return value
}

func calculateMetrics(value counts) Metrics {
	precision := 1.0
	if value.tp+value.fp > 0 {
		precision = float64(value.tp) / float64(value.tp+value.fp)
	}
	recall := 1.0
	if value.tp+value.fn > 0 {
		recall = float64(value.tp) / float64(value.tp+value.fn)
	}
	return Metrics{
		TruePositive: value.tp,
		FalsePositive: value.fp,
		FalseNegative: value.fn,
		Precision: precision,
		Recall: recall,
	}
}
