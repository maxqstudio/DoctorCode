package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	cppanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/cpp"
	goanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/golang"
	javascriptanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/javascript"
	jvmanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/jvm"
	pythonanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/python"
	rustanalysis "github.com/maxqstudio/DoctorCode/internal/analyzers/rust"
	"github.com/maxqstudio/DoctorCode/internal/application"
	"github.com/maxqstudio/DoctorCode/internal/benchmark"
	"github.com/maxqstudio/DoctorCode/internal/buildinfo"
	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/engine"
	"github.com/maxqstudio/DoctorCode/internal/evidence"
	"github.com/maxqstudio/DoctorCode/internal/model"
	"github.com/maxqstudio/DoctorCode/internal/scanner"
	"github.com/maxqstudio/DoctorCode/internal/toolchain"
	"github.com/maxqstudio/DoctorCode/internal/verification"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "toolchains":
		runToolchains(os.Args[2:])
	case "audit":
		runAudit(os.Args[2:])
	case "next":
		runNext(os.Args[2:])
	case "context":
		runContext(os.Args[2:])
	case "contract":
		runContract(os.Args[2:])
	case "verify":
		runVerify(os.Args[2:])
	case "benchmark":
		runBenchmark(os.Args[2:])
	case "version", "--version", "-v":
		runVersion(os.Args[2:])
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func runVersion(args []string) {
	asJSON := false
	for _, arg := range args {
		switch arg {
		case "--json":
			asJSON = true
		default:
			die(fmt.Errorf("version accepts only --json, got %s", arg))
		}
	}
	info := buildinfo.Current()
	if asJSON {
		writeJSON(info)
		return
	}
	fmt.Printf("doctorcode %s commit=%s built=%s\n", info.Version, info.Commit, info.BuildDate)
}

func runScan(args []string) {
	root, asJSON, _, err := parseArgs(args, 0)
	if err != nil {
		die(err)
	}
	result, err := scanner.Scan(root)
	if err != nil {
		die(fmt.Errorf("scan failed: %w", err))
	}

	if asJSON {
		writeJSON(result)
		return
	}

	fmt.Printf("ROOT %s\n", result.Root)
	fmt.Printf("FILES %d\n", result.Files)
	fmt.Printf("RECOGNIZED %d\n", result.RecognizedFiles)
	for _, item := range result.Languages {
		fmt.Printf("%s %d\n", item.Name, item.Files)
	}
}

func runToolchains(args []string) {
	root, asJSON, _, err := parseArgs(args, 0)
	if err != nil {
		die(err)
	}
	result := toolchain.Detect(root)
	if asJSON {
		writeJSON(result)
		return
	}

	for _, item := range result {
		state := "MISSING"
		if item.Available {
			state = "AVAILABLE"
		}
		fmt.Printf("%s %s %s", item.Language, state, item.Tool)
		if item.Path != "" {
			fmt.Printf(" %s", item.Path)
		}
		if len(item.Manifests) > 0 {
			fmt.Printf(" manifests=%s", strings.Join(item.Manifests, ","))
		}
		fmt.Println()
	}
}

func runAudit(args []string) {
	root, asJSON, maxFindings, err := parseArgs(args, 100)
	if err != nil {
		die(err)
	}
	result, err := application.Audit(context.Background(), root)
	if err != nil {
		die(fmt.Errorf("audit failed: %w", err))
	}
	total := len(result.Findings)
	if maxFindings > 0 && len(result.Findings) > maxFindings {
		result.Findings = result.Findings[:maxFindings]
	}

	if asJSON {
		writeJSON(struct {
			Root             string      `json:"root"`
			Analyzers        []string    `json:"analyzers"`
			TotalFindings    int         `json:"total_findings"`
			ReturnedFindings int         `json:"returned_findings"`
			Findings         interface{} `json:"findings"`
		}{
			Root: result.Root, Analyzers: result.Analyzers,
			TotalFindings: total, ReturnedFindings: len(result.Findings), Findings: result.Findings,
		})
		return
	}

	fmt.Printf("ROOT %s\nANALYZERS %s\nFINDINGS %d", result.Root, strings.Join(result.Analyzers, ","), total)
	if len(result.Findings) != total {
		fmt.Printf(" returned=%d", len(result.Findings))
	}
	fmt.Println()
	for _, finding := range result.Findings {
		fmt.Printf("%s %s %s %s:%d %s\n", finding.ID, finding.Category, finding.Confidence, finding.Path, finding.LineStart, finding.Summary)
	}
}

