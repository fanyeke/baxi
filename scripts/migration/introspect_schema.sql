-- ============================================================================
-- PostgreSQL Schema Introspection
-- Phase 2: 7-Schema Layered Architecture Inspection
--
-- Usage:
--   psql "$DATABASE_URL" -f scripts/migration/introspect_schema.sql
--
-- This script outputs structured information about all 7 schemas:
-- raw, dwd, mart, ops, gov, ai, audit
-- ============================================================================

-- ============================================================================
-- 1. Schema Overview: List all 7 schemas with table count
-- ============================================================================
\echo '=== Schema Overview ==='
SELECT table_schema, COUNT(*) AS table_count
FROM information_schema.tables
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')
GROUP BY table_schema
ORDER BY table_schema;

-- ============================================================================
-- 2. Table Inventory: List every table with its schema and column count
-- ============================================================================
\echo ''
\echo '=== Table Inventory ==='
SELECT table_schema, table_name,
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = t.table_schema AND table_name = t.table_name) AS column_count
FROM information_schema.tables t
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')
ORDER BY table_schema, table_name;

-- ============================================================================
-- 3. Column Type Distribution: Count types per schema
-- ============================================================================
\echo ''
\echo '=== Column Types Per Schema ==='
SELECT table_schema, data_type, COUNT(*) AS type_count
FROM information_schema.columns
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')
GROUP BY table_schema, data_type
ORDER BY table_schema, data_type;

-- ============================================================================
-- 4. Index Inventory: Count indexes per schema
-- ============================================================================
\echo ''
\echo '=== Index Inventory ==='
SELECT schemaname, COUNT(*) AS index_count
FROM pg_indexes
WHERE schemaname IN ('raw','dwd','mart','ops','gov','ai','audit')
GROUP BY schemaname
ORDER BY schemaname;

-- ============================================================================
-- 5. Foreign Key Inventory: List all FK constraints
-- ============================================================================
\echo ''
\echo '=== Foreign Key Constraints ==='
SELECT conname AS constraint_name,
  conrelid::regclass AS source_table,
  confrelid::regclass AS referenced_table
FROM pg_constraint
WHERE connamespace::regnamespace::text IN ('raw','dwd','mart','ops','gov','ai','audit')
  AND contype = 'f'
ORDER BY source_table;

-- ============================================================================
-- 6. Summary: Total counts across all 7 schemas
-- ============================================================================
\echo ''
\echo '=== Summary ==='
SELECT COUNT(*) AS total_tables,
  (SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit')) AS total_columns,
  (SELECT COUNT(*) FROM pg_indexes
   WHERE schemaname IN ('raw','dwd','mart','ops','gov','ai','audit')) AS total_indexes,
  (SELECT COUNT(*) FROM pg_constraint
   WHERE connamespace::regnamespace::text IN ('raw','dwd','mart','ops','gov','ai','audit')
     AND contype = 'f') AS total_foreign_keys
FROM information_schema.tables
WHERE table_schema IN ('raw','dwd','mart','ops','gov','ai','audit');
