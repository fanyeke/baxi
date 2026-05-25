import os
import re
from pathlib import Path

import yaml
from fastapi import APIRouter, Depends

from api.dependencies import get_current_user
from api.errors import APIError
from api.schemas import GovernanceConfigResponse, GovernanceStatusResponse

router = APIRouter(tags=["Governance"])

CONFIG_DIR = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "config"
)


def _load_yaml(filename):
    if not filename or not re.match(r'^[a-zA-Z0-9_\-\.]+$', filename):
        raise APIError(
            error_code="INVALID_FILENAME",
            message=f"Invalid filename: {filename!r}",
            diagnosis="Filename contains invalid characters or is empty",
            suggested_action="Use only alphanumeric characters, underscores, hyphens, and periods",
            http_status=400,
        )

    config_path = Path(CONFIG_DIR).resolve()
    target_path = (config_path / filename).resolve()

    if not str(target_path).startswith(str(config_path)):
        raise APIError(
            error_code="PATH_TRAVERSAL",
            message=f"Path traversal detected: {filename!r}",
            diagnosis="Resolved path escapes the config directory",
            suggested_action="Use a valid filename without path traversal sequences",
            http_status=400,
        )

    try:
        with open(target_path, encoding='utf-8') as f:
            return yaml.safe_load(f)
    except FileNotFoundError:
        raise APIError(
            error_code="CONFIG_NOT_FOUND",
            message=f"Config file not found: {filename}",
            diagnosis="The requested configuration file does not exist",
            suggested_action="Verify the filename is correct",
            http_status=404,
        )
    except yaml.YAMLError as e:
        raise APIError(
            error_code="YAML_PARSE_ERROR",
            message=f"YAML parse error in {filename}: {str(e)}",
            diagnosis="The configuration file contains invalid YAML syntax",
            suggested_action="Check the YAML syntax in the config file",
            http_status=500,
        )


@router.get("/governance/catalog", response_model=GovernanceConfigResponse)
def get_catalog(user=Depends(get_current_user)):
    return _load_yaml("data_catalog.yml")


@router.get("/governance/classification", response_model=GovernanceConfigResponse)
def get_classification(user=Depends(get_current_user)):
    return _load_yaml("data_classification.yml")


@router.get("/governance/markings", response_model=GovernanceConfigResponse)
def get_markings(user=Depends(get_current_user)):
    return _load_yaml("data_markings.yml")


@router.get("/governance/lineage", response_model=GovernanceConfigResponse)
def get_lineage(user=Depends(get_current_user)):
    return _load_yaml("data_lineage.yml")


@router.get("/governance/checkpoints", response_model=GovernanceConfigResponse)
def get_checkpoints(user=Depends(get_current_user)):
    return _load_yaml("checkpoint_rules.yml")


@router.get("/governance/health", response_model=GovernanceConfigResponse)
def get_health(user=Depends(get_current_user)):
    return _load_yaml("health_checks.yml")


@router.get("/governance/status", response_model=GovernanceStatusResponse)
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
        except (APIError, PermissionError):
            status[c] = "error"
    return {"governance_layer": "active", "configs": status}
