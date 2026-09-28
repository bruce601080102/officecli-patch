from os import environ
from pathlib import Path
from setuptools import find_packages, setup

root = Path(__file__).parent

setup(
    name="officecli-patch",
    version=environ.get("OFFICECLI_PATCH_VERSION", "0.3.0"),
    description="Cross-platform launcher for officecli-patch GitHub Release binaries",
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
    entry_points={"console_scripts": ["officecli-patch=officecli_patch.launcher:main"]},
)
