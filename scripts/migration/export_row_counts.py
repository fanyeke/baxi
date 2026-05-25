#!/usr/bin/env python3
import sqlite3, json, os
DB_PATH = os.path.join(os.path.dirname(__file__), "../../data/olist_ops.db")
OUT_PATH = os.path.join(os.path.dirname(__file__), "../../migration_baseline/table_counts.json")
conn = sqlite3.connect(DB_PATH)
tables = [r[0] for r in conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").fetchall()]
counts = {}
for table in tables:
    count = conn.execute(f'SELECT COUNT(*) FROM "{table}"').fetchone()[0]
    counts[table] = count
with open(OUT_PATH, "w") as f:
    json.dump(counts, f, indent=2)
conn.close()
total = sum(counts.values())
print(f"Exported {len(tables)} tables ({total:,} total rows) to {OUT_PATH}")
