package pythonanalysis

import (
	"bytes"
	"context"
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

func (a *Analyzer) Descriptor() detector.Descriptor {
	return detector.Descriptor{
		ID:         a.Name(),
		Language:   "Python",
		Extensions: []string{".py"},
		Parser: detector.ParserContract{
			Kind:       detector.ParserExternalAST,
			Provider:   "Python stdlib ast",
			FailClosed: true,
		},
		Availability: detector.AvailabilityContract{
			Mode:       detector.AvailabilityExternalTool,
			Dependency: "Python 3.8+",
		},
		EvidenceBoundary: "Python stdlib ast executed through an external Python 3.8+ interpreter; bounded module/import/reference reasoning is conservative and unresolved dynamic behavior remains unproven.",
		Rules: []detector.RuleMetadata{
			{ID: ruleBloat, Category: model.CategoryBloat, SafeAutofix: false},
			{ID: ruleDeadCode, Category: model.CategoryDeadCode, SafeAutofix: false},
			{ID: ruleLogic, Category: model.CategoryLogic, SafeAutofix: false},
			{ID: ruleSecurity, Category: model.CategorySecurity, SafeAutofix: false},
			{ID: ruleSimplify, Category: model.CategorySimplify, SafeAutofix: false},
		},
		Benchmark: detector.BenchmarkContract{SchemaVersion: 1, Languages: []string{"Python"}},
	}
}

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
		findings = append(findings, model.Finding{
			ID:           detector.FindingID(item.RuleID, item.Path, item.LineStart, item.Summary),
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

candidate_by_key = {}
candidates_by_name = {}
candidates_by_module_name = {}

def primary_module_name(rel):
    value = rel[:-3] if rel.endswith(".py") else rel
    if value == "__init__":
        return ""
    suffix = "/__init__"
    if value.endswith(suffix):
        value = value[:-len(suffix)]
    return value.replace("/", ".").strip(".")

def module_names(rel):
    primary = primary_module_name(rel)
    names = {primary}
    if primary.startswith("src.") and len(primary) > len("src."):
        names.add(primary[len("src."):])
    return tuple(sorted(names))

primary_module_by_rel = {rel: primary_module_name(rel) for _, rel, _, _, _ in files}
module_names_by_rel = {rel: module_names(rel) for _, rel, _, _, _ in files}
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
        key = (rel, name)
        candidate_by_key[key] = node
        candidates.append((rel, node, key))
        candidates_by_name.setdefault(name, []).append(key)
        for module in module_names_by_rel[rel]:
            candidates_by_module_name.setdefault((module, name), []).append(key)

usage_counts = {key: 0 for key in candidate_by_key}
live_keys = set()
forwarded_symbols = {}

def resolve_from_module(rel, node):
    if node.level == 0:
        return node.module or ""
    current = primary_module_by_rel.get(rel, "")
    package = current if rel.endswith("/__init__.py") or rel == "__init__.py" else current.rpartition(".")[0]
    parts = [part for part in package.split(".") if part]
    up = max(node.level - 1, 0)
    if up > len(parts):
        return node.module or ""
    base = parts[:len(parts) - up]
    if node.module:
        base.extend(node.module.split("."))
    return ".".join(base)

def unique_keys(keys):
    return list(dict.fromkeys(keys))

def symbol_keys(module, name):
    return unique_keys(
        candidates_by_module_name.get((module, name), [])
        + forwarded_symbols.get((module, name), [])
    )

# Propagate direct module-level re-exports to a small fixed point so
# "from pkg import _helper" can resolve through pkg/__init__.py to pkg.core.
for _ in range(len(files) + 1):
    changed = False
    for path, rel, source, tree, generated in files:
        current_modules = module_names_by_rel[rel]
        for node in tree.body:
            if not isinstance(node, ast.ImportFrom):
                continue
            target_module = resolve_from_module(rel, node)
            for alias in node.names:
                if alias.name == "*":
                    continue
                keys = symbol_keys(target_module, alias.name)
                if not keys:
                    continue
                local_name = alias.asname or alias.name
                for current_module in current_modules:
                    bucket = forwarded_symbols.setdefault((current_module, local_name), [])
                    before = len(bucket)
                    bucket[:] = unique_keys(bucket + keys)
                    if len(bucket) != before:
                        changed = True
    if not changed:
        break

def increment_usage(keys):
    for key in set(keys):
        if key in usage_counts:
            usage_counts[key] += 1

def mark_live(keys):
    for key in set(keys):
        if key in usage_counts:
            live_keys.add(key)

def local_candidate(rel, name):
    key = (rel, name)
    return [key] if key in usage_counts else []

def attribute_parts(node):
    parts = []
    current = node
    while isinstance(current, ast.Attribute):
        parts.append(current.attr)
        current = current.value
    if not isinstance(current, ast.Name):
        return None
    parts.append(current.id)
    parts.reverse()
    return parts

for path, rel, source, tree, generated in files:
    import_aliases = {}
    imported_modules = {}

    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom):
            target_module = resolve_from_module(rel, node)
            for alias in node.names:
                if alias.name == "*":
                    continue
                local_name = alias.asname or alias.name
                keys = symbol_keys(target_module, alias.name)
                if keys:
                    import_aliases.setdefault(local_name, []).extend(keys)
                    mark_live(keys)
        elif isinstance(node, ast.Import):
            for alias in node.names:
                if alias.asname:
                    imported_modules[alias.asname] = alias.name
                else:
                    root_name = alias.name.split(".")[0]
                    imported_modules[root_name] = root_name

    for node in ast.walk(tree):
        if isinstance(node, ast.Name) and isinstance(node.ctx, ast.Load):
            increment_usage(local_candidate(rel, node.id))
            increment_usage(import_aliases.get(node.id, []))
        elif isinstance(node, ast.Attribute):
            resolved = False
            parts = attribute_parts(node)
            if parts and len(parts) >= 2:
                imported_root = imported_modules.get(parts[0])
                if imported_root is not None:
                    module = ".".join([imported_root] + parts[1:-1])
                    keys = symbol_keys(module, parts[-1])
                    if keys:
                        increment_usage(keys)
                        resolved = True
            if not resolved:
                keys = candidates_by_name.get(node.attr, [])
                if len(keys) == 1:
                    increment_usage(keys)
        elif isinstance(node, ast.Call):
            if (
                isinstance(node.func, ast.Name)
                and node.func.id == "getattr"
                and len(node.args) >= 2
                and isinstance(node.args[1], ast.Constant)
                and isinstance(node.args[1].value, str)
            ):
                name = node.args[1].value
                keys = local_candidate(rel, name)
                fallback = candidates_by_name.get(name, [])
                increment_usage(keys if keys else (fallback if len(fallback) == 1 else []))
        elif isinstance(node, ast.Subscript):
            if (
                isinstance(node.value, ast.Call)
                and isinstance(node.value.func, ast.Name)
                and node.value.func.id == "globals"
                and not node.value.args
                and not node.value.keywords
                and isinstance(node.slice, ast.Constant)
                and isinstance(node.slice.value, str)
            ):
                increment_usage(local_candidate(rel, node.slice.value))

    exported = set()
    for node in tree.body:
        value = None
        if isinstance(node, ast.Assign) and any(isinstance(target, ast.Name) and target.id == "__all__" for target in node.targets):
            value = node.value
        elif isinstance(node, ast.AnnAssign) and isinstance(node.target, ast.Name) and node.target.id == "__all__":
            value = node.value
        if value is not None:
            for child in ast.walk(value):
                if isinstance(child, ast.Constant) and isinstance(child.value, str):
                    exported.add(child.value)
    for name in exported:
        keys = local_candidate(rel, name)
        if not keys:
            for module in module_names_by_rel[rel]:
                keys.extend(symbol_keys(module, name))
        mark_live(keys)

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

for rel, node, key in candidates:
    count = usage_counts.get(key, 0)
    if count == 0 and key not in live_keys:
        emit(
            "PY-DEADCODE-PRIVATE-ZERO-REF", "DEADCODE", "MEDIUM", "HIGH",
            rel, node,
            "private module-level function %s has no conservative references in the visible Python tree" % node.name,
            [
                "0 conservative usage references and no recognized import/export liveness evidence",
                "decorated, dunder, generated, and test-defined functions are excluded from candidates",
                "recognized source-layout aliases, re-exports, imports, module attributes, getattr string names, globals string subscripts, and __all__ contribute conservative evidence",
                "HIGH is below PROVEN_UNUSED because reflection, plugins, external callers, and unresolved dynamic behavior may exist",
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

for rel, node, key in candidates:
    if usage_counts.get(key, 0) == 1 and forwarded_call(node):
        emit(
            "PY-BLOAT-PASSTHROUGH-WRAPPER", "BLOAT", "LOW", "SUSPICIOUS",
            rel, node,
            "private one-call pass-through wrapper %s has one conservative usage reference" % node.name,
            [
                "body is a single return of another call",
                "plain positional parameters are forwarded unchanged and in order",
                "wrapper has exactly one candidate/module-aware usage reference",
                "import/export liveness evidence is tracked separately and does not inflate the one-use wrapper count",
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
    if not isinstance(right, ast.Constant):
        return None
    if not (right.value is None or right.value is True or right.value is False):
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
    markers = ("example", "dummy", "placeholder", "changeme", "change-me", "notasecret", "not-a-secret", "not_secret", "not-secret", "your_", "your-")
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