func runNext(args []string) {
	root := "."
	asJSON := false
	maxBytes := 4096
	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--max-bytes="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--max-bytes="))
			if err != nil {
				die(fmt.Errorf("invalid --max-bytes: %w", err))
			}
			maxBytes = value
		case strings.HasPrefix(arg, "-"):
			die(fmt.Errorf("unknown option %s", arg))
		default:
			root = arg
		}
	}

	result, err := engine.Default().Audit(context.Background(), root)
	if err != nil {
		die(fmt.Errorf("audit failed: %w", err))
	}
	if len(result.Findings) == 0 {
		if asJSON {
			writeJSON(map[string]string{"status": "NO_FINDINGS"})
		} else {
			fmt.Println("NO_FINDINGS")
		}
		return
	}
	packet, err := evidence.Build(result.Root, result.Findings[0], maxBytes)
	if err != nil {
		die(fmt.Errorf("build evidence packet: %w", err))
	}
	if asJSON {
		writeJSON(packet)
		return
	}
	printEvidencePacket(packet)
}

func printEvidencePacket(packet evidence.Packet) {
	fmt.Printf("%s %s %s %s:%d\n%s\n", packet.Finding.ID, packet.Finding.Category, packet.Finding.Confidence, packet.Finding.Path, packet.Finding.LineStart, packet.Finding.Summary)
	for _, item := range packet.Finding.Evidence {
		fmt.Printf("EVIDENCE %s\n", item)
	}
	if packet.SourceExcerpt != "" {
		fmt.Printf("SOURCE\n%s\n", packet.SourceExcerpt)
	}
	for _, item := range packet.RelatedExcerpts {
		fmt.Printf("RELATED %s %s:%d-%d\n%s\n", item.Kind, item.Path, item.LineStart, item.LineEnd, item.Excerpt)
	}
	if packet.RelatedTotal > len(packet.RelatedExcerpts) {
		fmt.Printf("RELATED OMITTED %d\n", packet.RelatedTotal-len(packet.RelatedExcerpts))
	}
	if packet.SensitiveExcerptOmitted {
		fmt.Println("SOURCE OMITTED_SENSITIVE")
	}
}

func runContext(args []string) {
	findingID, root, asJSON, maxBytes, err := parseContextArgs(args)
	if err != nil {
		die(err)
	}

	packet, err := application.Context(context.Background(), root, findingID, maxBytes)
	if err != nil {
		die(err)
	}
	if asJSON {
		writeJSON(packet)
		return
	}
	fmt.Printf("%s %s %s %s:%d\n%s\n", packet.Finding.ID, packet.Finding.Category, packet.Finding.Confidence, packet.Finding.Path, packet.Finding.LineStart, packet.Finding.Summary)
	for _, item := range packet.Finding.Evidence {
		fmt.Printf("EVIDENCE %s\n", item)
	}
	if packet.SourceExcerpt != "" {
		fmt.Printf("SOURCE\n%s\n", packet.SourceExcerpt)
	}
	if packet.SensitiveExcerptOmitted {
		fmt.Println("SOURCE OMITTED_SENSITIVE")
	}
}

