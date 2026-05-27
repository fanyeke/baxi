#!/usr/bin/env bash
# restore_pg.sh - PostgreSQL restore script for baxi project
# Usage: ./restore_pg.sh <backup_file> [--dry-run]
# WARNING: This will DROP and recreate the target database!
set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────
DATABASE_URL="${DATABASE_URL:-postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable}"
DRY_RUN=false

# ── Parse args ────────────────────────────────────────────────────────────
BACKUP_FILE=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help)
      echo "Usage: $0 <backup_file> [--dry-run]"
      echo "  backup_file   Path to .sql or .sql.gz backup file"
      echo "  --dry-run     Show what would happen without executing"
      exit 0 ;;
    -*) echo "Unknown option: $1"; exit 1 ;;
    *) BACKUP_FILE="$1"; shift ;;
  esac
done

if [ -z "$BACKUP_FILE" ]; then
  echo "ERROR: No backup file specified."
  echo "Usage: $0 <backup_file> [--dry-run]"
  echo ""
  echo "Available backups:"
  ls -1t ./backups/baxi_*.sql* 2>/dev/null | head -5 || echo "  (none found in ./backups/)"
  exit 1
fi

if [ ! -f "$BACKUP_FILE" ]; then
  echo "ERROR: Backup file not found: $BACKUP_FILE"
  exit 1
fi

# ── Prerequisites ─────────────────────────────────────────────────────────
command -v psql >/dev/null 2>&1  || { echo "ERROR: psql not found. Install postgresql-client."; exit 1; }
command -v gzip >/dev/null 2>&1  || { echo "ERROR: gzip not found."; exit 1; }

# ── Extract connection details from DATABASE_URL ──────────────────────────
URL_NO_SCHEME="${DATABASE_URL#postgres://}"
URL_NO_SCHEME="${URL_NO_SCHEME#postgresql://}"
DB_USER="${URL_NO_SCHEME%%:*}"
URL_REST="${URL_NO_SCHEME#*:}"
DB_PASS="${URL_REST%%@*}"
URL_REST="${URL_REST#*@}"
DB_HOST="${URL_REST%%:*}"
URL_REST="${URL_REST#*:}"
DB_PORT="${URL_REST%%/*}"
DB_PORT="${DB_PORT%%\?*}"
DB_NAME="${URL_REST#*/}"
DB_NAME="${DB_NAME%%\?*}"

export PGPASSWORD="$DB_PASS"

# ── Confirmation ──────────────────────────────────────────────────────────
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                    ⚠️  DATABASE RESTORE                     ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║  Backup file: $(basename "$BACKUP_FILE")"
echo "║  Target:      ${DB_NAME}@${DB_HOST}:${DB_PORT}"
echo "║  Dry run:     ${DRY_RUN}"
echo "║"
echo "║  WARNING: This will DROP and recreate the database!"
echo "║  All existing data will be LOST."
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
read -p "Type 'RESTORE' to confirm (anything else aborts): " CONFIRM
if [ "$CONFIRM" != "RESTORE" ]; then
  echo "Aborted."
  unset PGPASSWORD
  exit 0
fi

# ── Dry run check ─────────────────────────────────────────────────────────
if [ "$DRY_RUN" = true ]; then
  echo "[DRY RUN] Would execute:"
  echo "  1. DROP DATABASE ${DB_NAME}"
  echo "  2. CREATE DATABASE ${DB_NAME}"
  if [[ "$BACKUP_FILE" == *.gz ]]; then
    echo "  3. zcat ${BACKUP_FILE} | psql to ${DB_NAME}"
  else
    echo "  3. psql -f ${BACKUP_FILE} to ${DB_NAME}"
  fi
  echo ""
  echo "To execute for real, remove --dry-run flag."
  unset PGPASSWORD
  exit 0
fi

# ── Pre-restore backup ────────────────────────────────────────────────────
echo "[$(date)] Taking pre-restore safety backup..."
PRE_RESTORE_FILE="./backups/pre_restore_$(date +%Y%m%d_%H%M%S).sql"
pg_dump -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" --no-owner --no-privileges -f "$PRE_RESTORE_FILE" 2>/dev/null || true
gzip -f "$PRE_RESTORE_FILE" 2>/dev/null || true
echo "  Safety backup: ${PRE_RESTORE_FILE}.gz"

# ── Drop and recreate database ────────────────────────────────────────────
echo "[$(date)] Dropping database: ${DB_NAME}"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>&1
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE ${DB_NAME};" 2>&1
echo "  Database recreated."

# ── Restore from backup ───────────────────────────────────────────────────
echo "[$(date)] Restoring from: ${BACKUP_FILE}"
if [[ "$BACKUP_FILE" == *.gz ]]; then
  zcat "$BACKUP_FILE" | psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -q 2>&1
else
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$BACKUP_FILE" -q 2>&1
fi

# ── Verify ────────────────────────────────────────────────────────────────
TABLE_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
echo ""
echo "[$(date)] Restore complete. Tables in database: ${TABLE_COUNT}"
unset PGPASSWORD
