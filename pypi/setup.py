from os import environ
from setuptools import find_packages, setup

setup(
    name="officecli-patch",
    version=environ.get("OFFICECLI_PATCH_VERSION", "0.2.0"),
    description="Cross-platform launcher for officecli-patch GitHub Release binaries",
    python_requires=">=3.9",
    package_dir={"": "src"},
    packages=find_packages("src"),
    entry_points={"console_scripts": ["officecli-patch=officecli_patch.launcher:main"]},
)
