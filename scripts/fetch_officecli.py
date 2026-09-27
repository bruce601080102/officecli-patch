#!/usr/bin/env python3
"""Download and verify official OfficeCLI release assets without committing them."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import tempfile
from urllib.request import Request, urlopen

REPOSITORY = "iOfficeAI/OfficeCLI"
ASSETS = (
    "officecli-win-x64.exe", "officecli-win-arm64.exe",
    "officecli-linux-x64", "officecli-linux-arm64",
    "officecli-linux-alpine-x64", "officecli-linux-alpine-arm64",
    "officecli-mac-x64", "officecli-mac-arm64",
)


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def request_json(url: str) -> dict:
    headers = {"Accept": "application/vnd.github+json", "User-Agent": "officecli-patch-build"}
    if token := os.getenv("GITHUB_TOKEN"):
        headers["Authorization"] = f"Bearer {token}"
    with urlopen(Request(url, headers=headers), timeout=60) as response:
        return json.load(response)


def download(url: str, destination: Path) -> None:
    headers = {"Accept": "application/octet-stream", "User-Agent": "officecli-patch-build"}
    if token := os.getenv("GITHUB_TOKEN"):
        headers["Authorization"] = f"Bearer {token}"
    with urlopen(Request(url, headers=headers), timeout=300) as response, destination.open("wb") as output:
        while block := response.read(1024 * 1024):
            output.write(block)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", default="latest", help="OfficeCLI tag, e.g. v1.0.152, or latest")
    parser.add_argument("--tools-dir", type=Path, default=Path("tools"))
    args = parser.parse_args()
    endpoint = f"https://api.github.com/repos/{REPOSITORY}/releases/latest"
    if args.version != "latest":
        endpoint = f"https://api.github.com/repos/{REPOSITORY}/releases/tags/{args.version}"
    release = request_json(endpoint)
    assets = {item["name"]: item for item in release.get("assets", [])}
    missing = [name for name in ASSETS if name not in assets]
    if missing:
        raise SystemExit(f"官方 release {release.get('tag_name')} 缺少資產：{', '.join(missing)}")

    args.tools_dir.mkdir(parents=True, exist_ok=True)
    for name in ASSETS:
        asset = assets[name]
        digest = asset.get("digest", "")
        if not digest.startswith("sha256:"):
            raise SystemExit(f"官方資產未提供 SHA-256 digest，拒絕下載：{name}")
        expected = digest.removeprefix("sha256:").lower()
        target = args.tools_dir / name
        if target.is_file() and sha256(target) == expected:
            print(f"已驗證：{name}")
            continue
        with tempfile.NamedTemporaryFile(dir=args.tools_dir, prefix=f".{name}.", delete=False) as temp:
            temporary = Path(temp.name)
        try:
            print(f"下載：{name} ({release['tag_name']})")
            download(asset["browser_download_url"], temporary)
            actual = sha256(temporary)
            if actual != expected:
                raise SystemExit(f"SHA-256 不符：{name}\nexpected {expected}\nactual   {actual}")
            temporary.replace(target)
        finally:
            temporary.unlink(missing_ok=True)


if __name__ == "__main__":
    main()
