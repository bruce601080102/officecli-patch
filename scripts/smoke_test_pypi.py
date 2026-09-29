#!/usr/bin/env python3
"""Install a built wheel and verify its terminal entry points."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
import platform
import subprocess
import tempfile
from typing import Dict, List, Optional, Tuple
import venv


def environment_executables(root: Path) -> Tuple[Path, Path]:
    if os.name == "nt":
        scripts = root / "Scripts"
        return scripts / "python.exe", scripts / "officecli-patch.exe"
    scripts = root / "bin"
    return scripts / "python", scripts / "officecli-patch"


def run_checked(command: List[str], env: Optional[Dict[str, str]] = None) -> subprocess.CompletedProcess:
    return subprocess.run(command, check=True, capture_output=True, text=True, env=env)


def current_platform_tag() -> str:
    machine = platform.machine().lower()
    arch = "x86_64" if machine in {"amd64", "x86_64"} else "aarch64" if machine in {"arm64", "aarch64"} else None
    if arch is None:
        raise RuntimeError(f"unsupported CPU architecture: {platform.machine()}")
    if sys.platform == "darwin":
        return f"macosx_11_0_{'arm64' if arch == 'aarch64' else 'x86_64'}"
    if sys.platform.startswith("linux"):
        return f"manylinux_2_17_{arch}"
    if os.name == "nt":
        return f"win_{'arm64' if arch == 'aarch64' else 'amd64'}"
    raise RuntimeError(f"unsupported operating system: {sys.platform}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--wheel-dir", type=Path, default=Path("release/pypi"))
    args = parser.parse_args()

    tag = current_platform_tag()
    wheels = sorted(args.wheel_dir.glob(f"officecli_patch-*-{tag}.whl"))
    if len(wheels) != 1:
        parser.error(f"expected exactly one {tag} wheel in {args.wheel_dir}, found {len(wheels)}")

    with tempfile.TemporaryDirectory(prefix="officecli-patch-wheel-") as temporary:
        environment_root = Path(temporary) / "venv"
        venv.EnvBuilder(with_pip=True).create(environment_root)
        python, command = environment_executables(environment_root)

        run_checked([
            str(python), "-m", "pip", "install", "--no-index", "--no-deps", str(wheels[0].resolve())
        ])
        if not command.is_file():
            raise RuntimeError(f"wheel did not install the officecli-patch command: {command}")
        if os.name != "nt" and not os.access(command, os.X_OK):
            raise RuntimeError(f"installed officecli-patch command is not executable: {command}")

        # This exercises the binary copied from the installed wheel. No network
        # override is used: a GitHub download would make this test fail.
        run_checked([str(command), "--version"])
        run_checked([str(python), "-m", "officecli_patch", "--version"])

        print(f"PyPI smoke test passed: {command}")


if __name__ == "__main__":
    main()
