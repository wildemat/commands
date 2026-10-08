#!/usr/bin/env python3
"""Usage: launch.py <project>   (loads projects/<project>.py, which defines TABS)"""
import importlib
import pathlib
import sys

sys.path.insert(0, str(pathlib.Path(__file__).parent))

from lib import open_project  # noqa: E402

if len(sys.argv) != 2:
    sys.exit(__doc__)

project = importlib.import_module(f"projects.{sys.argv[1]}")
open_project(project.TABS, getattr(project, "WINDOW_TITLE", None))
