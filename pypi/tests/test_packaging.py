from __future__ import annotations

import os
from pathlib import Path
import runpy
import sys
import tempfile
import types
import unittest
from unittest import mock
from urllib.error import URLError


PYPI_ROOT = Path(__file__).resolve().parents[1]
SOURCE_ROOT = PYPI_ROOT / "src"
sys.path.insert(0, str(SOURCE_ROOT))

from officecli_patch import launcher  # noqa: E402


class PackagingTests(unittest.TestCase):
    @staticmethod
    def checksum_response(contents: str) -> mock.MagicMock:
        response = mock.MagicMock()
        response.read.return_value = contents.encode("utf-8")
        response.__enter__.return_value = response
        return response

    def test_setup_installs_terminal_command(self) -> None:
        captured = {}
        setuptools = types.ModuleType("setuptools")
        setuptools.find_packages = lambda *_args, **_kwargs: ["officecli_patch"]
        setuptools.setup = lambda **kwargs: captured.update(kwargs)

        with mock.patch.dict(sys.modules, {"setuptools": setuptools}), mock.patch.dict(
            os.environ, {"OFFICECLI_PATCH_VERSION": "9.9.9"}
        ):
            runpy.run_path(str(PYPI_ROOT / "setup.py"), run_name="__main__")

        self.assertEqual(captured["version"], "9.9.9")
        self.assertIn(
            "officecli-patch=officecli_patch.launcher:main",
            captured["entry_points"]["console_scripts"],
        )

    def test_binary_override(self) -> None:
        with tempfile.NamedTemporaryFile() as executable, mock.patch.dict(
            os.environ, {"OFFICECLI_PATCH_BINARY": executable.name}
        ):
            self.assertEqual(launcher.binary(), Path(executable.name))

    def test_expected_checksum_ignores_malformed_lines(self) -> None:
        checksum = "a" * 64
        response = self.checksum_response(f"temporary upstream response\n{checksum}  officecli-patch-win-x64.exe\n")
        with mock.patch.object(launcher, "urlopen", return_value=response):
            self.assertEqual(launcher.expected_checksum("officecli-patch-win-x64.exe", "v9.9.9"), checksum)

    def test_expected_checksum_reports_missing_asset_after_malformed_content(self) -> None:
        response = self.checksum_response("Bad Gateway\n")
        with mock.patch.object(launcher, "urlopen", return_value=response):
            with self.assertRaisesRegex(RuntimeError, "checksums.txt 找不到 officecli-patch-win-x64.exe"):
                launcher.expected_checksum("officecli-patch-win-x64.exe", "v9.9.9")

    def test_expected_checksum_rejects_invalid_digest(self) -> None:
        response = self.checksum_response("not-a-hash  officecli-patch-win-x64.exe\n")
        with mock.patch.object(launcher, "urlopen", return_value=response):
            with self.assertRaisesRegex(RuntimeError, "SHA-256 格式無效"):
                launcher.expected_checksum("officecli-patch-win-x64.exe", "v9.9.9")

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

    def test_main_reports_download_errors_without_a_traceback(self) -> None:
        with mock.patch.object(launcher, "binary", side_effect=URLError("network unavailable")), mock.patch.object(
            sys, "stderr"
        ) as stderr:
            with self.assertRaises(SystemExit) as exit_error:
                launcher.main()

        self.assertEqual(exit_error.exception.code, 1)
        stderr.write.assert_has_calls(
            [
                mock.call("officecli-patch: <urlopen error network unavailable>"),
                mock.call("\n"),
            ]
        )


if __name__ == "__main__":
    unittest.main()
