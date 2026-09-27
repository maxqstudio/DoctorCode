package pythonanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/maxqstudio/DoctorCode/internal/model"
)

type RelatedLocation struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Kind string `json:"kind"`
}

type rawRelatedResult struct {
	Locations []RelatedLocation `json:"locations"`
	Error     string            `json:"error,omitempty"`
}

func RelatedLocations(ctx context.Context, root string, finding model.Finding) ([]RelatedLocation, error) {
	if !strings.EqualFold(filepath.Ext(finding.Path), ".py") {
		return nil, nil
	}

	python, err := findPython(ctx)
	if err != nil {
		return nil, err
	}
	args := append([]string{}, python.prefix...)
	args = append(args,
		"-c", relatedScript,
		root,
		finding.Path,
		strconv.Itoa(finding.LineStart),
		strconv.Itoa(finding.LineEnd),
	)
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
		return nil, fmt.Errorf("python related-context analyzer failed: %s", message)
	}

	var raw rawRelatedResult
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode python related-context output: %w", err)
	}
	if raw.Error != "" {
		return nil, fmt.Errorf("python related-context analyzer failed: %s", raw.Error)
	}
	return raw.Locations, nil
}

const relatedScript = `
import ast
import json
import os
import sys

ROOT = os.path.realpath(os.path.abspath(sys.argv[1]))
SELECTED_REL = sys.argv[2].replace("\\", "/")
LINE_START = int(sys.argv[3])
LINE_END = int(sys.argv[4])
IGNORED = {".git", ".hg", ".svn", ".idea", ".vscode", "node_modules", "vendor", "dist", "build", "target", ".venv", "venv", "__pycache__", "testdata"}

def result(locations=None, error=""):
    json.dump({"locations": locations or [], "error": error}, sys.stdout, separators=(",", ":"))
    raise SystemExit(0)

def relpath(path):
    return os.path.relpath(path, ROOT).replace(os.sep, "/")

def contained(path):
    try:
        return os.path.normcase(os.path.commonpath([ROOT, os.path.realpath(path)])) == os.path.normcase(ROOT)
    except ValueError:
        return False

def is_test_path(rel):
    parts = rel.split("/")
    base = parts[-1].lower()
    return any(part.lower() in {"test", "tests"} for part in parts[:-1]) or base.startswith("test_") or base.endswith("_test.py")

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

selected_path = os.path.join(ROOT, *SELECTED_REL.split("/"))
if not contained(selected_path):
    result(error="selected Python path escapes repository root")

files = []
parse_errors = []
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
            parse_errors.append("%s:%d:%s" % (relpath(path), int(exc.lineno or 0), exc.msg))
            continue
        except (OSError, UnicodeError) as exc:
            parse_errors.append("%s:0:%s" % (relpath(path), type(exc).__name__))
            continue
        files.append((path, relpath(path), tree))

if parse_errors:
    result(error="python parse incomplete: " + sorted(parse_errors)[0])

by_rel = {rel: (path, tree) for path, rel, tree in files}
selected = by_rel.get(SELECTED_REL)
if selected is None:
    result(error="selected Python source is not part of the visible repository tree")
selected_tree = selected[1]

if LINE_START < 1:
    result()

if LINE_END < LINE_START:
    LINE_END = LINE_START

selected_fn = None
for node in selected_tree.body:
    if not isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
        continue
    end_lineno = int(getattr(node, "end_lineno", node.lineno) or node.lineno)
    if LINE_START >= node.lineno and LINE_END <= end_lineno:
        selected_fn = node
        break

if selected_fn is None:
    result()

TARGET_NAME = selected_fn.name
SELECTED_MODULES = set(module_names(SELECTED_REL))
primary_module_by_rel = {rel: primary_module_name(rel) for _, rel, _ in files}

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

SCOPE_TYPES = (ast.FunctionDef, ast.AsyncFunctionDef, ast.Lambda, ast.ClassDef, ast.ListComp, ast.SetComp, ast.DictComp, ast.GeneratorExp)

class ParentBuilder(ast.NodeVisitor):
    def __init__(self):
        self.parents = {}
    def generic_visit(self, node):
        for child in ast.iter_child_nodes(node):
            self.parents[child] = node
            self.visit(child)

class BindingCollector(ast.NodeVisitor):
    def __init__(self):
        self.bound = set()
        self.globals = set()
        self.nonlocals = set()

    def visit_Name(self, node):
        if isinstance(node.ctx, (ast.Store, ast.Del)):
            self.bound.add(node.id)

    def visit_Import(self, node):
        for alias in node.names:
            self.bound.add(alias.asname or alias.name.split(".")[0])

    def visit_ImportFrom(self, node):
        for alias in node.names:
            if alias.name != "*":
                self.bound.add(alias.asname or alias.name)

    def visit_Global(self, node):
        self.globals.update(node.names)

    def visit_Nonlocal(self, node):
        self.nonlocals.update(node.names)

    def visit_ExceptHandler(self, node):
        if isinstance(node.name, str):
            self.bound.add(node.name)
        for stmt in node.body:
            self.visit(stmt)

    def visit_FunctionDef(self, node):
        self.bound.add(node.name)

    def visit_AsyncFunctionDef(self, node):
        self.bound.add(node.name)

    def visit_ClassDef(self, node):
        self.bound.add(node.name)

    def visit_Lambda(self, node):
        return

    def visit_ListComp(self, node):
        return

    def visit_SetComp(self, node):
        return

    def visit_DictComp(self, node):
        return

    def visit_GeneratorExp(self, node):
        return

def target_names(target):
    out = set()
    if isinstance(target, ast.Name):
        out.add(target.id)
    elif isinstance(target, (ast.Tuple, ast.List)):
        for elt in target.elts:
            out.update(target_names(elt))
    return out

def scope_bound_names(scope):
    if isinstance(scope, (ast.ListComp, ast.SetComp, ast.DictComp, ast.GeneratorExp)):
        bound = set()
        for generator in scope.generators:
            bound.update(target_names(generator.target))
        return bound

    collector = BindingCollector()
    if isinstance(scope, (ast.FunctionDef, ast.AsyncFunctionDef, ast.Lambda)):
        args = scope.args
        for arg in list(args.posonlyargs) + list(args.args) + list(args.kwonlyargs):
            collector.bound.add(arg.arg)
        if args.vararg is not None:
            collector.bound.add(args.vararg.arg)
        if args.kwarg is not None:
            collector.bound.add(args.kwarg.arg)

    body = []
    if isinstance(scope, (ast.FunctionDef, ast.AsyncFunctionDef, ast.ClassDef)):
        body = scope.body
    elif isinstance(scope, ast.Lambda):
        body = [scope.body]

    for item in body:
        collector.visit(item)
    collector.bound.difference_update(collector.globals)
    collector.bound.update(collector.nonlocals)
    return collector.bound

def shadowed(node, name, parents, cache):
    current = parents.get(node)
    while current is not None:
        if isinstance(current, SCOPE_TYPES):
            key = id(current)
            if key not in cache:
                cache[key] = scope_bound_names(current)
            if name in cache[key]:
                return True
        current = parents.get(current)
    return False

locations = []
seen = set()

for path, rel, tree in files:
    direct_targets = set()
    imported_modules = {}

    if rel == SELECTED_REL:
        direct_targets.add(TARGET_NAME)

    for node in tree.body:
        if isinstance(node, ast.ImportFrom):
            target_module = resolve_from_module(rel, node)
            if target_module in SELECTED_MODULES:
                for alias in node.names:
                    if alias.name == TARGET_NAME:
                        direct_targets.add(alias.asname or alias.name)
        elif isinstance(node, ast.Import):
            for alias in node.names:
                if alias.asname:
                    imported_modules[alias.asname] = alias.name
                else:
                    root_name = alias.name.split(".")[0]
                    imported_modules[root_name] = root_name

    parents_builder = ParentBuilder()
    parents_builder.visit(tree)
    parents = parents_builder.parents
    cache = {}

    for node in ast.walk(tree):
        matched = False
        if isinstance(node, ast.Name) and isinstance(node.ctx, ast.Load) and node.id in direct_targets:
            if not shadowed(node, node.id, parents, cache):
                matched = True
        elif isinstance(node, ast.Attribute):
            parts = attribute_parts(node)
            if parts and len(parts) >= 2 and parts[-1] == TARGET_NAME:
                imported_root = imported_modules.get(parts[0])
                if imported_root is not None and not shadowed(node, parts[0], parents, cache):
                    module = ".".join([imported_root] + parts[1:-1])
                    if module in SELECTED_MODULES:
                        matched = True

        if not matched:
            continue

        line = int(getattr(node, "lineno", 0) or 0)
        if line < 1:
            continue
        kind = "test_reference" if is_test_path(rel) else "reference"
        key = (rel, line, kind)
        if key in seen:
            continue
        seen.add(key)
        locations.append({"path": rel, "line": line, "kind": kind})

locations.sort(key=lambda item: (item["path"], item["line"], item["kind"]))
result(locations=locations)
`
