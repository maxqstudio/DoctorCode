#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path
import platform
import subprocess


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", required=True)
    parser.add_argument("--bin-dir", required=True)
    return parser.parse_args()


def target() -> tuple[str, str]:
    system = platform.system().lower()
    if system == "darwin":
        goos = "darwin"
    elif system == "windows":
        goos = "windows"
    elif system == "linux":
        goos = "linux"
    else:
        raise SystemExit(f"unsupported smoke operating system: {system}")

    machine = platform.machine().lower()
    if machine in {"x86_64", "amd64"}:
        arch = "amd64"
    elif machine in {"arm64", "aarch64"}:
        arch = "arm64"
    else:
        raise SystemExit(f"unsupported smoke architecture: {machine}")
    return goos, arch


def version_json(path: Path, args: list[str]) -> dict:
    proc = subprocess.run([str(path), *args], text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False)
    if proc.returncode != 0:
        raise SystemExit(f"{path.name} version command failed: {proc.stderr}")
    try:
        return json.loads(proc.stdout)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"{path.name} returned invalid version JSON: {proc.stdout}") from exc


def main() -> int:
    args = parse_args()
    manifest = json.loads(Path(args.manifest).read_text(encoding="utf-8"))
    goos, arch = target()
    if not any((item["os"], item["arch"]) == (goos, arch) for item in manifest["artifacts"]):
        raise SystemExit(f"manifest does not contain native target {(goos, arch)}")

    bin_dir = Path(args.bin_dir)
    exe = ".exe" if goos == "windows" else ""
    expected = {
        "version": manifest["version"],
        "commit": manifest["commit"],
        "build_date": manifest["build_date"],
    }
    cli = version_json(bin_dir / f"doctorcode{exe}", ["version", "--json"])
    mcp = version_json(bin_dir / f"doctorcode-mcp{exe}", ["--version", "--json"])
    if cli != expected:
        raise SystemExit(f"doctorcode metadata mismatch: got={cli} want={expected}")
    if mcp != expected:
        raise SystemExit(f"doctorcode-mcp metadata mismatch: got={mcp} want={expected}")
    print(f"M17_INSTALL_SMOKE=PASS target={goos}/{arch}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
