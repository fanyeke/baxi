#!/usr/bin/env bash
# rollback_phase.sh - Rollback specific migration using goose
# Usage: ./rollback_phase.sh <migration_number> [--dry-run]
# Example: ./rollback_phase.sh 5  (rolls back migration 005)
set -euo pipefail

# ── Defaults ──────────────────────────────────────────────────────────────
DATABASE_URL="${DATABASE_URL:-postgres://baxi:baxi_dev@localhost:5432/baxi?sslmode=disable}"
MIGRATIONS_DIR="./migrations"
DRY_RUN=false

# ── Parse args ────────────────────────────────────────────────────────────
MIGRATION_NUM=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help)
      echo "Usage: $0 <migration_number> [--dry-run]"
      echo "  migration_number   Migration number to roll back (e.g. 5 for 005_init_schemas.sql)"
      echo "  --dry-run          Show what would happen without executing"
      echo ""
      echo "Examples:"
      echo "  $0 5              # Roll back migration 005"
      echo "  $0 5 --dry-run    # Preview rollback without executing"
      exit 0 ;;
    -*) echo "Unknown option: $1"; exit 1 ;;
    *)
      if [ -z "$MIGRATION_NUM" ]; then
        MIGRATION_NUM="$1"
      else
        echo "ERROR: Unexpected argument: $1"
        exit 1
      fi
      shift ;;
  esac
done

if [ -z "$MIGRATION_NUM" ]; then
  echo "ERROR: No migration number specified."
  echo "Usage: $0 <migration_number> [--dry-run]"
  echo ""
  echo "Available migrations:"
  ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sed 's|.*/||' || echo "  (none found)"
  exit 1
fi

# ── Prerequisites ─────────────────────────────────────────────────────────
command -v goose >/dev/null 2>&1 || { echo "ERROR: goose not found. Install: go install github.com/pressly/goose/v3/cmd/goose@latest"; exit 1; }

# ── Find migration file ───────────────────────────────────────────────────
PADDED_NUM=$(printf "%03d" "$MIGRATION_NUM")
MIGRATION_FILE=$(ls "$MIGRATIONS_DIR"/${PADDED_NUM}_*.sql 2>/dev/null | head -1)

if [ -z "$MIGRATION_FILE" ] || [ ! -f "$MIGRATION_FILE" ]; then
  echo "ERROR: Migration ${PADDED_NUM} not found in ${MIGRATIONS_DIR}/"
  echo ""
  echo "Available migrations:"
  ls -1 "$MIGRATIONS_DIR"/*.sql 2>/dev/null | sed 's|.*/||' || echo "  (none found)"
  exit 1
fi

MIGRATION_NAME=$(basename "$MIGRATION_FILE" .sql)

# ── Show current status ───────────────────────────────────────────────────
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                   ⚠️  MIGRATION ROLLBACK                    ║"
echo "╠══════════════════════════════════════════════════════════════╣"
echo "║  Migration:  ${MIGRATION_NAME}"
echo "║  File:       $(basename "$MIGRATION_FILE")"
echo "║  Target:     ${DATABASE_URL%%\?*}"
echo "║  Dry run:    ${DRY_RUN}"
echo "║"
echo "║  This will roll back the specified migration using goose."
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

echo "Current migration status:"
goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" status 2>&1 || true
echo ""

# ── Confirmation ──────────────────────────────────────────────────────────
read -p "Roll back migration ${MIGRATION_NAME}? (y/N): " CONFIRM
if [[ ! "$CONFIRM" =~ ^[yY]$ ]]; then
  echo "Aborted."
  exit 0
fi

# ── Dry run check ─────────────────────────────────────────────────────────
if [ "$DRY_RUN" = true ]; then
  echo "[DRY RUN] Would execute:"
  echo "  goose -dir ${MIGRATIONS_DIR} postgres ${DATABASE_URL} down-to ${MIGRATION_NAME}"
  echo ""
  echo "To execute for real, remove --dry-run flag."
  exit 0
fi

# ── Pre-rollback backup ───────────────────────────────────────────────────
echo "[$(date)] Taking pre-rollback backup..."
BACKUP_SCRIPT="./scripts/backup/backup_pg.sh"
if [ -x "$BACKUP_SCRIPT" ]; then
  "$BACKUP_SCRIPT" --output-dir ./backups
else
  echo "  WARNING: Backup script not found or not executable. Skipping backup."
  echo "  Manually backup before proceeding: make backup"
  read -p "Continue without backup? (y/N): " SKIP_BACKUP
  if [[ ! "$SKIP_BACKUP" =~ ^[yY]$ ]]; then
    echo "Aborted. Run 'make backup' first."
    exit 1
  fi
fi

# ── Execute rollback ──────────────────────────────────────────────────────
echo "[$(date)] Rolling back to: ${MIGRATION_NAME}"
goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" down-to "$MIGRATION_NAME"

# ── Verify ────────────────────────────────────────────────────────────────
echo ""
echo "[$(date)] Rollback complete. Current status:"
goose -dir "$MIGRATIONS_DIR" postgres "$DATABASE_URL" status 2>&1