func parseContextArgs(args []string) (findingID, root string, asJSON bool, maxBytes int, err error) {
	root = "."
	maxBytes = 4096
	positionals := make([]string, 0, 2)

	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--max-bytes="):
			value, parseErr := strconv.Atoi(strings.TrimPrefix(arg, "--max-bytes="))
			if parseErr != nil {
				return "", "", false, 0, fmt.Errorf("invalid --max-bytes: %w", parseErr)
			}
			maxBytes = value
		case strings.HasPrefix(arg, "-"):
			return "", "", false, 0, fmt.Errorf("unknown option %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}

	if len(positionals) == 0 {
		return "", "", false, 0, errors.New("context finding id is required")
	}
	if len(positionals) > 2 {
		return "", "", false, 0, errors.New("context accepts a finding id and optional repository path")
	}
	findingID = positionals[0]
	if len(positionals) == 2 {
		root = positionals[1]
	}
	return findingID, root, asJSON, maxBytes, nil
}

func findFindingByID(findings []model.Finding, findingID string) (model.Finding, bool) {
	for _, finding := range findings {
		if finding.ID == findingID {
			return finding, true
		}
	}
	return model.Finding{}, false
}

func runContract(args []string) {
	findingID, root, asJSON, err := parseVerificationArgs("contract", args)
	if err != nil {
		die(err)
	}

	contract, err := application.Contract(context.Background(), root, findingID)
	if err != nil {
		die(err)
	}

	if asJSON {
		writeJSON(contract)
		return
	}
	fmt.Printf("CONTRACT %s %s %s baseline=%d\n",
		contract.TargetID,
		contract.TargetRuleID,
		contract.TargetPath,
		contract.TargetBaselineCount,
	)
}

func runVerify(args []string) {
	contractPath, root, asJSON, err := parseVerificationArgs("verify", args)
	if err != nil {
		die(err)
	}

	contract, err := readVerificationContract(contractPath)
	if err != nil {
		die(fmt.Errorf("read verification contract: %w", err))
	}
	result, err := application.Verify(context.Background(), root, contract)
	if err != nil {
		die(err)
	}

	if asJSON {
		writeJSON(result)
	} else {
		status := "FAIL"
		if result.Passed {
			status = "PASS"
		}
		fmt.Printf("VERIFY %s target_resolved=%t baseline=%d current=%d blocking=%d\n",
			status,
			result.TargetResolved,
			result.TargetBaselineCount,
			result.TargetCurrentCount,
			len(result.NewBlockingFindings),
		)
		for _, item := range result.NewBlockingFindings {
			fmt.Printf("BLOCKING %s %s %s baseline=%d current=%d\n",
				item.RuleID,
				item.Severity,
				item.Path,
				item.BaselineCount,
				item.CurrentCount,
			)
		}
	}
	if !result.Passed {
		os.Exit(1)
	}
}

func parseVerificationArgs(command string, args []string) (first, root string, asJSON bool, err error) {
	root = "."
	positionals := make([]string, 0, 2)
	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "-"):
			return "", "", false, fmt.Errorf("unknown option %s", arg)
		default:
			positionals = append(positionals, arg)
		}
	}
	if len(positionals) == 0 {
		return "", "", false, fmt.Errorf("%s requires a primary argument", command)
	}
	if len(positionals) > 2 {
		return "", "", false, fmt.Errorf("%s accepts a primary argument and optional repository path", command)
	}
	first = positionals[0]
	if len(positionals) == 2 {
		root = positionals[1]
	}
	return first, root, asJSON, nil
}

const maxVerificationContractBytes = 1 << 20

func readVerificationContract(path string) (verification.Contract, error) {
	file, err := os.Open(path)
	if err != nil {
		return verification.Contract{}, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxVerificationContractBytes+1))
	if err != nil {
		return verification.Contract{}, err
	}
	if len(data) > maxVerificationContractBytes {
		return verification.Contract{}, fmt.Errorf("verification contract exceeds %d bytes", maxVerificationContractBytes)
	}

	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var contract verification.Contract
	if err := decoder.Decode(&contract); err != nil {
		return verification.Contract{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return verification.Contract{}, errors.New("verification contract contains multiple JSON values")
		}
		return verification.Contract{}, err
	}
	return contract, nil
}

