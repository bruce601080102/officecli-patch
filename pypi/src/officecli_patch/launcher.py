"""Install and run the matching officecli-patch GitHub Release asset."""
from __future__ import annotations

import hashlib
import os
from pathlib import Path
import platform
import subprocess
import sys
from urllib.request import Request, urlopen

from ._release import REPOSITORY


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


def cache_dir() -> Path:
    base = Path(os.getenv("LOCALAPPDATA", "")) if os.name == "nt" else Path(os.getenv("XDG_CACHE_HOME", Path.home() / ".cache"))
    return base / "officecli-patch" / "bin"


def download(url: str, target: Path) -> None:
    request = Request(url, headers={"User-Agent": "officecli-patch-pypi"})
    with urlopen(request, timeout=180) as response, target.open("wb") as output:
        while block := response.read(1024 * 1024):
            output.write(block)


def expected_checksum(name: str) -> str:
    request = Request(
        f"https://github.com/{REPOSITORY}/releases/latest/download/checksums.txt",
        headers={"User-Agent": "officecli-patch-pypi"},
    )
    with urlopen(request, timeout=60) as response:
        for line in response.read().decode("utf-8").splitlines():
            digest, filename = line.split(maxsplit=1)
            if filename.strip().lstrip("*") == name:
                return digest.lower()
    raise RuntimeError(f"Release checksums.txt 找不到 {name}")


def binary() -> Path:
    if REPOSITORY.startswith("CHANGE_ME/"):
        raise RuntimeError("此 PyPI launcher 尚未設定 GitHub repository。請使用正式發布的 wheel。")
    name = asset_name()
    target = cache_dir() / name
    if target.is_file():
        return target
    target.parent.mkdir(parents=True, exist_ok=True)
    temporary = target.with_suffix(target.suffix + ".download")
    try:
        download(f"https://github.com/{REPOSITORY}/releases/latest/download/{name}", temporary)
        actual = hashlib.sha256(temporary.read_bytes()).hexdigest()
        if actual != expected_checksum(name):
            raise RuntimeError(f"下載檔案的 SHA-256 驗證失敗：{name}")
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
