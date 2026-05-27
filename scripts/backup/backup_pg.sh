#!/usr/bin/env bash
# backup_pg.sh - PostgreSQL backup script for baxi project
# Usage: ./backup_pg.sh [--output-dir DIR] [--compress]
set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────
DATABASE_URL="${DATABASE_URL:-postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable}"
OUTPUT_DIR="./backups"
COMPRESS=true
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
BACKUP_NAME="baxi_${TIMESTAMP}"

# ── Parse args ────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --no-compress) COMPRESS=false; shift ;;
    -h|--help)
      echo "Usage: $0 [--output-dir DIR] [--no-compress]"
      echo "  --output-dir DIR   Directory to store backups (default: ./backups)"
      echo "  --no-compress      Skip gzip compression"
      exit 0 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ── Prerequisites ─────────────────────────────────────────────────────────
command -v pg_dump >/dev/null 2>&1 || { echo "ERROR: pg_dump not found. Install postgresql-client."; exit 1; }
command -v gzip >/dev/null 2>&1   || { echo "ERROR: gzip not found."; exit 1; }

# ── Extract connection details from DATABASE_URL ──────────────────────────
# Parse: postgres://user:pass@host:port/dbname?sslmode=...
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

# ── Create output directory ───────────────────────────────────────────────
mkdir -p "$OUTPUT_DIR"

# ── Backup ────────────────────────────────────────────────────────────────
echo "[$(date)] Starting backup: ${BACKUP_NAME}"
echo "  Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}"
echo "  Output:   ${OUTPUT_DIR}/"

BACKUP_FILE="${OUTPUT_DIR}/${BACKUP_NAME}.sql"

pg_dump \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --no-owner \
  --no-privileges \
  -f "$BACKUP_FILE"

echo "  Backup created: ${BACKUP_FILE}"

# ── Compress ──────────────────────────────────────────────────────────────
if [ "$COMPRESS" = true ]; then
  gzip "$BACKUP_FILE"
  BACKUP_FILE="${BACKUP_FILE}.gz"
  echo "  Compressed: ${BACKUP_FILE}"
fi

# ── Verify ────────────────────────────────────────────────────────────────
if [ -f "$BACKUP_FILE" ]; then
  FILE_SIZE=$(stat -c%s "$BACKUP_FILE" 2>/dev/null || stat -f%z "$BACKUP_FILE" 2>/dev/null || echo "unknown")
  echo "[$(date)] Backup complete: ${BACKUP_FILE} (${FILE_SIZE} bytes)"
else
  echo "ERROR: Backup file not found after creation!"
  exit 1
fi

# ── Cleanup old backups (keep last 10) ────────────────────────────────────
BACKUP_COUNT=$(ls -1 "${OUTPUT_DIR}"/baxi_*.sql* 2>/dev/null | wc -l)
if [ "$BACKUP_COUNT" -gt 10 ]; then
  echo "  Cleaning old backups (keeping last 10)..."
  ls -1t "${OUTPUT_DIR}"/baxi_*.sql* | tail -n +11 | xargs rm -f
  echo "  Removed $((BACKUP_COUNT - 10)) old backup(s)"
fi

unset PGPASSWORD
