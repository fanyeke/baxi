import os, sys, json, sqlite3, yaml, pytest
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'scripts'))
import config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


def test_dimensional_rules_loaded():
    data = yaml.safe_load(open(config.DIMENSIONAL_RULES_FILE))
    assert len(data['rules']) == 6

def test_6_rules_have_correct_ids():
    data = yaml.safe_load(open(config.DIMENSIONAL_RULES_FILE))
    ids = {r['rule_id'] for r in data['rules']}
    expected = {'seller_late_delivery_spike', 'seller_review_score_drop',
                'category_gmv_drop', 'category_low_review_cluster',
                'region_cancel_rate_spike', 'region_late_delivery_spike'}
    assert ids == expected

def test_all_3_dimension_types_in_rules():
    data = yaml.safe_load(open(config.DIMENSIONAL_RULES_FILE))
    dims = {r['dimension_type'] for r in data['rules']}
    assert dims == {'seller', 'category', 'region'}

def test_rule_engine_produces_dimensional_alerts(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM alert_events WHERE object_type != 'global'"
    ).fetchone()
    assert r[0] >= 30, f"Only {r[0]} dimensional alerts"

def test_alert_has_object_type_not_global(connection):
    r = connection.execute(
        "SELECT object_type FROM alert_events WHERE object_type != 'global' LIMIT 5"
    ).fetchall()
    for row in r:
        assert row[0] in ('seller', 'category', 'region')

def test_alert_has_non_null_impact_score(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM alert_events WHERE object_type != 'global' AND impact_score IS NULL"
    ).fetchone()
    assert r[0] == 0

def test_alert_has_affected_orders_and_gmv(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM alert_events WHERE object_type != 'global' AND (affected_orders IS NULL OR affected_gmv IS NULL)"
    ).fetchone()
    assert r[0] == 0

def test_impact_score_positive(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM alert_events WHERE object_type != 'global' AND impact_score <= 0"
    ).fetchone()
    assert r[0] == 0

def test_high_severity_has_higher_avg_impact(connection):
    high = connection.execute(
        "SELECT AVG(impact_score) FROM alert_events WHERE severity='high' AND object_type != 'global'"
    ).fetchone()[0]
    medium = connection.execute(
        "SELECT AVG(impact_score) FROM alert_events WHERE severity='medium' AND object_type != 'global'"
    ).fetchone()[0]
    assert high is not None and medium is not None, "Must have both high and medium alerts"
    assert high >= medium, f"high={high:.1f} should be >= medium={medium:.1f}"

def test_rule_engine_idempotent(connection):
    r = connection.execute(
        "SELECT event_id, COUNT(*) as cnt FROM alert_events WHERE object_type != 'global' GROUP BY event_id HAVING cnt > 1"
    ).fetchall()
    assert len(r) == 0, f"Found {len(r)} duplicate event_ids"

def test_top_n_caps_respected(connection):
    nonsupp = connection.execute(
        "SELECT COUNT(*) FROM alert_events WHERE object_type != 'global' AND status != 'suppressed'"
    ).fetchone()[0]
    assert nonsupp <= 50, f"Non-suppressed alerts: {nonsupp}"

def test_event_outbox_has_dimensional_entries(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM event_outbox WHERE event_type='dimensional_alert'"
    ).fetchone()
    assert r[0] > 0

def test_recommendations_have_target_object(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM strategy_recommendations WHERE target_object_type != 'global' AND target_object_id != 'global'"
    ).fetchone()
    assert r[0] >= 15

def test_tasks_have_dimensional_source(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM action_tasks WHERE task_source='dimensional_rule'"
    ).fetchone()
    assert r[0] >= 30

def test_alert_dedup_same_rule_different_runs(connection):
    alerts = connection.execute("""
        SELECT rule_id, event_date, object_type, object_id, COUNT(*) as cnt
        FROM alert_events WHERE object_type != 'global'
        GROUP BY rule_id, event_date, object_type, object_id
        HAVING cnt > 1
    """).fetchall()
    assert len(alerts) == 0


class TestTargetObjectPopulation:
    """Test that target_object is properly populated for dimensional rules."""

    def test_strategy_target_object_populated(self, connection):
        """Dimensional recommendations have non-null target_object_type/id."""
        r = connection.execute("""
            SELECT COUNT(*) FROM strategy_recommendations sr
            JOIN alert_events ae ON sr.event_id = ae.event_id
            WHERE ae.object_type != 'global'
              AND (sr.target_object_type IS NULL
                   OR sr.target_object_type = ''
                   OR sr.target_object_type = 'global'
                   OR sr.target_object_id IS NULL
                   OR sr.target_object_id = ''
                   OR sr.target_object_id = 'global')
        """).fetchone()
        assert r[0] == 0, f"{r[0]} dimensional recommendations missing target_object"

    def test_action_tasks_target_object_populated(self, connection):
        """Action tasks from dimensional_rule source have non-null target_object_type/id."""
        r = connection.execute("""
            SELECT task_id, target_object_type, target_object_id FROM action_tasks
            WHERE task_source = 'dimensional_rule'
              AND (target_object_type IS NULL
                   OR target_object_type = ''
                   OR target_object_type = 'global'
                   OR target_object_id IS NULL
                   OR target_object_id = ''
                   OR target_object_id = 'global')
        """).fetchall()
        assert len(r) == 0, (
            f"{len(r)} dimensional tasks missing target_object: "
            f"{[row[0] for row in r[:5]]}"
        )
