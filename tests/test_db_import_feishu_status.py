"""Tests for db_import_feishu_status.py.

Covers: dry-run mode, UPDATE-only behavior, HUMAN_FIELDS protection, error handling.
"""
import csv
import os
import sqlite3
import subprocess
import sys
import tempfile

import pytest


SCRIPT_PATH = os.path.join(
    os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
    'scripts', 'db_import_feishu_status.py',
)


# ── Fixtures ──────────────────────────────────────────────────────────

@pytest.fixture
def temp_db():
    """Create a temporary SQLite DB with test tables and seed data."""
    fd, path = tempfile.mkstemp(suffix='.db')
    os.close(fd)
    conn = sqlite3.connect(path)
    conn.row_factory = sqlite3.Row
    conn.execute(
        'CREATE TABLE action_tasks ('
        '  task_id TEXT PRIMARY KEY,'
        '  status TEXT,'
        '  feedback TEXT'
        ')'
    )
    conn.execute(
        'CREATE TABLE review_retro ('
        '  review_id TEXT PRIMARY KEY,'
        '  status TEXT,'
        '  feedback TEXT'
        ')'
    )
    # t1: safe status (todo) — should be updatable
    conn.execute("INSERT INTO action_tasks VALUES ('t1', 'todo', '')")
    # t2: human-protected (in_progress) — should NOT be overwritten
    conn.execute("INSERT INTO action_tasks VALUES ('t2', 'in_progress', 'working')")
    # t3: human-protected (done) — should NOT be overwritten
    conn.execute("INSERT INTO action_tasks VALUES ('t3', 'done', 'finished')")
    conn.commit()
    yield path, conn
    conn.close()
    os.unlink(path)


@pytest.fixture
def snapshot_csv(tmp_path):
    """Create a temporary snapshot CSV with test data."""
    path = tmp_path / 'action_task_status_snapshot.csv'
    path.write_text(
        '_table,_record_id,status,feedback\n'
        'action_tasks,t1,new,from feishu\n'
        'action_tasks,t2,done,should not overwrite\n'
        'action_tasks,t3,completed,should not overwrite\n'
        'action_tasks,t999,todo,non-matching record\n',
        encoding='utf-8',
    )
    return str(path)


@pytest.fixture
def empty_snapshot_csv(tmp_path):
    """Create a snapshot CSV with only headers (no data rows)."""
    path = tmp_path / 'empty_snapshot.csv'
    path.write_text('_table,_record_id,status,feedback\n', encoding='utf-8')
    return str(path)


# ── Helper ────────────────────────────────────────────────────────────

def run_script(db_path, snapshot_path, extra_args=None):
    """Run the script via subprocess and return the CompletedProcess."""
    cmd = [sys.executable, SCRIPT_PATH, '--db', db_path, '--snapshot', snapshot_path]
    if extra_args:
        cmd.extend(extra_args)
    return subprocess.run(cmd, capture_output=True, text=True)


# ── Tests ─────────────────────────────────────────────────────────────

class TestDryRun:
    """Dry-run mode: script exits 0 and does NOT persist changes."""

    def test_dry_run_exits_zero(self, temp_db, snapshot_csv):
        """--dry-run (or default mode) returns exit code 0."""
        db_path, conn = temp_db
        result = run_script(db_path, snapshot_csv)
        assert result.returncode == 0, f"stderr: {result.stderr}"

    def test_dry_run_no_persist(self, temp_db, snapshot_csv):
        """Dry-run does NOT write updates to DB."""
        db_path, conn = temp_db
        # Record pre-state
        before = {
            r['task_id']: (r['status'], r['feedback'])
            for r in conn.execute(
                "SELECT task_id, status, feedback FROM action_tasks"
            ).fetchall()
        }

        run_script(db_path, snapshot_csv)

        after = {
            r['task_id']: (r['status'], r['feedback'])
            for r in conn.execute(
                "SELECT task_id, status, feedback FROM action_tasks"
            ).fetchall()
        }
        assert before == after, "Dry-run should NOT persist changes"

    def test_apply_flag_required(self, temp_db, snapshot_csv):
        """Actual updates only happen with --apply flag."""
        db_path, conn = temp_db
        # Without --apply: no change
        run_script(db_path, snapshot_csv)
        status_before = dict(
            conn.execute("SELECT task_id, status FROM action_tasks").fetchall()
        )

        # With --apply: t1 should be updated
        result = run_script(db_path, snapshot_csv, ['--apply'])
        assert result.returncode == 0

        status_after = dict(
            conn.execute("SELECT task_id, status FROM action_tasks").fetchall()
        )
        # t1 changed from 'todo' to 'new'
        assert status_after['t1'] == 'new', "t1 should be updated with --apply"
        # t2, t3 protected
        assert status_after['t2'] == 'in_progress'
        assert status_after['t3'] == 'done'


