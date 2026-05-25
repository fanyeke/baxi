"""
Backward-compatibility shim for scripts.config.

DEPRECATED: Import from core.config instead. This module will be removed in a future release.
"""

import os
import sys
import warnings

# Ensure project root is on sys.path so core.config can be resolved
# when scripts are run directly (e.g. python3 scripts/foo.py)
_PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if _PROJECT_ROOT not in sys.path:
    sys.path.insert(0, _PROJECT_ROOT)

warnings.warn(
    "scripts.config is deprecated. Import from core.config instead.",
    DeprecationWarning,
    stacklevel=2,
)

from core.config import *  # noqa: F401,F403
