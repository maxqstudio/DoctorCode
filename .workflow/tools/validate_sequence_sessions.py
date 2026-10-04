#!/usr/bin/env python3
"""Validate all sequence session contracts required by the project profile."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

from project_profile import parse_profile, sequence_settings


def run_validator(root: Path, validator: Path, rel: str, child_env: dict[str, str]) -> tuple[int, str]:
    command = [
        sys.executable,
        str(validator),
        "--root",
        str(root),
        "--session",
        rel,
    ]
    if validator.name == "validate_sequence_contract.py":
        command.append("--no-write-report")
    proc = subprocess.run(
        command,
        cwd=root,
        env=child_env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        check=False,
    )
    return proc.returncode, proc.stdout


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=".")
    ap.add_argument("--sessions-dir", default="docs/sequence/sessions")
    args = ap.parse_args()

    start = Path(args.root).resolve()
    try:
        root = Path(subprocess.check_output(
            ["git", "-C", str(start), "rev-parse", "--show-toplevel"],
            text=True,
        ).strip())
    except Exception as exc:
        print(json.dumps({"result": "FAIL", "error": f"GIT_ERROR:{exc}"}, indent=2))
        return 1

    try:
        policy = sequence_settings(parse_profile(root / "PROJECT_PROFILE.yaml"))
    except Exception as exc:
        print(json.dumps({"result": "FAIL", "error": f"PROFILE_ERROR:{exc}"}, indent=2))
        return 1

    sessions_root = root / args.sessions_dir
    sessions = sorted(sessions_root.rglob("*.json")) if sessions_root.is_dir() else []

    if policy.get("required", False) and not sessions:
        print(json.dumps({
            "result": "FAIL",
            "sequence_required": True,
            "sessions": 0,
            "failures": ["SEQUENCE_SESSION_CONTRACT_MISSING"],
        }, indent=2))
        return 1

    tool_dir = Path(__file__).resolve().parent
    contract_validator = tool_dir / "validate_sequence_contract.py"
    if not contract_validator.is_file():
        contract_validator = root / "scripts" / "validate_sequence_contract.py"
    if not contract_validator.is_file():
        print(json.dumps({
            "result": "FAIL",
            "failures": ["SEQUENCE_VALIDATOR_MISSING"],
        }, indent=2))
        return 1

    human_validator = tool_dir / "validate_sequence_human_view.py"
    if not human_validator.is_file():
        human_validator = root / "scripts" / "validate_sequence_human_view.py"

    results = []
    failed = 0
    human_views = 0
    failed_human_views = 0
    child_env = os.environ.copy()
    child_env["PYTHONDONTWRITEBYTECODE"] = "1"

    for session in sessions:
        rel = session.relative_to(root).as_posix()
        contract_rc, contract_output = run_validator(
            root, contract_validator, rel, child_env
        )

        try:
            session_data = json.loads(session.read_text(encoding="utf-8"))
        except Exception:
            session_data = {}
        human_declared = isinstance(session_data.get("human_view"), dict) and bool(session_data.get("human_view"))
        human_rc = 0
        human_output = "NOT_APPLICABLE:HUMAN_VIEW_NOT_DECLARED\n"

        if human_declared:
            human_views += 1
            if not human_validator.is_file():
                human_rc = 1
                human_output = "HUMAN_SEQUENCE_VALIDATOR_MISSING\n"
            else:
                human_rc, human_output = run_validator(
                    root, human_validator, rel, child_env
                )
            if human_rc != 0:
                failed_human_views += 1

        session_failed = contract_rc != 0 or human_rc != 0
        if session_failed:
            failed += 1

        # Keep legacy returncode/output as machine-contract results so existing
        # consumers do not silently change semantics.
        results.append({
            "session": rel,
            "returncode": contract_rc,
            "output": contract_output,
            "human_view_declared": human_declared,
            "human_view_returncode": human_rc,
            "human_view_output": human_output,
        })

    report = {
        "sequence_required": policy.get("required", False),
        "runtime_trace_required": policy.get("runtime_trace_required", False),
        "sessions": len(sessions),
        "failed_sessions": failed,
        "human_views": human_views,
        "failed_human_views": failed_human_views,
        "results": results,
        "result": "FAIL" if failed else "PASS",
    }
    print(json.dumps(report, indent=2))
    return 1 if failed else 0


if __name__ == "__main__":
    raise SystemExit(main())
