-- audit_reconcile.sql — Audit Reconciliation Script
--
-- Purpose: Find discrepancies between audit.audit_log entries and actual
-- proposal/outbox state. Read-only (SELECT only), safe for any user with
-- read access. Idempotent — can be run repeatedly with no side effects.
--
-- Checks performed:
--   1. Proposals applied but no audit log entry
--   2. Proposals approved but no review record
--   3. Review records without corresponding audit log
--   4. Outbox events dispatched but no audit log entry
--   5. Proposals stuck in 'applying' state (> 1 hour)
--
-- Usage: psql -f scripts/audit_reconcile.sql

\set ON_ERROR_STOP on

-- ============================================================
-- Check 1: Proposals applied but no audit log
-- ============================================================
\echo ''
\echo '=== Check 1: Proposals applied but no audit log ==='
\echo 'Looking for: ai.action_proposal WHERE apply_status='\''applied'\'' but NO'
\echo '             audit.audit_log WHERE action IN ('\''proposal_executed'\'','\''proposal_execution_failed'\'')'
\echo '             AND resource_type='\''action_proposal'\'' AND resource_id=proposal_id'
\echo ''

SELECT
    COUNT(*) AS discrepancies_found
FROM ai.action_proposal p
WHERE p.apply_status = 'applied'
  AND NOT EXISTS (
      SELECT 1
      FROM audit.audit_log a
      WHERE a.action IN ('proposal_executed', 'proposal_execution_failed')
        AND a.resource_type = 'action_proposal'
        AND a.resource_id = p.proposal_id
  );

\echo ''
\echo 'Details:'

SELECT
    p.proposal_id,
    p.case_id,
    p.action_type,
    p.applied_at,
    p.applied_by
FROM ai.action_proposal p
WHERE p.apply_status = 'applied'
  AND NOT EXISTS (
      SELECT 1
      FROM audit.audit_log a
      WHERE a.action IN ('proposal_executed', 'proposal_execution_failed')
        AND a.resource_type = 'action_proposal'
        AND a.resource_id = p.proposal_id
  )
ORDER BY p.proposal_id;

-- ============================================================
-- Check 2: Proposals approved but no review record
-- ============================================================
\echo ''
\echo '=== Check 2: Proposals approved but no review record ==='
\echo 'Looking for: ai.action_proposal WHERE apply_status='\''approved'\'' but NO'
\echo '             ai.review_record WHERE proposal_id=proposal_id'
\echo ''

SELECT
    COUNT(*) AS discrepancies_found
FROM ai.action_proposal p
WHERE p.apply_status = 'approved'
  AND NOT EXISTS (
      SELECT 1
      FROM ai.review_record r
      WHERE r.proposal_id = p.proposal_id
  );

\echo ''
\echo 'Details:'

SELECT
    p.proposal_id,
    p.case_id,
    p.action_type,
    p.created_at
FROM ai.action_proposal p
WHERE p.apply_status = 'approved'
  AND NOT EXISTS (
      SELECT 1
      FROM ai.review_record r
      WHERE r.proposal_id = p.proposal_id
  )
ORDER BY p.proposal_id;

-- ============================================================
-- Check 3: Review records without audit log
-- ============================================================
\echo ''
\echo '=== Check 3: Review records without audit log ==='
\echo 'Looking for: ai.review_record but NO'
\echo '             audit.audit_log WHERE action='\''proposal_reviewed'\'''
\echo '             AND resource_type='\''action_proposal'\'' AND resource_id=review_record.proposal_id'
\echo ''

SELECT
    COUNT(*) AS discrepancies_found
FROM ai.review_record r
WHERE NOT EXISTS (
    SELECT 1
    FROM audit.audit_log a
    WHERE a.action = 'proposal_reviewed'
      AND a.resource_type = 'action_proposal'
      AND a.resource_id = r.proposal_id
);

\echo ''
\echo 'Details:'

SELECT
    r.record_id,
    r.proposal_id,
    r.reviewer_id,
    r.verdict,
    r.created_at
FROM ai.review_record r
WHERE NOT EXISTS (
    SELECT 1
    FROM audit.audit_log a
    WHERE a.action = 'proposal_reviewed'
      AND a.resource_type = 'action_proposal'
      AND a.resource_id = r.proposal_id
)
ORDER BY r.record_id;

-- ============================================================
-- Check 4: Outbox events dispatched but no audit log
-- ============================================================
\echo ''
\echo '=== Check 4: Outbox events dispatched but no audit log ==='
\echo 'Looking for: ops.outbox_event WHERE status='\''dispatched'\'' but NO'
\echo '             audit.audit_log WHERE action IN ('\''outbox_dispatched'\'','\''outbox_dispatch_failed'\'')'
\echo '             AND resource_type='\''outbox_event'\'' AND resource_id=event_id'
\echo ''

SELECT
    COUNT(*) AS discrepancies_found
FROM ops.outbox_event e
WHERE e.status = 'dispatched'
  AND NOT EXISTS (
      SELECT 1
      FROM audit.audit_log a
      WHERE a.action IN ('outbox_dispatched', 'outbox_dispatch_failed')
        AND a.resource_type = 'outbox_event'
        AND a.resource_id = e.event_id
  );

\echo ''
\echo 'Details:'

SELECT
    e.event_id,
    e.channel,
    e.created_at,
    e.dispatched_at
FROM ops.outbox_event e
WHERE e.status = 'dispatched'
  AND NOT EXISTS (
      SELECT 1
      FROM audit.audit_log a
      WHERE a.action IN ('outbox_dispatched', 'outbox_dispatch_failed')
        AND a.resource_type = 'outbox_event'
        AND a.resource_id = e.event_id
  )
ORDER BY e.event_id;

-- ============================================================
-- Check 5: Proposals stuck in 'applying' state (> 1 hour)
-- ============================================================
\echo ''
\echo '=== Check 5: Proposals stuck in '\''applying'\'' state (> 1 hour) ==='
\echo 'Looking for: ai.action_proposal WHERE apply_status='\''applying'\'''
\echo '             AND applied_at IS NULL AND NOW() - updated_at > INTERVAL '\''1 hour'\'''
\echo ''

SELECT
    COUNT(*) AS discrepancies_found
FROM ai.action_proposal p
WHERE p.apply_status = 'applying'
  AND p.applied_at IS NULL
  AND NOW() - p.updated_at > INTERVAL '1 hour';

\echo ''
\echo 'Details:'

SELECT
    p.proposal_id,
    p.case_id,
    p.action_type,
    p.updated_at,
    NOW() - p.updated_at AS stuck_duration
FROM ai.action_proposal p
WHERE p.apply_status = 'applying'
  AND p.applied_at IS NULL
  AND NOW() - p.updated_at > INTERVAL '1 hour'
ORDER BY p.updated_at;

\echo ''
\echo '=== Reconciliation complete ==='
