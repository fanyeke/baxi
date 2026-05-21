"""
Centralized path configuration for the Olist data product project.
All scripts should import from this module instead of hardcoding paths.
"""

import os

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


def ensure_dirs_exist():
    """Create all directories if they don't exist."""
    for d in [RAW_DIR, INTERIM_DIR, PROCESSED_DIR, SYSTEM_DIR, ADS_DIR,
              AIP_DIR, FEISHU_DIR, CONFIG_DIR, OUTPUTS_DIR, CHARTS_DIR,
              TABLES_DIR, VALIDATION_DIR, DOCS_DIR, REPORTS_DIR, TESTS_DIR,
              SCRIPTS_DIR]:
        os.makedirs(d, exist_ok=True)


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
