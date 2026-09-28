#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import tarfile
import zipfile

EXPECTED_TARGETS = {
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
    ("windows", "amd64"),
    ("windows", "arm64"),
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("directory")
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--date", required=True)
    return parser.parse_args()


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def archive_names(path: Path, archive_format: str) -> list[str]:
    if archive_format == "zip":
        with zipfile.ZipFile(path) as handle:
            return handle.namelist()
    if archive_format == "tar.gz":
        with tarfile.open(path, "r:gz") as handle:
            return handle.getnames()
    raise RuntimeError(f"unsupported archive format: {archive_format}")


def validate_archive(path: Path, item: dict, version: str) -> None:
    names = archive_names(path, item["format"])
    for name in names:
        pure = PurePosixPath(name)
        if pure.is_absolute() or ".." in pure.parts:
            raise RuntimeError(f"unsafe archive path in {path.name}: {name}")

    root = f"doctorcode_{version}_{item['os']}_{item['arch']}"
    exe = ".exe" if item["os"] == "windows" else ""
    required = {
        f"{root}/doctorcode{exe}",
        f"{root}/doctorcode-mcp{exe}",
        f"{root}/README.md",
        f"{root}/LICENSE",
        f"{root}/skills/doctorcode/SKILL.md",
    }
    files = {name.rstrip("/") for name in names if not name.endswith("/")}
    if files != required:
        raise RuntimeError(f"archive content mismatch for {path.name}: got={sorted(files)} want={sorted(required)}")


def main() -> int:
    args = parse_args()
    directory = Path(args.directory)
    manifest_path = directory / "release-manifest.json"
    checksums_path = directory / "checksums.txt"
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))

    if manifest.get("schema_version") != 1 or manifest.get("product") != "DoctorCode":
        raise SystemExit("invalid release manifest identity")
    expected_metadata = (args.version, args.commit, args.date)
    actual_metadata = (manifest.get("version"), manifest.get("commit"), manifest.get("build_date"))
    if actual_metadata != expected_metadata:
        raise SystemExit(f"release metadata mismatch: got={actual_metadata} want={expected_metadata}")

    seen_targets = set()
    expected_checksums = {}
    artifacts = manifest.get("artifacts")
    if not isinstance(artifacts, list) or len(artifacts) != len(EXPECTED_TARGETS):
        raise SystemExit("release manifest must contain exactly six artifacts")

    for item in artifacts:
        target = (item.get("os"), item.get("arch"))
        if target in seen_targets:
            raise SystemExit(f"duplicate release target: {target}")
        seen_targets.add(target)
        path = directory / item["filename"]
        if not path.is_file():
            raise SystemExit(f"missing release artifact: {path.name}")
        if path.stat().st_size != item.get("size"):
            raise SystemExit(f"release size mismatch: {path.name}")
        actual_hash = sha256(path)
        if actual_hash != item.get("sha256"):
            raise SystemExit(f"release sha256 mismatch: {path.name}")
        validate_archive(path, item, args.version)
        expected_checksums[path.name] = actual_hash

    if seen_targets != EXPECTED_TARGETS:
        raise SystemExit(f"release targets mismatch: got={sorted(seen_targets)}")
    expected_checksums[manifest_path.name] = sha256(manifest_path)

    actual_checksums = {}
    for raw in checksums_path.read_text(encoding="utf-8").splitlines():
        if not raw.strip():
            continue
        parts = raw.split(None, 1)
        if len(parts) != 2:
            raise SystemExit(f"invalid checksum line: {raw}")
        digest, name = parts
        name = name.strip()
        if name in actual_checksums:
            raise SystemExit(f"duplicate checksum entry: {name}")
        actual_checksums[name] = digest
    if actual_checksums != expected_checksums:
        raise SystemExit(f"checksums mismatch: got={actual_checksums} want={expected_checksums}")

    print("M17_RELEASE_VERIFY=PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
