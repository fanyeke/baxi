import os, sys, json, sqlite3, csv, pytest
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'scripts'))
import config

pytestmark = pytest.mark.integration

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')
FEISHU_DIR = config.FEISHU_DIR


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


def test_outbox_has_business_anomaly_channel(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM event_outbox WHERE target_channel='feishu_cli'"
    ).fetchone()
    assert r[0] > 0

def test_outbox_channels_are_valid(connection):
    valid = {'feishu_cli', 'github_issue', 'manual', 'qoder_pending', 'local_cli'}
    for row in connection.execute("SELECT DISTINCT target_channel FROM event_outbox"):
        assert row[0] in valid

def test_outbox_payload_json_parseable(connection):
    for row in connection.execute("SELECT payload_json FROM event_outbox LIMIT 10"):
        assert json.loads(row[0]) is not None

def test_feishu_alert_csv_has_affected_orders(connection):
    f = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'affected_orders' in headers

def test_feishu_alert_csv_has_affected_gmv(connection):
    f = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'affected_gmv' in headers

def test_feishu_alert_csv_has_impact_score(connection):
    f = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'impact_score' in headers

def test_feishu_alert_csv_has_object_type(connection):
    f = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'object_type' in headers

def test_feishu_recommendations_csv_has_target_object_type(connection):
    f = os.path.join(FEISHU_DIR, 'strategy_recommendations_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'target_object_type' in headers

def test_feishu_tasks_csv_has_task_source(connection):
    f = os.path.join(FEISHU_DIR, 'action_tasks_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            headers = next(reader)
            assert 'task_source' in headers

def test_feishu_csv_contains_dimensional_rows(connection):
    f = os.path.join(FEISHU_DIR, 'alert_events_for_feishu.csv')
    if os.path.exists(f):
        with open(f) as fh:
            reader = csv.reader(fh)
            rows = list(reader)
            assert len(rows) > 1
