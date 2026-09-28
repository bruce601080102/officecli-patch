#!/usr/bin/env python3
"""Install a built wheel and verify its terminal entry points."""
from __future__ import annotations

import argparse
import os
from pathlib import Path
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


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--wheel-dir", type=Path, default=Path("release/pypi"))
    args = parser.parse_args()

    wheels = sorted(args.wheel_dir.glob("officecli_patch-*.whl"))
    if len(wheels) != 1:
        parser.error(f"expected exactly one officecli-patch wheel in {args.wheel_dir}, found {len(wheels)}")

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

        # Use Python itself as a harmless stand-in for the downloaded native
        # release. This verifies argument forwarding without network access.
        smoke_env = {**os.environ, "OFFICECLI_PATCH_BINARY": str(python)}
        direct = run_checked([str(command), "--version"], env=smoke_env)
        module = run_checked([str(python), "-m", "officecli_patch", "--version"], env=smoke_env)
        if not direct.stdout.startswith("Python ") or not module.stdout.startswith("Python "):
            raise RuntimeError("installed launchers did not forward --version to the executable")

        print(f"PyPI smoke test passed: {command}")


if __name__ == "__main__":
    main()
