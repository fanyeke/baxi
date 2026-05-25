"""Tests for services/_query_utils.py."""
from services._query_utils import _build_conditions


class TestBuildConditions:
    """Cover all parameter combinations for _build_conditions()."""

    def test_empty_dict(self):
        """An empty column dict produces no WHERE clause."""
        where, params = _build_conditions({})
        assert where == ""
        assert params == []

    def test_all_none(self):
        """All-None values produce no WHERE clause."""
        where, params = _build_conditions({
            "status": None,
            "severity": None,
            "object_type": None,
            "object_id": None,
        })
        assert where == ""
        assert params == []

    def test_single_column(self):
        """A single non-None column produces a WHERE with one condition."""
        where, params = _build_conditions({"status": "new"})
        assert where == "WHERE status = ?"
        assert params == ["new"]

    def test_two_columns(self):
        """Two non-None columns produce a WHERE with AND."""
        where, params = _build_conditions({"status": "new", "severity": "high"})
        assert where == "WHERE status = ? AND severity = ?"
        assert params == ["new", "high"]

    def test_all_columns(self):
        """All alert-like columns produce a full WHERE clause."""
        where, params = _build_conditions({
            "status": "new",
            "severity": "high",
            "object_type": "seller",
            "object_id": "seller-42",
        })
        assert where == "WHERE status = ? AND severity = ? AND object_type = ? AND object_id = ?"
        assert params == ["new", "high", "seller", "seller-42"]

    def test_mixed_none_and_values(self):
        """None columns are skipped; only non-None columns appear."""
        where, params = _build_conditions({
            "status": "new",
            "severity": None,
            "object_type": "seller",
            "object_id": None,
        })
        assert where == "WHERE status = ? AND object_type = ?"
        assert params == ["new", "seller"]

    def test_task_like_columns(self):
        """Task-service style columns work correctly."""
        where, params = _build_conditions({
            "status": "pending",
            "priority": "high",
            "owner_role": "ops",
        })
        assert where == "WHERE status = ? AND priority = ? AND owner_role = ?"
        assert params == ["pending", "high", "ops"]

    def test_single_int_value(self):
        """Non-string values (ints, floats) are preserved as-is."""
        where, params = _build_conditions({"limit": 100})
        assert where == "WHERE limit = ?"
        assert params == [100]

    def test_empty_string_value(self):
        """An empty string is not None, so it IS included."""
        where, params = _build_conditions({"status": ""})
        assert where == "WHERE status = ?"
        assert params == [""]

    def test_false_value(self):
        """A boolean False is not None, so it IS included."""
        where, params = _build_conditions({"active": False})
        assert where == "WHERE active = ?"
        assert params == [False]

    def test_zero_value(self):
        """Integer 0 is not None, so it IS included."""
        where, params = _build_conditions({"priority": 0})
        assert where == "WHERE priority = ?"
        assert params == [0]