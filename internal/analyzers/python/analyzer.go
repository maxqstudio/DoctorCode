package pythonanalysis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/detector"
	"github.com/maxqstudio/DoctorCode/internal/model"
)

const (
	ruleDeadCode = "PY-DEADCODE-PRIVATE-ZERO-REF"
	ruleLogic    = "PY-LOGIC-DUPLICATE-IDENTITY-CONDITION"
	ruleSimplify = "PY-SIMPLIFY-BOOL-RETURN"
	ruleSecurity = "PY-SEC-HARDCODED-CREDENTIAL"
	ruleBloat    = "PY-BLOAT-PASSTHROUGH-WRAPPER"
)

type Analyzer struct{}

func New() *Analyzer { return &Analyzer{} }

func (a *Analyzer) Name() string { return "python/stdlib-ast-v1" }

type rawFinding struct {
	RuleID       string           `json:"rule_id"`
	Category     model.Category   `json:"category"`
	Severity     model.Severity   `json:"severity"`
	Confidence   model.Confidence `json:"confidence"`
	Path         string           `json:"path"`
	LineStart    int              `json:"line_start"`
	LineEnd      int              `json:"line_end"`
	Summary      string           `json:"summary"`
	Evidence     []string         `json:"evidence"`
	Verification []string         `json:"verification"`
}

type rawParseError struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type rawResult struct {
	Findings    []rawFinding    `json:"findings"`
	ParseErrors []rawParseError `json:"parse_errors"`
}

type pythonCommand struct {
	path   string
	prefix []string
}

func (a *Analyzer) Analyze(ctx context.Context, root string) ([]model.Finding, error) {
	hasFiles, err := hasPythonFiles(ctx, root)
	if err != nil {
		return nil, err
	}
	if !hasFiles {
		return nil, fmt.Errorf("%w: no Python source files", detector.ErrUnavailable)
	}

	python, err := findPython(ctx)
	if err != nil {
		return nil, err
	}

	args := append([]string{}, python.prefix...)
	args = append(args, "-c", analyzerScript, root)
	cmd := exec.CommandContext(ctx, python.path, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("python ast analyzer failed: %s", message)
	}

	var raw rawResult
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode python analyzer output: %w", err)
	}
	if len(raw.ParseErrors) > 0 {
		first := raw.ParseErrors[0]
		return nil, fmt.Errorf("python parse incomplete: %s:%d: %s", first.Path, first.Line, first.Message)
	}

	findings := make([]model.Finding, 0, len(raw.Findings))
	for _, item := range raw.Findings {
		key := fmt.Sprintf("%s|%s|%d|%s", item.RuleID, item.Path, item.LineStart, item.Summary)
		digest := sha256.Sum256([]byte(key))
		findings = append(findings, model.Finding{
			ID:           fmt.Sprintf("%s-%x", item.RuleID, digest[:5]),
			RuleID:       item.RuleID,
			Category:     item.Category,
			Severity:     item.Severity,
			Confidence:   item.Confidence,
			Path:         item.Path,
			LineStart:    item.LineStart,
			LineEnd:      item.LineEnd,
			Summary:      item.Summary,
			Evidence:     item.Evidence,
			Verification: item.Verification,
			SafeAutofix:  false,
		})
	}
	return findings, nil
}

func hasPythonFiles(ctx context.Context, root string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && ignoredDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), ".py") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found, err
}

func ignoredDirectory(name string) bool {
	switch name {
	case ".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", ".venv", "venv", "__pycache__", "testdata":
		return true
	default:
		return false
	}
}

func findPython(ctx context.Context) (pythonCommand, error) {
	candidates := []struct {
		name   string
		prefix []string
	}{
		{name: "python3"},
		{name: "python"},
		{name: "py", prefix: []string{"-3"}},
	}
	for _, candidate := range candidates {
		path, err := exec.LookPath(candidate.name)
		if err != nil {
			continue
		}
		args := append([]string{}, candidate.prefix...)
		args = append(args, "-c", "import sys; raise SystemExit(0 if sys.version_info >= (3, 8) else 1)")
		cmd := exec.CommandContext(ctx, path, args...)
		if err := cmd.Run(); err == nil {
			return pythonCommand{path: path, prefix: candidate.prefix}, nil
		}
	}
	return pythonCommand{}, fmt.Errorf("%w: Python 3.8+ interpreter not found", detector.ErrUnavailable)
}

