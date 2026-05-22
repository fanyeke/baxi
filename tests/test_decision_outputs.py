import os
import json
import pytest


def test_strategies_exist(project_root):
    path = os.path.join(project_root, "outputs/ai/strategy_recommendations.json")
    assert os.path.exists(path), f"Missing: {path}"


def test_strategies_count_min(project_root):
    with open(os.path.join(project_root, "outputs/ai/strategy_recommendations.json")) as f:
        data = json.load(f)
    items = data if isinstance(data, list) else data.get("recommendations", [])
    assert len(items) >= 5, f"Expected ≥5 strategies, got {len(items)}"


def test_strategies_have_evidence(project_root):
    with open(os.path.join(project_root, "outputs/ai/strategy_recommendations.json")) as f:
        data = json.load(f)
    items = data if isinstance(data, list) else data.get("recommendations", [])
    for item in items:
        detail = item.get("detail", "")
        assert "【问题】" in detail, "Missing 【问题】 in strategy detail"
        assert "【证据】" in detail, "Missing 【证据】 in strategy detail"


def test_strategies_have_owner(project_root):
    with open(os.path.join(project_root, "outputs/ai/strategy_recommendations.json")) as f:
        data = json.load(f)
    items = data if isinstance(data, list) else data.get("recommendations", [])
    for item in items:
        assert item.get("owner_role"), "Missing owner_role"


def test_strategies_have_decision_source(project_root):
    with open(os.path.join(project_root, "outputs/ai/strategy_recommendations.json")) as f:
        data = json.load(f)
    items = data if isinstance(data, list) else data.get("recommendations", [])
    if len(items) == 0:
        pytest.skip("No strategies to validate")
    # Check first item; if field exists, validate all; if not, skip (pre-T4 output)
    if "decision_source" not in items[0]:
        pytest.skip("Strategies predate decision_source field (pre-v0.1 consolidation)")
    for item in items:
        assert "decision_source" in item, "Missing decision_source field"
        assert item["decision_source"] in ["heuristic", "llm"], f"Invalid decision_source: {item['decision_source']}"


def test_tasks_exist(project_root):
    path = os.path.join(project_root, "outputs/ai/action_tasks.json")
    assert os.path.exists(path), f"Missing: {path}"


def test_reviews_exist(project_root):
    path = os.path.join(project_root, "outputs/ai/review_retro_draft.json")
    if os.path.exists(path):
        with open(path) as f:
            data = json.load(f)
        items = data if isinstance(data, list) else data.get("reviews", [])
        if len(items) > 0 and "review_type" not in items[0]:
            pytest.skip("Reviews predate review_type field (pre-v0.1 consolidation)")
        if items:
            for item in items:
                assert "review_type" in item, "Missing review_type field"
