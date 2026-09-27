#!/usr/bin/env python3
"""Build a PyPI launcher wheel with the repository name embedded."""
from __future__ import annotations

import argparse
from pathlib import Path
import subprocess
import sys


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repository", required=True, help="GitHub owner/repository")
    parser.add_argument("--version", required=True)
    parser.add_argument("--output", type=Path, default=Path("release/pypi"))
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    package_root = root / "pypi"
    release_file = package_root / "src" / "officecli_patch" / "_release.py"
    release_file.write_text(f'REPOSITORY = {args.repository!r}\n', encoding="utf-8")
    args.output.mkdir(parents=True, exist_ok=True)
    subprocess.run([
        sys.executable, "-m", "build", "--wheel", "--sdist", "--outdir", str(args.output), str(package_root)
    ], check=True, env={**__import__("os").environ, "OFFICECLI_PATCH_VERSION": args.version})


if __name__ == "__main__":
    main()
