"""Tests for services/db_service.py."""

import logging
import sqlite3
import tempfile

import pytest

from services.db_service import get_table_counts


@pytest.fixture
def temp_db_with_mixed_tables():
    """Create a temp SQLite DB with one whitelisted table and one non-whitelisted table."""
    with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
        db_path = f.name
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    # Create a whitelisted table
    conn.execute(
        "CREATE TABLE pipeline_runs (id INTEGER PRIMARY KEY, status TEXT)"
    )
    conn.execute("INSERT INTO pipeline_runs (status) VALUES ('ok')")
    # Create a non-whitelisted table (simulates temp_cache, extension tables, etc.)
    conn.execute(
        "CREATE TABLE temp_cache (id INTEGER PRIMARY KEY, data TEXT)"
    )
    conn.execute("INSERT INTO temp_cache (data) VALUES ('temp')")
    conn.commit()
    yield conn
    conn.close()
    import os
    os.unlink(db_path)


def test_get_table_counts_skips_unknown_tables(temp_db_with_mixed_tables):
    """get_table_counts should skip non-whitelist tables without raising ValueError."""
    conn = temp_db_with_mixed_tables
    counts = get_table_counts(conn)
    # Should have counts for the whitelisted table
    assert "pipeline_runs" in counts
    assert counts["pipeline_runs"] == 1
    # Should NOT have the non-whitelisted table
    assert "temp_cache" not in counts


def test_get_table_counts_logs_warning(temp_db_with_mixed_tables, caplog):
    """get_table_counts should log a warning for each skipped unknown table."""
    conn = temp_db_with_mixed_tables
    with caplog.at_level(logging.WARNING):
        get_table_counts(conn)
    # Should have logged a warning about temp_cache
    assert any("temp_cache" in record.message for record in caplog.records)
    assert any("Skipping unknown table" in record.message for record in caplog.records)
