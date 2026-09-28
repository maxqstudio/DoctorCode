#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import stat
import subprocess
import tarfile
import tempfile
import zipfile

TARGETS = (
    ("linux", "amd64"),
    ("linux", "arm64"),
    ("darwin", "amd64"),
    ("darwin", "arm64"),
    ("windows", "amd64"),
    ("windows", "arm64"),
)
VERSION_RE = re.compile(r"^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$")
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")
BUILDINFO_PATH = "github.com/maxqstudio/DoctorCode/internal/buildinfo"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", required=True)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--date", required=True)
    parser.add_argument("--output", required=True)
    return parser.parse_args()


def validate_metadata(version: str, commit: str, build_date: str) -> tuple[int, tuple[int, int, int, int, int, int]]:
    if not VERSION_RE.fullmatch(version):
        raise SystemExit(f"invalid release version: {version}")
    if not COMMIT_RE.fullmatch(commit):
        raise SystemExit(f"invalid release commit: {commit}")
    try:
        parsed = dt.datetime.fromisoformat(build_date.replace("Z", "+00:00"))
    except ValueError as exc:
        raise SystemExit(f"invalid release date: {build_date}") from exc
    if parsed.tzinfo is None:
        raise SystemExit("release date must include a timezone")
    parsed = parsed.astimezone(dt.timezone.utc)
    epoch = int(parsed.timestamp())
    zip_year = max(1980, parsed.year)
    zip_time = (zip_year, parsed.month, parsed.day, parsed.hour, parsed.minute, parsed.second)
    return epoch, zip_time


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def build_binary(repo: Path, cwd: Path, package: str, output: Path, goos: str, goarch: str, ldflags: str) -> None:
    env = os.environ.copy()
    env.update({"GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"})
    subprocess.run(
        [
            "go",
            "build",
            "-trimpath",
            "-buildvcs=false",
            "-ldflags",
            ldflags,
            "-o",
            str(output),
            package,
        ],
        cwd=cwd,
        env=env,
        check=True,
    )


def populate_package(repo: Path, package_dir: Path, version: str, commit: str, build_date: str, goos: str, goarch: str) -> None:
    package_dir.mkdir(parents=True)
    exe = ".exe" if goos == "windows" else ""
    ldflags = " ".join(
        (
            "-buildid=",
            f"-X {BUILDINFO_PATH}.Version={version}",
            f"-X {BUILDINFO_PATH}.Commit={commit}",
            f"-X {BUILDINFO_PATH}.BuildDate={build_date}",
        )
    )
    build_binary(repo, repo, "./cmd/doctorcode", package_dir / f"doctorcode{exe}", goos, goarch, ldflags)
    build_binary(repo, repo / "mcp", "./cmd/doctorcode-mcp", package_dir / f"doctorcode-mcp{exe}", goos, goarch, ldflags)

    shutil.copyfile(repo / "README.md", package_dir / "README.md")
    shutil.copyfile(repo / "LICENSE", package_dir / "LICENSE")
    skill_dir = package_dir / "skills" / "doctorcode"
    skill_dir.mkdir(parents=True)
    shutil.copyfile(repo / "skills" / "doctorcode" / "SKILL.md", skill_dir / "SKILL.md")

    if goos != "windows":
        (package_dir / "doctorcode").chmod(0o755)
        (package_dir / "doctorcode-mcp").chmod(0o755)


def iter_archive_paths(package_dir: Path):
    yield package_dir
    yield from sorted(package_dir.rglob("*"), key=lambda item: item.relative_to(package_dir.parent).as_posix())


def write_tar_gz(package_dir: Path, archive: Path, epoch: int) -> None:
    with archive.open("wb") as raw:
        with gzip.GzipFile(filename="", mode="wb", fileobj=raw, compresslevel=9, mtime=epoch) as gz:
            with tarfile.open(fileobj=gz, mode="w", format=tarfile.PAX_FORMAT) as tar:
                for path in iter_archive_paths(package_dir):
                    name = path.relative_to(package_dir.parent).as_posix()
                    info = tarfile.TarInfo(name + ("/" if path.is_dir() else ""))
                    info.uid = 0
                    info.gid = 0
                    info.uname = ""
                    info.gname = ""
                    info.mtime = epoch
                    if path.is_dir():
                        info.type = tarfile.DIRTYPE
                        info.mode = 0o755
                        tar.addfile(info)
                        continue
                    data = path.read_bytes()
                    info.size = len(data)
                    info.mode = 0o755 if path.name in {"doctorcode", "doctorcode-mcp"} else 0o644
                    tar.addfile(info, io.BytesIO(data))


def write_zip(package_dir: Path, archive: Path, zip_time: tuple[int, int, int, int, int, int]) -> None:
    with zipfile.ZipFile(archive, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for path in iter_archive_paths(package_dir):
            name = path.relative_to(package_dir.parent).as_posix()
            is_dir = path.is_dir()
            if is_dir:
                name += "/"
            info = zipfile.ZipInfo(name, date_time=zip_time)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.create_system = 3
            mode = (stat.S_IFDIR | 0o755) if is_dir else (stat.S_IFREG | 0o644)
            info.external_attr = mode << 16
            if is_dir:
                zf.writestr(info, b"")
            else:
                zf.writestr(info, path.read_bytes())


def main() -> int:
    args = parse_args()
    epoch, zip_time = validate_metadata(args.version, args.commit, args.date)
    repo = Path(__file__).resolve().parent.parent
    output = Path(args.output).resolve()
    if output.exists():
        shutil.rmtree(output)
    output.mkdir(parents=True)

    artifacts = []
    with tempfile.TemporaryDirectory(prefix="doctorcode-release-") as tmp:
        staging = Path(tmp)
        for goos, goarch in TARGETS:
            package_name = f"doctorcode_{args.version}_{goos}_{goarch}"
            package_dir = staging / package_name
            populate_package(repo, package_dir, args.version, args.commit, args.date, goos, goarch)
            extension = ".zip" if goos == "windows" else ".tar.gz"
            archive = output / f"{package_name}{extension}"
            if goos == "windows":
                write_zip(package_dir, archive, zip_time)
            else:
                write_tar_gz(package_dir, archive, epoch)
            artifacts.append(
                {
                    "filename": archive.name,
                    "os": goos,
                    "arch": goarch,
                    "format": "zip" if goos == "windows" else "tar.gz",
                    "sha256": sha256(archive),
                    "size": archive.stat().st_size,
                }
            )

    manifest = {
        "schema_version": 1,
        "product": "DoctorCode",
        "version": args.version,
        "commit": args.commit,
        "build_date": args.date,
        "artifacts": artifacts,
    }
    manifest_path = output / "release-manifest.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8", newline="\n")

    checksum_entries = [(item["sha256"], item["filename"]) for item in artifacts]
    checksum_entries.append((sha256(manifest_path), manifest_path.name))
    checksum_entries.sort(key=lambda item: item[1])
    (output / "checksums.txt").write_text(
        "".join(f"{digest}  {name}\n" for digest, name in checksum_entries),
        encoding="utf-8",
        newline="\n",
    )
    print(f"M17_RELEASE_BUILD=PASS artifacts={len(artifacts)} output={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