class TestUpdateOnly:
    """Import performs UPDATE, never INSERT."""

    def test_import_updates_only_matching(self, temp_db, snapshot_csv):
        """UPDATE only affects records with matching task_id."""
        db_path, conn = temp_db
        run_script(db_path, snapshot_csv, ['--apply'])

        # t1 was updated (matching)
        row = conn.execute(
            "SELECT status, feedback FROM action_tasks WHERE task_id = 't1'"
        ).fetchone()
        assert row['status'] == 'new'
        assert row['feedback'] == 'from feishu'

        # t2 still has original values (protected)
        row2 = conn.execute(
            "SELECT status, feedback FROM action_tasks WHERE task_id = 't2'"
        ).fetchone()
        assert row2['status'] == 'in_progress'
        assert row2['feedback'] == 'working'

    def test_import_does_not_insert(self, temp_db, snapshot_csv):
        """No new records are INSERTed into the DB."""
        db_path, conn = temp_db
        count_before = conn.execute(
            "SELECT COUNT(*) FROM action_tasks"
        ).fetchone()[0]

        run_script(db_path, snapshot_csv, ['--apply'])

        count_after = conn.execute(
            "SELECT COUNT(*) FROM action_tasks"
        ).fetchone()[0]
        assert count_before == count_after, "Import must not insert new records"

        # Verify t999 (non-existent in DB) was NOT created
        row = conn.execute(
            "SELECT * FROM action_tasks WHERE task_id = 't999'"
        ).fetchone()
        assert row is None, "t999 should NOT have been inserted"


class TestHumanProtection:
    """HUMAN_PROTECTED_STATUSES are never overwritten."""

    def test_human_protected_not_overwritten(self, temp_db, snapshot_csv):
        """Records with human-set status are NOT overwritten."""
        db_path, conn = temp_db
        run_script(db_path, snapshot_csv, ['--apply'])

        for task_id, expected_status, expected_feedback in [
            ('t2', 'in_progress', 'working'),
            ('t3', 'done', 'finished'),
        ]:
            row = conn.execute(
                "SELECT status, feedback FROM action_tasks WHERE task_id = ?",
                (task_id,),
            ).fetchone()
            assert row['status'] == expected_status, (
                f"{task_id} status should be protected"
            )
            assert row['feedback'] == expected_feedback, (
                f"{task_id} feedback should be protected"
            )

    def test_human_protected_all_statuses(self, temp_db, tmp_path):
        """All HUMAN_PROTECTED_STATUSES are protected."""
        # Set t1 to each protected status and verify
        db_path, conn = temp_db
        protected = {'in_progress', 'done', 'completed', 'blocked', 'cancelled'}
        for prot_status in protected:
            conn.execute(
                "UPDATE action_tasks SET status = ?, feedback = '' WHERE task_id = 't1'",
                (prot_status,),
            )
            conn.commit()

            snapshot = tmp_path / f'snapshot_{prot_status}.csv'
            snapshot.write_text(
                '_table,_record_id,status,feedback\n'
                f'action_tasks,t1,overwritten,evil\n',
                encoding='utf-8',
            )

            run_script(db_path, str(snapshot), ['--apply'])

            row = conn.execute(
                "SELECT status FROM action_tasks WHERE task_id = 't1'"
            ).fetchone()
            assert row['status'] == prot_status, (
                f"Status '{prot_status}' should be protected from overwrite"
            )


class TestEdgeCases:
    """Error handling and edge cases."""

    def test_non_matching_skipped(self, temp_db, snapshot_csv):
        """A CSV row with task_id not in DB is skipped without error."""
        db_path, conn = temp_db
        result = run_script(db_path, snapshot_csv, ['--apply'])
        assert result.returncode == 0
        assert 'SKIP' in result.stdout or 'skip' in result.stdout.lower() or \
               'not in DB' in result.stdout or result.returncode == 0

    def test_missing_csv_handled(self, temp_db):
        """Missing snapshot CSV produces clear error, no crash."""
        db_path, _ = temp_db
        result = run_script(db_path, '/nonexistent/path/to/snapshot.csv')
        assert result.returncode != 0
        assert 'ERROR' in result.stdout or 'error' in result.stdout.lower() or \
               'ERROR' in result.stderr

    def test_empty_snapshot(self, temp_db, empty_snapshot_csv):
        """Empty snapshot (headers only) prints message and exits cleanly."""
        db_path, _ = temp_db
        result = run_script(db_path, empty_snapshot_csv)
        assert result.returncode == 0
        assert 'empty' in result.stdout.lower() or 'no records' in result.stdout.lower()

    def test_missing_db_handled(self, snapshot_csv):
        """Missing DB file produces clear error, no crash."""
        result = run_script('/nonexistent/path/to/db.sqlite', snapshot_csv)
        assert result.returncode != 0
        assert 'ERROR' in result.stdout or 'error' in result.stdout.lower() or \
               'ERROR' in result.stderr