const analyzerScript = `
import ast
import hashlib
import json
import os
import sys

ROOT = os.path.abspath(sys.argv[1])
IGNORED = {".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", ".venv", "venv", "__pycache__", "testdata"}
findings = []
parse_errors = []
files = []

def relpath(path):
    return os.path.relpath(path, ROOT).replace(os.sep, "/")

def is_test_path(rel):
    parts = rel.split("/")
    base = parts[-1].lower()
    return any(p.lower() in {"test", "tests"} for p in parts[:-1]) or base.startswith("test_") or base.endswith("_test.py")

def generated_source(text):
    head = "\n".join(text.splitlines()[:8]).lower()
    return "generated" in head and ("do not edit" in head or "do not modify" in head)

for dirpath, dirnames, filenames in os.walk(ROOT, followlinks=False):
    dirnames[:] = [
        name for name in dirnames
        if name not in IGNORED and not os.path.islink(os.path.join(dirpath, name))
    ]
    for filename in filenames:
        if not filename.lower().endswith(".py"):
            continue
        path = os.path.join(dirpath, filename)
        if os.path.islink(path):
            continue
        try:
            with open(path, "r", encoding="utf-8-sig") as handle:
                source = handle.read()
            tree = ast.parse(source, filename=path)
        except SyntaxError as exc:
            parse_errors.append({
                "path": relpath(path),
                "line": int(exc.lineno or 0),
                "message": str(exc.msg),
            })
            continue
        except (OSError, UnicodeError) as exc:
            parse_errors.append({
                "path": relpath(path),
                "line": 0,
                "message": type(exc).__name__,
            })
            continue
        files.append((path, relpath(path), source, tree, generated_source(source)))

candidate_names = set()
candidates = []

for path, rel, source, tree, generated in files:
    if is_test_path(rel):
        continue
    for node in tree.body:
        if not isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
            continue
        name = node.name
        if not name.startswith("_") or name.startswith("__"):
            continue
        if node.decorator_list:
            continue
        if generated:
            continue
        candidate_names.add(name)
        candidates.append((rel, node))

reference_counts = {name: 0 for name in candidate_names}
for path, rel, source, tree, generated in files:
    for node in ast.walk(tree):
        if isinstance(node, ast.Name) and isinstance(node.ctx, ast.Load) and node.id in reference_counts:
            reference_counts[node.id] += 1
        elif isinstance(node, ast.Attribute) and node.attr in reference_counts:
            reference_counts[node.attr] += 1

def emit(rule_id, category, severity, confidence, rel, node, summary, evidence):
    findings.append({
        "rule_id": rule_id,
        "category": category,
        "severity": severity,
        "confidence": confidence,
        "path": rel,
        "line_start": int(getattr(node, "lineno", 0) or 0),
        "line_end": int(getattr(node, "end_lineno", getattr(node, "lineno", 0)) or 0),
        "summary": summary,
        "evidence": evidence,
        "verification": ["python -m compileall .", "project tests"],
    })

for rel, node in candidates:
    count = reference_counts.get(node.name, 0)
    if count == 0:
        emit(
            "PY-DEADCODE-PRIVATE-ZERO-REF", "DEADCODE", "MEDIUM", "HIGH",
            rel, node,
            "private module-level function %s has no lexical references in the visible Python tree" % node.name,
            [
                "0 Name-load or Attribute references found across parsed Python files",
                "decorated, dunder, generated, and test-defined functions are excluded from candidates",
                "HIGH is below PROVEN_UNUSED because imports, reflection, plugins, and external callers may exist",
            ],
        )

def simple_parameters(node):
    args = node.args
    if args.vararg is not None or args.kwarg is not None or args.kwonlyargs or args.defaults or args.kw_defaults:
        return None
    params = list(args.posonlyargs) + list(args.args)
    return [arg.arg for arg in params]

def forwarded_call(node):
    if len(node.body) != 1 or not isinstance(node.body[0], ast.Return):
        return False
    call = node.body[0].value
    if not isinstance(call, ast.Call) or call.keywords:
        return False
    params = simple_parameters(node)
    if params is None or len(params) != len(call.args):
        return False
    for expected, actual in zip(params, call.args):
        if not isinstance(actual, ast.Name) or actual.id != expected:
            return False
    return True

for rel, node in candidates:
    if reference_counts.get(node.name, 0) == 1 and forwarded_call(node):
        emit(
            "PY-BLOAT-PASSTHROUGH-WRAPPER", "BLOAT", "LOW", "SUSPICIOUS",
            rel, node,
            "private one-call pass-through wrapper %s has one lexical reference" % node.name,
            [
                "body is a single return of another call",
                "plain positional parameters are forwarded unchanged and in order",
                "wrapper has exactly one conservative lexical reference",
                "SUSPICIOUS only: naming, policy, instrumentation, or API intent may justify the wrapper",
            ],
        )

def identity_key(expr):
    if not isinstance(expr, ast.Compare) or len(expr.ops) != 1 or len(expr.comparators) != 1:
        return None
    if not isinstance(expr.left, ast.Name):
        return None
    op = expr.ops[0]
    if not isinstance(op, (ast.Is, ast.IsNot)):
        return None
    right = expr.comparators[0]
    if not isinstance(right, ast.Constant) or right.value not in (None, True, False):
        return None
    return (expr.left.id, type(op).__name__, repr(right.value))

for path, rel, source, tree, generated in files:
    if is_test_path(rel):
        continue
    for node in ast.walk(tree):
        if not isinstance(node, ast.If):
            continue
        seen = {}
        current = node
        while isinstance(current, ast.If):
            key = identity_key(current.test)
            if key is not None:
                if key in seen:
                    emit(
                        "PY-LOGIC-DUPLICATE-IDENTITY-CONDITION", "LOGIC", "MEDIUM", "HIGH",
                        rel, current.test,
                        "duplicate side-effect-free identity condition appears later in the same if/elif chain",
                        [
                            "same name/is-identity condition already appeared at line %d" % seen[key],
                            "rule is intentionally limited to Name is/is-not None/True/False",
                        ],
                    )
                    break
                seen[key] = current.test.lineno
            if len(current.orelse) == 1 and isinstance(current.orelse[0], ast.If):
                current = current.orelse[0]
            else:
                break

def bool_literal_return(block):
    if len(block) != 1 or not isinstance(block[0], ast.Return):
        return None
    value = block[0].value
    if isinstance(value, ast.Constant) and isinstance(value.value, bool):
        return value.value
    return None

for path, rel, source, tree, generated in files:
    if is_test_path(rel):
        continue
    for node in ast.walk(tree):
        if not isinstance(node, ast.If) or len(node.orelse) != 1 or isinstance(node.orelse[0], ast.If):
            continue
        left = bool_literal_return(node.body)
        right = bool_literal_return(node.orelse)
        if left is None or right is None or left == right:
            continue
        replacement = "return bool(condition)" if left else "return not bool(condition)"
        emit(
            "PY-SIMPLIFY-BOOL-RETURN", "SIMPLIFY", "LOW", "PROVEN",
            rel, node,
            "boolean if/else returns opposite literals and can be represented directly",
            [
                "both branches contain exactly one boolean return",
                "branches return opposite boolean literals",
                "behavior-preserving shape: %s" % replacement,
            ],
        )

def credential_like(name):
    normalized = "".join(ch for ch in name.lower() if ch.isalnum())
    suffixes = ("password", "passwd", "secret", "apikey", "token", "privatekey", "accesskey")
    return any(normalized == suffix or normalized.endswith(suffix) for suffix in suffixes)

def placeholder(value):
    lower = value.strip().lower()
    if lower.startswith("$" + "{") or lower.startswith("$("):
        return True
    markers = ("example", "dummy", "placeholder", "changeme", "change-me", "notasecret", "not-a-secret", "your_", "your-")
    return any(marker in lower for marker in markers)

def assigned_names(target):
    if isinstance(target, ast.Name):
        return [target.id]
    if isinstance(target, (ast.Tuple, ast.List)):
        out = []
        for elt in target.elts:
            out.extend(assigned_names(elt))
        return out
    return []

for path, rel, source, tree, generated in files:
    if is_test_path(rel):
        continue
    for node in ast.walk(tree):
        pairs = []
        if isinstance(node, ast.Assign):
            for target in node.targets:
                for name in assigned_names(target):
                    pairs.append((name, node.value))
        elif isinstance(node, ast.AnnAssign):
            for name in assigned_names(node.target):
                pairs.append((name, node.value))
        for name, value_node in pairs:
            if not credential_like(name):
                continue
            if not isinstance(value_node, ast.Constant) or not isinstance(value_node.value, str):
                continue
            value = value_node.value
            if len(value) < 8 or placeholder(value):
                continue
            digest = hashlib.sha256(value.encode("utf-8")).hexdigest()[:8]
            emit(
                "PY-SEC-HARDCODED-CREDENTIAL", "SECURITY", "HIGH", "SUSPICIOUS",
                rel, value_node,
                "credential-like identifier %s is assigned a hardcoded string literal" % name,
                [
                    "literal value redacted; byte_length=%d; sha256_prefix=%s" % (len(value.encode("utf-8")), digest),
                    "SUSPICIOUS only: the literal may be a non-secret fixture not recognized by the conservative filter",
                ],
            )

findings.sort(key=lambda item: (item["rule_id"], item["path"], item["line_start"], item["summary"]))
parse_errors.sort(key=lambda item: (item["path"], item["line"], item["message"]))
json.dump({"findings": findings, "parse_errors": parse_errors}, sys.stdout, separators=(",", ":"))
`
