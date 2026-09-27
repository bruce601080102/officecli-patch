from os import environ
from pathlib import Path
from setuptools import find_packages, setup

root = Path(__file__).parent

setup(
    name="officecli-patch",
    version=environ.get("OFFICECLI_PATCH_VERSION", "0.2.0"),
    description="Cross-platform launcher for officecli-patch GitHub Release binaries",
    long_description=(root / "README.md").read_text(encoding="utf-8"),
    long_description_content_type="text/markdown",
    python_requires=">=3.9",
    package_dir={"": "src"},
    packages=find_packages("src"),
    entry_points={"console_scripts": ["officecli-patch=officecli_patch.launcher:main"]},
)
