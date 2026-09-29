from __future__ import annotations

import os
from pathlib import Path
import runpy
import sys
import tempfile
import types
import unittest
from unittest import mock

import setuptools


PYPI_ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOT = PYPI_ROOT / "src"
sys.path.insert(0, str(SOURCE_ROOT))

from officecli_patch import launcher  # noqa: E402


class PackagingTests(unittest.TestCase):
    def test_setup_installs_terminal_command(self) -> None:
        captured = {}
        with mock.patch.object(setuptools, "find_packages", return_value=["officecli_patch"]), mock.patch.object(
            setuptools, "setup", side_effect=lambda **kwargs: captured.update(kwargs)
        ), mock.patch.dict(
            os.environ, {"OFFICECLI_PATCH_VERSION": "9.9.9"}
        ):
            runpy.run_path(str(PYPI_ROOT / "setup.py"), run_name="__main__")

        self.assertEqual(captured["version"], "9.9.9")
        self.assertIn(
            "officecli-patch=officecli_patch.launcher:main",
            captured["entry_points"]["console_scripts"],
        )
        self.assertEqual(captured["package_data"], {"officecli_patch": ["binaries/*"]})

    def test_binary_override(self) -> None:
        with tempfile.NamedTemporaryFile() as executable, mock.patch.dict(
            os.environ, {"OFFICECLI_PATCH_BINARY": executable.name}
        ):
            self.assertEqual(launcher.binary(), Path(executable.name))

    def test_binary_extracts_the_bundled_platform_binary(self) -> None:
        name = "officecli-patch-win-x64.exe"
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "binaries" / name
            source.parent.mkdir()
            source.write_bytes(b"bundled executable")
            target_directory = root / "cache"
            with mock.patch.object(launcher, "asset_name", return_value=name), mock.patch.object(
                launcher, "cache_dir", return_value=target_directory
            ), mock.patch.object(launcher.resources, "files", return_value=root):
                extracted = launcher.binary()

            self.assertEqual(extracted, target_directory / name)
            self.assertEqual(extracted.read_bytes(), b"bundled executable")

    def test_binary_reports_a_wheel_without_the_platform_asset(self) -> None:
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(
            launcher.resources, "files", return_value=Path(temporary)
        ):
            with self.assertRaisesRegex(RuntimeError, "未包含目前平台的執行檔"):
                with launcher.bundled_binary("officecli-patch-win-x64.exe"):
                    pass

    def test_main_forwards_arguments_and_exit_code(self) -> None:
        executable = Path(sys.executable)
        completed = types.SimpleNamespace(returncode=7)
        with mock.patch.object(launcher, "binary", return_value=executable), mock.patch.object(
            launcher.subprocess, "run", return_value=completed
        ) as run, mock.patch.object(sys, "argv", ["officecli-patch", "--version"]):
            with self.assertRaises(SystemExit) as exit_error:
                launcher.main()

        self.assertEqual(exit_error.exception.code, 7)
        run.assert_called_once_with([str(executable), "--version"])

    def test_main_reports_binary_errors_without_a_traceback(self) -> None:
        with mock.patch.object(launcher, "binary", side_effect=RuntimeError("binary unavailable")), mock.patch.object(
            sys, "stderr"
        ) as stderr:
            with self.assertRaises(SystemExit) as exit_error:
                launcher.main()

        self.assertEqual(exit_error.exception.code, 1)
        stderr.write.assert_has_calls(
            [
                mock.call("officecli-patch: binary unavailable"),
                mock.call("\n"),
            ]
        )


if __name__ == "__main__":
    unittest.main()
