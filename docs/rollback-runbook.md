# Rollback Runbook

Step-by-step procedures for rolling back database changes in the baxi project.

## Overview

The baxi project uses **goose** for database migrations with 21 migration files (001–022). Each migration should have a corresponding `down` migration for safe rollback.

## Quick Reference

| Operation | Command |
|-----------|---------|
| Backup database | `make backup` |
| Restore from backup | `make restore FILE=./backups/baxi_YYYYMMDD_HHMMSS.sql.gz` |
| Roll back specific migration | `make rollback MIGRATION=5` |
| Check migration status | `make migrate-status` |

## Prerequisites

- PostgreSQL client tools (`psql`, `pg_dump`) installed
- `goose` installed: `go install github.com/pressly/goose/v3/cmd/goose@latest`
- `gzip` available (for compressed backups)
- Database accessible at `DATABASE_URL`

## Common Scenarios

### Scenario 1: Rollback After Failed Migration

**Situation**: A migration partially applied and broke the schema.

```bash
# 1. Check current status
make migrate-status

# 2. Rollback the failed migration
make rollback MIGRATION=5

# 3. Verify
make migrate-status
```

### Scenario 2: Restore from Pre-Phase Backup

**Situation**: Need to revert all changes since last known good state.

```bash
# 1. List available backups
ls -la ./backups/

# 2. Restore specific backup
make restore FILE=./backups/baxi_20260520_100000.sql.gz

# 3. Verify tables exist
psql -U baxi -d baxi -c "\dt"
```

### Scenario 3: Emergency Full Rollback

**Situation**: Critical production issue, need to restore to yesterday's state.

```bash
# 1. Stop API and worker services
docker compose stop api worker

# 2. Backup current state (safety net)
make backup

# 3. Restore from yesterday's backup
make restore FILE=./backups/baxi_20260519_120000.sql.gz

# 4. Restart services
docker compose up -d api worker

# 5. Verify health
curl http://localhost:8080/health
```

## Backup Management

### Creating Backups

```bash
# Standard backup (compressed, saved to ./backups/)
make backup

# Custom output directory
./scripts/backup/backup_pg.sh --output-dir /secure/backups

# Uncompressed backup (for compatibility)
./scripts/backup/backup_pg.sh --no-compress
```

### Backup Naming Convention

Backups follow the pattern: `baxi_YYYYMMDD_HHMMSS.sql[.gz]`

Example: `baxi_20260520_143022.sql.gz`

### Backup Retention

- Automatic cleanup keeps the **last 10 backups**
- Pre-restore/pre-rollback backups are kept separately (prefixed `pre_restore_` or `pre_rollback_`)
- Manual backups in custom directories are not auto-cleaned

### Listing Backups

```bash
ls -la ./backups/ | grep baxi_
```

## Restore Process

### From Compressed Backup

```bash
make restore FILE=./backups/baxi_20260520_143022.sql.gz
```

### From Uncompressed Backup

```bash
make restore FILE=./backups/baxi_20260520_143022.sql
```

### Dry Run (Preview)

```bash
./scripts/backup/restore_pg.sh ./backups/baxi_20260520_143022.sql.gz --dry-run
```

### Restore Confirmation

The restore script requires manual confirmation:

```
Type 'RESTORE' to confirm (anything else aborts): RESTORE
```

**Always verify** the backup file before confirming.

## Rollback Specific Migrations

### List Available Migrations

```bash
ls ./migrations/
```

Current migrations (21 total):

