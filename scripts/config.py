"""
Centralized path configuration for the Olist data product project.
All scripts should import from this module instead of hardcoding paths.
"""

import os
import re
import logging

logger = logging.getLogger(__name__)

# Project root (parent of scripts/ directory)
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Data directories
RAW_DIR = os.path.join(PROJECT_ROOT, 'data', 'raw')
INTERIM_DIR = os.path.join(PROJECT_ROOT, 'data', 'interim')
PROCESSED_DIR = os.path.join(PROJECT_ROOT, 'data', 'processed')
SYSTEM_DIR = os.path.join(PROJECT_ROOT, 'data', 'system')
ADS_DIR = os.path.join(PROJECT_ROOT, 'data', 'ads')
AIP_DIR = os.path.join(PROJECT_ROOT, 'data', 'aip')
FEISHU_DIR = os.path.join(PROJECT_ROOT, 'data', 'feishu')

# Config directory
CONFIG_DIR = os.path.join(PROJECT_ROOT, 'config')

# Output directories
OUTPUTS_DIR = os.path.join(PROJECT_ROOT, 'outputs')
CHARTS_DIR = os.path.join(OUTPUTS_DIR, 'charts')
TABLES_DIR = os.path.join(OUTPUTS_DIR, 'tables')
VALIDATION_DIR = os.path.join(OUTPUTS_DIR, 'validation')

# Documentation directories
DOCS_DIR = os.path.join(PROJECT_ROOT, 'docs')
REPORTS_DIR = os.path.join(PROJECT_ROOT, 'reports')

# Tests directory
TESTS_DIR = os.path.join(PROJECT_ROOT, 'tests')

# Scripts directory
SCRIPTS_DIR = os.path.join(PROJECT_ROOT, 'scripts')

# Key output files
INGESTION_STATE_FILE = os.path.join(SYSTEM_DIR, 'ingestion_state.json')
RUN_MANIFEST_FILE = os.path.join(SYSTEM_DIR, 'run_manifest.csv')
VALIDATION_RESULTS_FILE = os.path.join(SYSTEM_DIR, 'validation_results.json')
DATA_QUALITY_EXCEPTIONS_FILE = os.path.join(SYSTEM_DIR, 'data_quality_exceptions.csv')

# Database
DB_PATH = os.path.join(PROJECT_ROOT, 'data', 'olist_ops.db')

# v0.3 Dimensional alerts
DIMENSIONAL_RULES_FILE = os.path.join(CONFIG_DIR, 'dimensional_alert_rules.yml')
ACTION_TEMPLATES_FILE = os.path.join(CONFIG_DIR, 'action_templates.yml')
MIGRATIONS_DIR = os.path.join(PROJECT_ROOT, 'sql', 'migrations')

# Config files
DATA_QUALITY_RULES_FILE = os.path.join(CONFIG_DIR, 'data_quality_rules.yml')
METRICS_FILE = os.path.join(CONFIG_DIR, 'metrics.yml')
ALERT_RULES_FILE = os.path.join(CONFIG_DIR, 'alert_rules.yml')
OWNER_MAPPING_FILE = os.path.join(CONFIG_DIR, 'owner_mapping.yml')
AIP_OBJECT_SCHEMA_FILE = os.path.join(CONFIG_DIR, 'aip_object_schema.yml')
FEISHU_BASE_SCHEMA_FILE = os.path.join(CONFIG_DIR, 'feishu_base_schema.yml')
STATUS_ENUMS_FILE = os.path.join(CONFIG_DIR, 'status_enums.yml')
ACTION_REGISTRY_FILE = os.path.join(CONFIG_DIR, 'action_registry.yml')
WAKE_IO_CONTRACT_FILE = os.path.join(CONFIG_DIR, 'wake_io_contract.yml')
FEISHU_FIELD_MAPPING_FILE = os.path.join(CONFIG_DIR, 'feishu_field_mapping.yml')
FEISHU_TABLE_IDS_FILE = os.path.join(CONFIG_DIR, 'feishu_table_ids.yml')
FEISHU_APP_CONFIG_FILE = os.path.join(CONFIG_DIR, 'feishu_app.yml')
FEISHU_USER_MAPPING_FILE = os.path.join(CONFIG_DIR, 'feishu_user_mapping.yml')
ADAPTER_REGISTRY_FILE = os.path.join(CONFIG_DIR, 'adapter_registry.yml')
# Governance config files (v0.5.3)
DATA_CLASSIFICATION_FILE = os.path.join(CONFIG_DIR, 'data_classification.yml')
DATA_MARKINGS_FILE = os.path.join(CONFIG_DIR, 'data_markings.yml')
DATA_LINEAGE_FILE = os.path.join(CONFIG_DIR, 'data_lineage.yml')
CHECKPOINT_RULES_FILE = os.path.join(CONFIG_DIR, 'checkpoint_rules.yml')
RETENTION_POLICIES_FILE = os.path.join(CONFIG_DIR, 'retention_policies.yml')
HEALTH_CHECKS_FILE = os.path.join(CONFIG_DIR, 'health_checks.yml')
DECISION_EVAL_RULES_FILE = os.path.join(CONFIG_DIR, 'decision_eval_rules.yml')
ACCESS_POLICY_FILE = os.path.join(CONFIG_DIR, 'access_policy.yml')
DATA_CATALOG_FILE = os.path.join(CONFIG_DIR, 'data_catalog.yml')


