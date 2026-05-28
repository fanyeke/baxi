#!/usr/bin/env python3
"""Copy all YAML config files to migration baseline snapshot."""
import os, shutil

BASE = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CONFIG_DIR = os.path.join(BASE, "config")
OUT_DIR = os.path.join(BASE, "migration_baseline", "configs_snapshot")
os.makedirs(OUT_DIR, exist_ok=True)

count = 0
for fname in sorted(os.listdir(CONFIG_DIR)):
    if fname.endswith((".yml", ".yaml")):
        src = os.path.join(CONFIG_DIR, fname)
        dst = os.path.join(OUT_DIR, fname)
        shutil.copy2(src, dst)
        print(f"OK: {fname}")
        count += 1

print(f"Copied {count} YAML files to {OUT_DIR}")
