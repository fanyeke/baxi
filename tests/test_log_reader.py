"""Tests for services/log_reader._tail_jsonl."""

import json
import os
import tempfile

import pytest

from services.log_reader import _tail_jsonl


def _write_jsonl(filepath: str, entries: list[dict]):
    """Write a list of dicts as JSONL to filepath."""
    with open(filepath, "w") as f:
        for entry in entries:
            f.write(json.dumps(entry) + "\n")


def test_tail_jsonl_returns_correct_entries():
    """Last N lines returned in reverse chronological order (newest first)."""
    entries = [{"id": i, "msg": f"line-{i}"} for i in range(20)]
    with tempfile.NamedTemporaryFile(mode="w", suffix=".jsonl", delete=False) as f:
        filepath = f.name
    try:
        _write_jsonl(filepath, entries)
        result = _tail_jsonl(filepath, limit=5)
        assert len(result) == 5
        expected = [
            {"id": 19, "msg": "line-19"},
            {"id": 18, "msg": "line-18"},
            {"id": 17, "msg": "line-17"},
            {"id": 16, "msg": "line-16"},
            {"id": 15, "msg": "line-15"},
        ]
        assert result == expected
    finally:
        os.remove(filepath)


def test_tail_jsonl_memory_bounded():
    """Large file (1MB+) completes quickly without OOM."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".jsonl", delete=False) as f:
        filepath = f.name
    try:
        with open(filepath, "w") as f:
            for i in range(5000):
                payload = {"id": i, "data": "x" * 200, "msg": f"line-{i}"}
                f.write(json.dumps(payload) + "\n")
        file_size = os.path.getsize(filepath)
        assert file_size > 1_000_000, f"File too small: {file_size}"
        result = _tail_jsonl(filepath, limit=100)
        assert len(result) == 100
        # Verify we got the last 100
        assert result[0]["id"] == 4999
        assert result[-1]["id"] == 4900
    finally:
        os.remove(filepath)


def test_tail_jsonl_empty_file():
    """Empty file returns []."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".jsonl", delete=False) as f:
        filepath = f.name
    try:
        result = _tail_jsonl(filepath, limit=10)
        assert result == []
    finally:
        os.remove(filepath)


def test_tail_jsonl_malformed_lines():
    """Malformed JSON lines are skipped, valid lines returned."""
    with tempfile.NamedTemporaryFile(mode="w", suffix=".jsonl", delete=False) as f:
        filepath = f.name
    try:
        with open(filepath, "w") as f:
            f.write('{"a": 1}\n')
            f.write('this is not json\n')
            f.write('{"b": 2}\n')
            f.write('{broken json\n')
            f.write('{"c": 3}\n')
        result = _tail_jsonl(filepath, limit=10)
        assert len(result) == 3
        assert result[0] == {"c": 3}
        assert result[1] == {"b": 2}
        assert result[2] == {"a": 1}
    finally:
        os.remove(filepath)