| # | File | Description |
|---|------|-------------|
| 001 | init_schemas.sql | Initial schema creation |
| 002 | raw_tables.sql | Raw data tables |
| 003 | dwd_tables.sql | Data warehouse detail tables |
| 004 | mart_tables.sql | Mart/reporting tables |
| 005 | ops_tables.sql | Operations tables |
| 006 | gov_tables.sql | Governance tables |
| 007 | ai_tables.sql | AI/LLM tables |
| 008 | audit_tables.sql | Audit trail tables |
| 009 | gov_indexes.sql | Governance indexes |
| 010 | ai_tables_enhance.sql | AI tables enhancement |
| 011 | review_action_outbox.sql | Review action outbox |
| 012 | llm_activation_eval.sql | LLM activation evaluation |
| 013 | fix_decision_case_index.sql | Decision case index fix |
| 014 | add_outbox_next_retry_at.sql | Outbox retry support |
| 016 | config_versions.sql | Config versioning |
| 017 | ontology_tables.sql | Ontology tables |
| 018 | marking_tables.sql | Marking/tagging tables |
| 019 | decision_lineage.sql | Decision lineage tracking |
| 020 | seed_ontology_data.sql | Ontology seed data |
| 021 | seed_marking_data.sql | Marking seed data |
| 022 | seed_lineage_data.sql | Lineage seed data |

### Rollback Command

```bash
# Roll back migration 007 (ai_tables.sql)
make rollback MIGRATION=7

# Preview without executing
./scripts/rollback/rollback_phase.sh 7 --dry-run
```

### What Happens During Rollback

1. **Pre-rollback backup** is automatically created
2. Goose runs the `down` migration for the specified file
3. Status is displayed after rollback

## Migration Phases

The project organizes migrations into logical phases:

### Phase 1: Core Schema (001–005)
- Initial schemas, raw tables, DWD tables, mart tables, operations tables
- **Rollback impact**: HIGH - affects all data processing

### Phase 2: Governance (006, 009)
- Governance tables and indexes
- **Rollback impact**: MEDIUM - affects governance features only

### Phase 3: AI/LLM (007, 010, 012)
- AI tables, LLM activation, decision tracking
- **Rollback impact**: LOW - AI features disabled

### Phase 4: Audit & Outbox (008, 011, 014)
- Audit trail, outbox pattern, retry logic
- **Rollback impact**: MEDIUM - audit trail lost

### Phase 5: Advanced (016–022)
- Config versions, ontology, marking, lineage
- **Rollback impact**: LOW - advanced features disabled

## Troubleshooting

### Migration Status Shows Errors

```bash
# Check goose table
psql -U baxi -d baxi -c "SELECT * FROM goose_db_version ORDER BY id DESC LIMIT 5;"

# If corrupted, manually fix:
psql -U baxi -d baxi -c "DELETE FROM goose_db_version WHERE version = <problem_version>;"
```

### Rollback Fails with Foreign Key Errors

```bash
# Temporarily disable foreign key checks
psql -U baxi -d baxi -c "SET session_replication_role = 'replica;"

# Run rollback
make rollback MIGRATION=7

# Re-enable
psql -U baxi -d baxi -c "SET session_replication_role = 'origin';"
```

### Database Connection Refused

```bash
# Check PostgreSQL container
docker compose ps postgres

# Restart if needed
docker compose restart postgres

# Wait for healthcheck
docker compose exec postgres pg_isready -U baxi
```

### Backup File Not Found

```bash
# List all backups
ls -la ./backups/

# Check for compressed variants
ls -la ./backups/*.gz

# Create a fresh backup
make backup
```

## Best Practices

1. **Always backup before rollback** - The scripts do this automatically, but verify
2. **Test rollback in dev first** - Don't rollback production without testing
3. **Document the reason** - Note why rollback was needed in commit messages
4. **Notify team** - Communicate rollback actions to the team
5. **Verify after rollback** - Run `make test` and health checks
6. **Keep backups offsite** - Copy critical backups to separate storage

## Safety Features

- **Manual confirmation required** for all destructive operations
- **Automatic pre-restore backups** before any database modification
- **Dry-run mode** available for all scripts
- **Backup retention policy** prevents unbounded disk usage
- **No auto-execution** - scripts require explicit user interaction