# AIP output files
AIP_BUSINESS_OBJECTS_FILE = os.path.join(AIP_DIR, 'aip_business_objects.json')
AIP_METRICS_FILE = os.path.join(AIP_DIR, 'aip_metrics.json')
AIP_EVENTS_FILE = os.path.join(AIP_DIR, 'aip_events.json')
AIP_ACTION_RECOMMENDATIONS_FILE = os.path.join(AIP_DIR, 'aip_action_recommendations.json')
AIP_CONTEXT_BUNDLE_FILE = os.path.join(AIP_DIR, 'aip_context_bundle.json')
AIP_CONTEXT_BUNDLE_LATEST_FILE = os.path.join(AIP_DIR, 'aip_context_bundle_latest.json')

# Key intermediate tables
ORDER_LEVEL_BASE_FILE = os.path.join(INTERIM_DIR, 'order_level_base.csv')
ITEM_LEVEL_BASE_FILE = os.path.join(INTERIM_DIR, 'item_level_base.csv')
CHANNEL_CLASSIFICATION_FILE = os.path.join(INTERIM_DIR, 'channel_classification.csv')

# ADS output files
DAILY_METRICS_FILE = os.path.join(ADS_DIR, 'daily_metrics.csv')
METRIC_ALERTS_FILE = os.path.join(ADS_DIR, 'metric_alerts.csv')
METRIC_ALERTS_FULL_FILE = os.path.join(ADS_DIR, 'metric_alerts_full.csv')


def ensure_dirs_exist():
    """Create all directories if they don't exist."""
    for d in [RAW_DIR, INTERIM_DIR, PROCESSED_DIR, SYSTEM_DIR, ADS_DIR,
              AIP_DIR, FEISHU_DIR, CONFIG_DIR, OUTPUTS_DIR, CHARTS_DIR,
              TABLES_DIR, VALIDATION_DIR, DOCS_DIR, REPORTS_DIR, TESTS_DIR,
              SCRIPTS_DIR]:
        os.makedirs(d, exist_ok=True)


_SQL_IDENTIFIER_RE = re.compile(r'^[a-zA-Z_][a-zA-Z0-9_]*$')


def validate_sql_identifier(name: str, context: str = "") -> str:
    """Validate a SQL identifier (table/column name) against a safe pattern.

    Raises ValueError if the name does not match [a-zA-Z_][a-zA-Z0-9_]*.

    Args:
        name: The identifier to validate.
        context: Optional description for error messages.

    Returns:
        The validated name (unchanged if valid).
    """
    if not _SQL_IDENTIFIER_RE.match(name):
        ctx = f" for {context}" if context else ""
        raise ValueError(
            f"Invalid SQL identifier{ctx}: {name!r}. "
            f"Must match pattern: [a-zA-Z_][a-zA-Z0-9_]*"
        )
    return name


def load_feishu_credentials() -> dict:
    """Load Feishu credentials from environment variables and YAML config.

    Priority: environment variables > config/feishu_app.yml

    Returns:
        dict with keys: app_id, app_secret, app_token, chat_id
    """
    import yaml

    cfg = {
        "app_id": os.environ.get("FEISHU_APP_ID", ""),
        "app_secret": os.environ.get("FEISHU_APP_SECRET", ""),
        "app_token": os.environ.get("FEISHU_BASE_APP_TOKEN", ""),
        "chat_id": os.environ.get("FEISHU_CHAT_ID", ""),
    }

    if os.path.exists(FEISHU_APP_CONFIG_FILE):
        try:
            with open(FEISHU_APP_CONFIG_FILE) as f:
                yml = yaml.safe_load(f) or {}
            for key in ("app_id", "app_secret", "app_token", "chat_id"):
                if not cfg[key]:
                    cfg[key] = yml.get(key, "")
        except Exception as e:
            logger.warning("Failed to load Feishu app config: %s", e)

    return cfg


if __name__ == '__main__':
    ensure_dirs_exist()
    print("Project paths:")
    print(f"  PROJECT_ROOT={PROJECT_ROOT}")
    print(f"  RAW_DIR={RAW_DIR}")
    print(f"  INTERIM_DIR={INTERIM_DIR}")
    print(f"  CONFIG_DIR={CONFIG_DIR}")
    print(f"  SYSTEM_DIR={SYSTEM_DIR}")
    print(f"  ADS_DIR={ADS_DIR}")
    print(f"  AIP_DIR={AIP_DIR}")
    print(f"  FEISHU_DIR={FEISHU_DIR}")
    print(f"  OUTPUTS_DIR={OUTPUTS_DIR}")
    print(f"  DOCS_DIR={DOCS_DIR}")
