"""Run the platform-native officecli-patch binary bundled in this wheel."""
from __future__ import annotations

from contextlib import contextmanager
from importlib import resources
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
from typing import Iterator

from ._release import RELEASE_VERSION


def asset_name() -> str:
    system = platform.system().lower()
    machine = platform.machine().lower()
    arch = "x64" if machine in {"amd64", "x86_64"} else "arm64" if machine in {"arm64", "aarch64"} else None
    if arch is None:
        raise RuntimeError(f"不支援的 CPU 架構：{platform.machine()}")
    if system == "windows":
        return f"officecli-patch-win-{arch}.exe"
    if system == "darwin":
        return f"officecli-patch-mac-{arch}"
    if system == "linux":
        libc = platform.libc_ver()[0].lower()
        try:
            ldd = subprocess.run(["ldd", "--version"], capture_output=True, text=True, check=False).stdout.lower()
        except OSError:
            ldd = ""
        family = "linux-alpine" if "musl" in libc or "musl" in ldd else "linux"
        return f"officecli-patch-{family}-{arch}"
    raise RuntimeError(f"不支援的作業系統：{platform.system()}")


def package_version() -> str:
    if RELEASE_VERSION == "CHANGE_ME":
        raise RuntimeError("此 PyPI wheel 尚未設定版本。請使用正式發布的 wheel。")
    return RELEASE_VERSION.removeprefix("v")


def cache_dir() -> Path:
    base = Path(os.getenv("LOCALAPPDATA", "")) if os.name == "nt" else Path(os.getenv("XDG_CACHE_HOME", Path.home() / ".cache"))
    # A wheel upgrade must not reuse a binary from an older wheel.
    return base / "officecli-patch" / "bin" / package_version()


@contextmanager
def bundled_binary(name: str) -> Iterator[Path]:
    asset = resources.files("officecli_patch").joinpath("binaries", name)
    if not asset.is_file():
        raise RuntimeError(
            f"安裝的 officecli-patch wheel 未包含目前平台的執行檔：{name}。請重新安裝相符平台的 wheel。"
        )
    with resources.as_file(asset) as source:
        yield source


def binary() -> Path:
    override = os.getenv("OFFICECLI_PATCH_BINARY")
    if override:
        path = Path(override).expanduser()
        if not path.is_file():
            raise RuntimeError(f"OFFICECLI_PATCH_BINARY 找不到檔案：{path}")
        return path

    name = asset_name()
    target = cache_dir() / name
    if target.is_file():
        return target
    target.parent.mkdir(parents=True, exist_ok=True)
    temporary = target.with_suffix(target.suffix + ".extract")
    try:
        with bundled_binary(name) as source:
            shutil.copyfile(source, temporary)
        temporary.replace(target)
        if os.name != "nt":
            target.chmod(0o755)
    finally:
        temporary.unlink(missing_ok=True)
    return target


def main() -> None:
    try:
        result = subprocess.run([str(binary()), *sys.argv[1:]])
    except (OSError, RuntimeError) as error:
        print(f"officecli-patch: {error}", file=sys.stderr)
        raise SystemExit(1)
    raise SystemExit(result.returncode)
