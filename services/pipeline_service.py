"""Pipeline command preview for v0.5.1 Pipeline API.

Provides preview_pipeline_run() that returns the shell command and metadata
for each pipeline type — without executing anything.
"""

import os

from scripts import config

# Pipeline definitions: type -> (script_name, args, estimate)
_PIPELINES = {
    "daily": {
        "script": "run_daily_pipeline.py",
        "args": "",
        "duration": "~30 seconds (daily mode, single-day simulation)",
        "description": "8-step daily pipeline: ingest → quality → metrics → alerts → AIP → wake → Feishu",
    },
    "full": {
        "script": "run_full_pipeline.py",
        "args": "",
        "duration": "~5 minutes (full mode, all 634 days)",
        "description": "5-step full pipeline: metrics → alerts → AIP → AI decision → Feishu",
    },
    "db_full": {
        "script": "run_db_pipeline.py",
        "args": "--mode full --dimensional",
        "duration": "~2 minutes (DB mode, all data via SQLite)",
        "description": "5-step DB pipeline: init → ingest → metrics → rules → export",
    },
}

_REQUIRED_ENV_VARS = [
    "API_BEARER_TOKEN",
    "FEISHU_APP_ID",
    "FEISHU_APP_SECRET",
    "FEISHU_BASE_APP_TOKEN",
    "FEISHU_CHAT_ID",
]


def _check_env_warnings() -> list[str]:
    """Check for missing env vars or config files that would affect pipeline runs."""
    warnings = []

    # Optional: warn about Feishu env vars
    for var in ["FEISHU_APP_ID", "FEISHU_APP_SECRET", "FEISHU_BASE_APP_TOKEN", "FEISHU_CHAT_ID"]:
        if not os.environ.get(var, ""):
            warnings.append(f"Env var {var} not set — Feishu operations will use dry-run defaults")

    # Warn if LLM key is not set
    if not os.environ.get("LLM_API_KEY", ""):
        warnings.append("LLM_API_KEY not set — AI decision engine will use heuristic fallback")

    # Check config files exist
    if not os.path.exists(config.ALERT_RULES_FILE):
        warnings.append(f"Alert rules config missing: {config.ALERT_RULES_FILE}")
    if not os.path.exists(config.METRICS_FILE):
        warnings.append(f"Metrics config missing: {config.METRICS_FILE}")

    return warnings


def preview_pipeline_run(pipeline_type: str = "daily") -> dict:
    """Preview a pipeline run without executing anything.

    Args:
        pipeline_type: One of 'daily', 'full', 'db_full'.

    Returns:
        dict with keys: command, pipeline_type, estimated_duration,
                        required_env_vars, warnings, description (str).
    """
    if pipeline_type not in _PIPELINES:
        return {
            "command": "",
            "pipeline_type": pipeline_type,
            "estimated_duration": "",
            "required_env_vars": [],
            "warnings": [f"Unknown pipeline type: '{pipeline_type}'. Valid: {', '.join(_PIPELINES.keys())}"],
            "description": "",
        }

    pipe = _PIPELINES[pipeline_type]
    script_path = os.path.join("scripts", pipe["script"])
    command = f"python3 {script_path}"
    if pipe["args"]:
        command += f" {pipe['args']}"

    warnings = _check_env_warnings()

    return {
        "command": command,
        "pipeline_type": pipeline_type,
        "estimated_duration": pipe["duration"],
        "required_env_vars": _REQUIRED_ENV_VARS,
        "warnings": warnings,
        "description": pipe["description"],
    }


def get_available_pipelines() -> list:
    """Return list of available pipeline types with their descriptions."""
    return [
        {"type": t, "description": p["description"]}
        for t, p in _PIPELINES.items()
    ]