func runBenchmark(args []string) {
	asJSON := false
	manifest := ""
	analyzerName := "go"
	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--analyzer="):
			analyzerName = strings.TrimSpace(strings.TrimPrefix(arg, "--analyzer="))
		case strings.HasPrefix(arg, "-"):
			die(fmt.Errorf("unknown option %s", arg))
		case manifest == "":
			manifest = arg
		default:
			die(errors.New("benchmark accepts exactly one manifest path"))
		}
	}
	if manifest == "" {
		die(errors.New("benchmark manifest path is required"))
	}

	var analyzer detector.Analyzer
	switch analyzerName {
	case "go":
		analyzer = goanalysis.New()
	case "python":
		analyzer = pythonanalysis.New()
	case "javascript":
		analyzer = javascriptanalysis.New()
	case "rust":
		analyzer = rustanalysis.New()
	case "jvm":
		analyzer = jvmanalysis.New()
	case "cpp":
		analyzer = cppanalysis.New()
	default:
		die(fmt.Errorf("unknown benchmark analyzer %q", analyzerName))
	}

	report, err := benchmark.Evaluate(context.Background(), manifest, analyzer)
	if err != nil {
		die(fmt.Errorf("benchmark failed: %w", err))
	}
	if asJSON {
		writeJSON(report)
	} else {
		fmt.Printf("BENCHMARK %s cases=%d precision=%.4f recall=%.4f passed=%t\n",
			report.Analyzer, report.Cases, report.Metrics.Precision, report.Metrics.Recall, report.Passed)
		for rule, metrics := range report.Rules {
			fmt.Printf("%s tp=%d fp=%d fn=%d precision=%.4f recall=%.4f\n",
				rule, metrics.TruePositive, metrics.FalsePositive, metrics.FalseNegative, metrics.Precision, metrics.Recall)
		}
		for _, failure := range report.Failures {
			fmt.Printf("FAIL %s\n", failure)
		}
	}
	if !report.Passed {
		os.Exit(1)
	}
}

func parseArgs(args []string, defaultMaxFindings int) (string, bool, int, error) {
	root := "."
	asJSON := false
	maxFindings := defaultMaxFindings
	for _, arg := range args {
		switch {
		case arg == "--json":
			asJSON = true
		case strings.HasPrefix(arg, "--max-findings="):
			value, err := strconv.Atoi(strings.TrimPrefix(arg, "--max-findings="))
			if err != nil {
				return "", false, 0, fmt.Errorf("invalid --max-findings: %w", err)
			}
			maxFindings = value
		case strings.HasPrefix(arg, "-"):
			return "", false, 0, fmt.Errorf("unknown option %s", arg)
		default:
			root = arg
		}
	}
	return root, asJSON, maxFindings, nil
}

func writeJSON(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		die(fmt.Errorf("json encode failed: %w", err))
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func usage() {
	fmt.Println(`DoctorCode - deterministic code intelligence for small AI agents

Usage:
  doctorcode scan [path] [--json]
  doctorcode toolchains [path] [--json]
  doctorcode audit [path] [--json] [--max-findings=N]
  doctorcode next [path] [--json] [--max-bytes=N]
  doctorcode context <finding-id> [path] [--json] [--max-bytes=N]
  doctorcode contract <finding-id> [path] [--json]
  doctorcode verify <contract.json> [path] [--json]
  doctorcode benchmark <manifest.json> [--analyzer=go|python|javascript|rust|jvm|cpp] [--json]
  doctorcode version [--json]

M01 detector foundation:
  Go has intentionally narrow built-in rules for BLOAT, SECURITY, SIMPLIFY,
  LOGIC, and DEADCODE. Findings carry evidence and confidence boundaries.
  No M01 rule enables automatic deletion or automatic fixing.

M02+ precision benchmarks:
  Labeled corpora measure false positives and false negatives per analyzer.
  Benchmark thresholds are regression gates, not general precision claims.

M07 Python semantic adapter:
  Python uses the host Python 3 standard-library ast parser when available.
  Missing Python skips the optional analyzer during normal audit; the Python
  benchmark gate requires the interpreter explicitly.

Use "doctorcode next --json --max-bytes=4096" to give a small LLM one bounded
evidence packet instead of the whole repository. Use "doctorcode context <finding-id>"
to reproduce a bounded packet for one deterministic audit finding.

Before a repair, use "doctorcode contract <finding-id> --json" to freeze the
deterministic verification baseline. After the repair, use "doctorcode verify
<contract.json> --json". Verification re-audits the repository; it never runs
repository-provided test, shell, build, or verification commands.`)
}
