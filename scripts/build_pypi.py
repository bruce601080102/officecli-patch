#!/usr/bin/env python3
"""Build one platform-specific PyPI wheel for every native release binary."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


# The platform tags intentionally describe the bundled executable, rather than
# the Python code (which itself is pure Python). This lets pip select exactly
# one wheel and means installation never needs to reach GitHub.
TARGETS = (
    ("officecli-patch-win-x64.exe", "win_amd64"),
    ("officecli-patch-win-arm64.exe", "win_arm64"),
    ("officecli-patch-linux-x64", "manylinux_2_17_x86_64"),
    ("officecli-patch-linux-arm64", "manylinux_2_17_aarch64"),
    ("officecli-patch-linux-alpine-x64", "musllinux_1_2_x86_64"),
    ("officecli-patch-linux-alpine-arm64", "musllinux_1_2_aarch64"),
    ("officecli-patch-mac-x64", "macosx_10_15_x86_64"),
    ("officecli-patch-mac-arm64", "macosx_11_0_arm64"),
)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--version", required=True)
    parser.add_argument("--release-dir", type=Path, default=Path("release"))
    parser.add_argument("--output", type=Path, default=Path("release/pypi"))
    parser.add_argument("--no-isolation", action="store_true", help="reuse the current Python build environment")
    args = parser.parse_args()
    version = args.version.removeprefix("v")
    if not version:
        parser.error("--version must not be empty")

    root = Path(__file__).resolve().parents[1]
    package_root = root / "pypi"
    assets = [(args.release_dir / asset, asset, platform_tag) for asset, platform_tag in TARGETS]
    missing = [str(path) for path, _asset, _tag in assets if not path.is_file()]
    if missing:
        parser.error(f"release binaries are missing: {', '.join(missing)}")

    args.output.mkdir(parents=True, exist_ok=True)
    # Build from a disposable copy. Apart from keeping the checkout clean, this
    # makes two release builds unable to leak binaries into one another.
    with tempfile.TemporaryDirectory(prefix="officecli-patch-pypi-build-") as temporary:
        temporary_package_root = Path(temporary) / "pypi"
        shutil.copytree(
            package_root,
            temporary_package_root,
            ignore=shutil.ignore_patterns("build", "*.egg-info", "__pycache__"),
        )
        release_file = temporary_package_root / "src" / "officecli_patch" / "_release.py"
        binaries_dir = temporary_package_root / "src" / "officecli_patch" / "binaries"
        for stale in binaries_dir.iterdir():
            if stale.name != ".gitkeep":
                stale.unlink()
        release_file.write_text(f"RELEASE_VERSION = {version!r}\n", encoding="utf-8")
        for source, asset, platform_tag in assets:
            # setuptools retains build/lib between invocations. Remove it so a
            # wheel can never accidentally contain a prior platform's binary.
            shutil.rmtree(temporary_package_root / "build", ignore_errors=True)
            shutil.rmtree(temporary_package_root / "src" / "officecli_patch.egg-info", ignore_errors=True)
            for stale in binaries_dir.iterdir():
                if stale.name != ".gitkeep":
                    stale.unlink()
            shutil.copyfile(source, binaries_dir / asset)
            command = [
                sys.executable,
                "-m",
                "build",
                "--wheel",
                "--outdir",
                str(args.output),
                f"--config-setting=--build-option=--plat-name={platform_tag}",
            ]
            if args.no_isolation:
                command.append("--no-isolation")
            command.append(str(temporary_package_root))
            subprocess.run(
                command,
                check=True,
                env={**os.environ, "OFFICECLI_PATCH_VERSION": version},
            )


if __name__ == "__main__":
    main()
