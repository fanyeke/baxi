import os

import yaml
from fastapi import APIRouter, Depends, HTTPException

from api.dependencies import get_current_user

router = APIRouter(tags=["Governance"])

CONFIG_DIR = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "config"
)


def _load_yaml(filename):
    try:
        with open(os.path.join(CONFIG_DIR, filename), encoding='utf-8') as f:
            return yaml.safe_load(f)
    except FileNotFoundError:
        raise HTTPException(status_code=404, detail=f"Config file not found: {filename}")
    except yaml.YAMLError as e:
        raise HTTPException(status_code=500, detail=f"YAML parse error in {filename}: {str(e)}")


@router.get("/governance/catalog")
def get_catalog(user=Depends(get_current_user)):
    return _load_yaml("data_catalog.yml")


@router.get("/governance/classification")
def get_classification(user=Depends(get_current_user)):
    return _load_yaml("data_classification.yml")


@router.get("/governance/markings")
def get_markings(user=Depends(get_current_user)):
    return _load_yaml("data_markings.yml")


@router.get("/governance/lineage")
def get_lineage(user=Depends(get_current_user)):
    return _load_yaml("data_lineage.yml")


@router.get("/governance/checkpoints")
def get_checkpoints(user=Depends(get_current_user)):
    return _load_yaml("checkpoint_rules.yml")


@router.get("/governance/health")
def get_health(user=Depends(get_current_user)):
    return _load_yaml("health_checks.yml")


@router.get("/governance/status")
def get_status(user=Depends(get_current_user)):
    configs = [
        "data_catalog.yml",
        "data_classification.yml",
        "data_markings.yml",
        "data_lineage.yml",
        "checkpoint_rules.yml",
        "retention_policies.yml",
        "health_checks.yml",
        "decision_eval_rules.yml",
        "access_policy.yml",
    ]
    status = {}
    for c in configs:
        try:
            _load_yaml(c)
            status[c] = "loaded"
        except (HTTPException, PermissionError):
            status[c] = "error"
    return {"governance_layer": "active", "configs": status}
