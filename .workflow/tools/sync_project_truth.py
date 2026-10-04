#!/usr/bin/env python3
"""Synchronize Project Truth Compiler outputs and record PROJECT_DOCS_SYNC.

This script remains fail-closed and keeps the accepted public CLI contract.
SW2-01 executes sibling compiler/validator functions in-process and reuses one
active ProjectSnapshot so repeated passes do not rescan or reread source files.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import generate_project_docs
import validate_project_docs
from project_profile import (
    PROFILE_FILE,
    documentation_settings,
    parse_profile,
)
from project_snapshot import (
    ProjectSnapshot,
    active_project_snapshot,
    active_snapshot_for,
)
from script_runner import invoke_main


def run_main(main_func, argv: list[str], program: str) -> tuple[int, str]:
    return invoke_main(main_func, argv, program=program)


def sync_once(root: Path, *, no_record: bool) -> int:
    profile_path = root / PROFILE_FILE
    if not profile_path.is_file():
        print("FAIL MISSING_PROJECT_PROFILE")
        return 1

    try:
        documentation = documentation_settings(parse_profile(profile_path))
    except Exception as exc:
        print("FAIL PROJECT_PROFILE_INVALID:" + str(exc))
        return 1

    code, output = run_main(
        generate_project_docs.main,
        ["--root", str(root)],
        "generate_project_docs.py",
    )
    print(output, end="")
    if code != 0:
        print("PROJECT_DOCS_SYNC=FAIL")
        return code

    code, output = run_main(
        validate_project_docs.main,
        ["--root", str(root)],
        "validate_project_docs.py",
    )
    print(output, end="")
    if code != 0:
        print("PROJECT_DOCS_SYNC=FAIL")
        return code

    if not documentation.get("generated", False):
        print("PROJECT_DOCS_SYNC=NOT_APPLICABLE")
        return 0

    if no_record:
        print("PROJECT_DOCS_SYNC=PASS")
        return 0

    spec_root = Path(str(documentation.get("spec_root", ".workflow")))
    if not spec_root.is_absolute():
        spec_root = root / spec_root
    acceptance_path = spec_root / "acceptance.json"
    if not acceptance_path.is_file():
        print("FAIL ACCEPTANCE_SPEC_MISSING:" + str(acceptance_path))
        return 1

    try:
        acceptance = json.loads(acceptance_path.read_text(encoding="utf-8"))
        gates = acceptance.setdefault("truth_gates", {})
        gates["ROADMAP_SYNC"] = "PASS"
        gates["DOC_LAYOUT"] = "PASS"
        gates["PROJECT_DOCS_NORMALIZED"] = "PASS"
        gates["DOC_READABILITY"] = "PASS"
        gates["PROJECT_DOCS_SYNC"] = "PASS"
        acceptance_path.write_bytes(
            (json.dumps(acceptance, indent=2, sort_keys=True) + "\n").encode("utf-8")
        )
    except Exception as exc:
        print("FAIL ACCEPTANCE_SPEC_UPDATE_ERROR:" + str(exc))
        return 1

    code, output = run_main(
        generate_project_docs.main,
        ["--root", str(root)],
        "generate_project_docs.py",
    )
    print(output, end="")
    if code != 0:
        print("PROJECT_DOCS_SYNC=FAIL")
        return code

    code, output = run_main(
        validate_project_docs.main,
        ["--root", str(root)],
        "validate_project_docs.py",
    )
    print(output, end="")
    if code != 0:
        print("PROJECT_DOCS_SYNC=FAIL")
        return code

    print("PROJECT_DOCS_SYNC=PASS")
    try:
        gate_ref = acceptance_path.relative_to(root).as_posix()
    except ValueError:
        gate_ref = str(acceptance_path)
    print(
        "RECORDED_GATE="
        + gate_ref
        + "::truth_gates.{ROADMAP_SYNC,DOC_LAYOUT,PROJECT_DOCS_NORMALIZED,DOC_READABILITY,PROJECT_DOCS_SYNC}"
    )
    return 0


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=".")
    ap.add_argument("--no-record", action="store_true")
    args = ap.parse_args()

    root = Path(args.root).resolve()
    if active_snapshot_for(root) is not None:
        return sync_once(root, no_record=args.no_record)

    snapshot = ProjectSnapshot.capture(root)
    with active_project_snapshot(snapshot):
        return sync_once(root, no_record=args.no_record)


if __name__ == "__main__":
    raise SystemExit(main())
