#!/usr/bin/env python3
import sqlite3, os
DB_PATH = os.path.join(os.path.dirname(__file__), "../../data/olist_ops.db")
OUT_PATH = os.path.join(os.path.dirname(__file__), "../../migration_baseline/sqlite_schema.sql")
conn = sqlite3.connect(DB_PATH)
tables = [r[0] for r in conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").fetchall()]
with open(OUT_PATH, "w") as f:
    for table in tables:
        ddl_rows = conn.execute(f"SELECT sql FROM sqlite_master WHERE type='table' AND name='{table}'").fetchall()
        for (ddl,) in ddl_rows:
            if ddl:
                f.write(ddl + ";\n\n")
conn.close()
print(f"Exported {len(tables)} tables to {OUT_PATH}")
