# ⚠️ Scripts Status: FROZEN (Phase A)

## What You Need to Know

All Python scripts in this directory have been moved from the project root and renamed with `phaseXX_` prefixes for organization.

### Path Status: BROKEN

Every script uses **hardcoded relative paths** (e.g., `pd.read_csv('olist_orders_dataset.csv')`, `to_csv('outputs/charts/file.png')`). Since the source CSV files have been moved to `data/raw/`, **none of these scripts will run without modification**.

### What Works
- ✅ Scripts can be read for reference — the code logic is intact
- ✅ You can study the analysis patterns and data transformations

### What Doesn't Work
- ❌ `python scripts/phase01_explore_data.py` → FileNotFoundError (CSVs in data/raw/ now)
- ❌ `python scripts/phase02_build_data_model.py` → FileNotFoundError (same issue)
- ❌ All 14 scripts will fail with path errors

### Phase B (Planned)

The next step will:
1. Create `scripts/config.py` with centralized path constants
2. Update all scripts to use config-based paths
3. Run end-to-end tests to verify

### How to Run a Script (Temporary Workaround)

```bash
# Example: run Phase 1 data exploration
cd /path/to/baxi/data/raw  # go where CSVs are
cp /path/to/baxi/scripts/phase01_explore_data.py .  # copy script to data/
python phase01_explore_data.py  # run from data/ (where CSVs live)
rm phase01_explore_data.py  # clean up
```

This is not ideal — it's a temporary workaround until Phase B fixes all paths properly.
