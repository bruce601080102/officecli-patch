from os import environ
from pathlib import Path
from setuptools import find_packages, setup
from setuptools.command.bdist_wheel import bdist_wheel

root = Path(__file__).parent


class PlatformBinaryWheel(bdist_wheel):
    """Tag a wheel for its bundled executable, while keeping Python ABI-neutral."""

    def finalize_options(self):
        super().finalize_options()
        self.root_is_pure = False

    def get_tag(self):
        _python, _abi, platform = super().get_tag()
        return "py3", "none", platform

setup(
    name="officecli-patch",
    version=environ.get("OFFICECLI_PATCH_VERSION", "0.3.0"),
    description="Cross-platform officecli-patch command with a bundled native binary",
    long_description=(root / "README.md").read_text(encoding="utf-8"),
    long_description_content_type="text/markdown",
    url="https://github.com/bruce601080102/officecli-patch",
    project_urls={
        "Homepage": "https://github.com/bruce601080102/officecli-patch",
        "Source": "https://github.com/bruce601080102/officecli-patch",
        "Issues": "https://github.com/bruce601080102/officecli-patch/issues",
        "Official OfficeCLI": "https://github.com/iOfficeAI/OfficeCLI",
    },
    python_requires=">=3.9",
    package_dir={"": "src"},
    packages=find_packages("src"),
    package_data={"officecli_patch": ["binaries/*"]},
    zip_safe=False,
    cmdclass={"bdist_wheel": PlatformBinaryWheel},
    entry_points={"console_scripts": ["officecli-patch=officecli_patch.launcher:main"]},
)
