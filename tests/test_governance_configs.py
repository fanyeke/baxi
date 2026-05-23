import os
import yaml
import pytest

CONFIG_DIR = "config"


def _load_yaml(project_root, filename):
    path = os.path.join(project_root, CONFIG_DIR, filename)
    assert os.path.exists(path), f"Missing config file: {path}"
    with open(path) as f:
        return yaml.safe_load(f)



def test_catalog_has_assets(project_root):
    data = _load_yaml(project_root, "data_catalog.yml")
    assert "assets" in data, "data_catalog.yml missing top-level 'assets' key"


def test_catalog_asset_count(project_root):
    data = _load_yaml(project_root, "data_catalog.yml")
    assert len(data["assets"]) >= 30, (
        f"Expected >= 30 assets, got {len(data['assets'])}"
    )



def test_classification_has_classifications(project_root):
    data = _load_yaml(project_root, "data_classification.yml")
    assert "classifications" in data, (
        "data_classification.yml missing top-level 'classifications' key"
    )


def test_classification_count(project_root):
    data = _load_yaml(project_root, "data_classification.yml")
    assert len(data["classifications"]) >= 15, (
        f"Expected >= 15 classifications, got {len(data['classifications'])}"
    )


def test_classification_all_levels_present(project_root):
    data = _load_yaml(project_root, "data_classification.yml")
    levels = {c.get("level") for c in data["classifications"]}
    expected_levels = {"public_internal", "internal", "sensitive",
                       "derived_sensitive", "pii"}
    missing = expected_levels - levels
    assert not missing, f"Missing classification levels: {missing}"



def test_markings_has_markings(project_root):
    data = _load_yaml(project_root, "data_markings.yml")
    assert "markings" in data, (
        "data_markings.yml missing top-level 'markings' key"
    )


def test_markings_count(project_root):
    data = _load_yaml(project_root, "data_markings.yml")
    assert len(data["markings"]) >= 4, (
        f"Expected >= 4 markings, got {len(data['markings'])}"
    )


def test_markings_each_has_mandatory_control(project_root):
    data = _load_yaml(project_root, "data_markings.yml")
    for name, marking in data["markings"].items():
        assert "mandatory_control" in marking, (
            f"Marking '{name}' missing 'mandatory_control'"
        )



def test_lineage_has_nodes_and_edges(project_root):
    data = _load_yaml(project_root, "data_lineage.yml")
    assert "nodes" in data, "data_lineage.yml missing 'nodes'"
    assert "edges" in data, "data_lineage.yml missing 'edges'"


def test_lineage_node_count(project_root):
    data = _load_yaml(project_root, "data_lineage.yml")
    assert len(data["nodes"]) >= 15, (
        f"Expected >= 15 nodes, got {len(data['nodes'])}"
    )


def test_lineage_edge_count(project_root):
    data = _load_yaml(project_root, "data_lineage.yml")
    assert len(data["edges"]) >= 12, (
        f"Expected >= 12 edges, got {len(data['edges'])}"
    )



def test_checkpoint_rules_key(project_root):
    """RED phase: assert 'checkpoints' — file key is 'checkpoint_rules'."""
    data = _load_yaml(project_root, "checkpoint_rules.yml")
    assert "checkpoints" in data, (
        "checkpoint_rules.yml missing top-level 'checkpoints' key"
    )



def test_retention_policies_key(project_root):
    """RED phase: assert 'policies' — file key is 'retention_policies'."""
    data = _load_yaml(project_root, "retention_policies.yml")
    assert "policies" in data, (
        "retention_policies.yml missing top-level 'policies' key"
    )



def test_health_checks_key(project_root):
    """RED phase: assert 'health_checks' exists."""
    data = _load_yaml(project_root, "health_checks.yml")
    assert "health_checks" in data, (
        "health_checks.yml missing top-level 'health_checks' key"
    )



def test_decision_eval_key(project_root):
    """RED phase: assert 'evaluation_dimensions' — file key is 'decision_eval_rules'."""
    data = _load_yaml(project_root, "decision_eval_rules.yml")
    assert "evaluation_dimensions" in data, (
        "decision_eval_rules.yml missing top-level 'evaluation_dimensions' key"
    )



def test_access_policy_key(project_root):
    """RED phase: assert 'access_policy' exists."""
    data = _load_yaml(project_root, "access_policy.yml")
    assert "access_policy" in data, (
        "access_policy.yml missing top-level 'access_policy' key"
    )
