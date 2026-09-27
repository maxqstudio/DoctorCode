package model

type Category string

const (
	CategoryBloat    Category = "BLOAT"
	CategorySecurity Category = "SECURITY"
	CategorySimplify Category = "SIMPLIFY"
	CategoryLogic    Category = "LOGIC"
	CategoryDeadCode Category = "DEADCODE"
)

type Confidence string

const (
	ConfidenceProven     Confidence = "PROVEN"
	ConfidenceHigh       Confidence = "HIGH"
	ConfidenceSuspicious Confidence = "SUSPICIOUS"
	ConfidenceUnknown    Confidence = "UNKNOWN"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

type Finding struct {
	ID           string     `json:"id"`
	RuleID       string     `json:"rule_id"`
	Category     Category   `json:"category"`
	Severity     Severity   `json:"severity"`
	Confidence   Confidence `json:"confidence"`
	Path         string     `json:"path"`
	LineStart    int        `json:"line_start,omitempty"`
	LineEnd      int        `json:"line_end,omitempty"`
	Summary      string     `json:"summary"`
	Evidence     []string   `json:"evidence,omitempty"`
	Verification []string   `json:"verification,omitempty"`
	SafeAutofix  bool       `json:"safe_autofix"`
}

type AuditResult struct {
	Root      string    `json:"root"`
	Analyzers []string  `json:"analyzers"`
	Findings  []Finding `json:"findings"`
}

type LanguageSummary struct {
	Name  string `json:"name"`
	Files int    `json:"files"`
}

type ScanResult struct {
	Root            string            `json:"root"`
	Files           int               `json:"files"`
	RecognizedFiles int               `json:"recognized_files"`
	Languages       []LanguageSummary `json:"languages"`
}

type ToolchainStatus struct {
	Language  string   `json:"language"`
	Tool      string   `json:"tool"`
	Available bool     `json:"available"`
	Path      string   `json:"path,omitempty"`
	Manifests []string `json:"manifests,omitempty"`
}
