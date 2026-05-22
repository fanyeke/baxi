import os, sys, sqlite3, yaml, pytest
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'scripts'))
import config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


def _columns(conn, table):
    return [r[1] for r in conn.execute(f"PRAGMA table_info({table})").fetchall()]


def test_alert_events_has_affected_orders(connection):
    assert 'affected_orders' in _columns(connection, 'alert_events')

def test_alert_events_has_affected_gmv(connection):
    assert 'affected_gmv' in _columns(connection, 'alert_events')

def test_alert_events_has_impact_score(connection):
    assert 'impact_score' in _columns(connection, 'alert_events')

def test_strategy_recommendations_has_confidence(connection):
    assert 'confidence' in _columns(connection, 'strategy_recommendations')

def test_strategy_recommendations_has_target_object_type(connection):
    assert 'target_object_type' in _columns(connection, 'strategy_recommendations')

def test_strategy_recommendations_has_target_object_id(connection):
    assert 'target_object_id' in _columns(connection, 'strategy_recommendations')

def test_dimensional_rules_config_valid():
    data = yaml.safe_load(open(config.DIMENSIONAL_RULES_FILE))
    assert len(data['rules']) == 6
    assert 'limits' in data

def test_action_templates_config_valid():
    data = yaml.safe_load(open(config.ACTION_TEMPLATES_FILE))
    assert len(data) == 6
    for k, v in data.items():
        assert 'strategy_detail_template' in v
        assert 'task_title' in v

def test_config_py_has_dimensional_constants():
    assert config.DB_PATH.endswith('olist_ops.db')
    assert 'dimensional_alert_rules.yml' in config.DIMENSIONAL_RULES_FILE
    assert 'action_templates.yml' in config.ACTION_TEMPLATES_FILE
    assert config.MIGRATIONS_DIR.endswith('migrations')
