import os, sys, sqlite3, pytest
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'scripts'))
import config

DB_PATH = os.path.join(config.PROJECT_ROOT, 'data', 'olist_ops.db')


@pytest.fixture
def connection():
    conn = sqlite3.connect(DB_PATH)
    yield conn
    conn.close()


def test_dimension_metrics_not_empty(connection):
    r = connection.execute("SELECT COUNT(*) FROM metric_dimension_daily").fetchone()
    assert r[0] > 0

def test_all_three_dimension_types(connection):
    r = connection.execute(
        "SELECT COUNT(DISTINCT dimension_type) FROM metric_dimension_daily"
    ).fetchone()
    assert r[0] == 3

def test_seller_metrics_populated(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM metric_dimension_daily WHERE dimension_type='seller'"
    ).fetchone()
    assert r[0] > 0

def test_category_metrics_populated(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM metric_dimension_daily WHERE dimension_type='category'"
    ).fetchone()
    assert r[0] > 0

def test_region_metrics_populated(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM metric_dimension_daily WHERE dimension_type='region'"
    ).fetchone()
    assert r[0] > 0

def test_at_least_5_metrics_per_dimension(connection):
    for dim in ['seller', 'category', 'region']:
        r = connection.execute(
            "SELECT COUNT(DISTINCT metric_name) FROM metric_dimension_daily WHERE dimension_type=?",
            (dim,)
        ).fetchone()
        assert r[0] >= 5, f"{dim} has {r[0]} metrics"

def test_no_null_dimension_values(connection):
    r = connection.execute(
        "SELECT COUNT(*) FROM metric_dimension_daily WHERE dimension_value IS NULL OR dimension_value=''"
    ).fetchone()
    assert r[0] == 0

def test_metric_calculation_idempotent(connection):
    """Verify no duplicate rows in metric_dimension_daily."""
    r = connection.execute("""
        SELECT metric_date, dimension_type, dimension_value, metric_name, COUNT(*) as cnt
        FROM metric_dimension_daily
        GROUP BY metric_date, dimension_type, dimension_value, metric_name
        HAVING cnt > 1
    """).fetchall()
    assert len(r) == 0, f"Found {len(r)} duplicate (date,dim_type,dim_value,metric) rows"

def test_single_dimension_mode(connection):
    r = connection.execute(
        "SELECT DISTINCT dimension_type FROM metric_dimension_daily ORDER BY dimension_type"
    ).fetchall()
    assert len(r) == 3
