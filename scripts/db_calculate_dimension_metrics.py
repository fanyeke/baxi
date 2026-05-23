#!/usr/bin/env python3
"""db_calculate_dimension_metrics.py - Compute dimension-level daily metrics."""
import os, sys, datetime, sqlite3, argparse
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import config

METRICS = ['gmv', 'order_count', 'customer_count', 'avg_order_value',
           'avg_review_score', 'late_delivery_rate', 'cancel_rate']


def get_db(db_path):
    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    return conn


QUERIES = {
    'seller': """
        SELECT o.purchase_date, i.seller_id,
               SUM(i.price) as gmv,
               COUNT(DISTINCT o.order_id) as order_count,
               COUNT(DISTINCT o.customer_unique_id) as customer_count,
               ROUND(SUM(i.price) * 1.0 / NULLIF(COUNT(DISTINCT o.order_id), 0), 2) as avg_order_value,
               ROUND(AVG(CASE WHEN o.review_score IS NOT NULL THEN o.review_score END), 4) as avg_review_score,
               ROUND(CAST(SUM(CASE WHEN o.order_status='delivered' AND o.is_late=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(CASE WHEN o.order_status='delivered' THEN 1 END), 0), 4) as late_delivery_rate,
               ROUND(CAST(SUM(CASE WHEN o.is_cancelled=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(o.order_id), 0), 4) as cancel_rate
        FROM dwd_item_level i
        JOIN dwd_order_level o ON i.order_id = o.order_id
        WHERE o.purchase_date IS NOT NULL
          AND i.seller_id IS NOT NULL
          AND o.purchase_date BETWEEN ? AND ?
        GROUP BY o.purchase_date, i.seller_id
    """,
    'category': """
        SELECT o.purchase_date, i.product_category_name_english,
               SUM(i.price) as gmv,
               COUNT(DISTINCT o.order_id) as order_count,
               COUNT(DISTINCT o.customer_unique_id) as customer_count,
               ROUND(SUM(i.price) * 1.0 / NULLIF(COUNT(DISTINCT o.order_id), 0), 2) as avg_order_value,
               ROUND(AVG(CASE WHEN o.review_score IS NOT NULL THEN o.review_score END), 4) as avg_review_score,
               ROUND(CAST(SUM(CASE WHEN o.order_status='delivered' AND o.is_late=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(CASE WHEN o.order_status='delivered' THEN 1 END), 0), 4) as late_delivery_rate,
               ROUND(CAST(SUM(CASE WHEN o.is_cancelled=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(o.order_id), 0), 4) as cancel_rate
        FROM dwd_item_level i
        JOIN dwd_order_level o ON i.order_id = o.order_id
        WHERE o.purchase_date IS NOT NULL
          AND i.product_category_name_english IS NOT NULL
          AND o.purchase_date BETWEEN ? AND ?
        GROUP BY o.purchase_date, i.product_category_name_english
    """,
    'region': """
        SELECT o.purchase_date, o.customer_state,
               SUM(i.price) as gmv,
               COUNT(DISTINCT o.order_id) as order_count,
               COUNT(DISTINCT o.customer_unique_id) as customer_count,
               ROUND(SUM(i.price) * 1.0 / NULLIF(COUNT(DISTINCT o.order_id), 0), 2) as avg_order_value,
               ROUND(AVG(CASE WHEN o.review_score IS NOT NULL THEN o.review_score END), 4) as avg_review_score,
               ROUND(CAST(SUM(CASE WHEN o.order_status='delivered' AND o.is_late=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(CASE WHEN o.order_status='delivered' THEN 1 END), 0), 4) as late_delivery_rate,
               ROUND(CAST(SUM(CASE WHEN o.is_cancelled=1 THEN 1 ELSE 0 END) AS REAL)
                     / NULLIF(COUNT(o.order_id), 0), 4) as cancel_rate
        FROM dwd_item_level i
        JOIN dwd_order_level o ON i.order_id = o.order_id
        WHERE o.purchase_date IS NOT NULL
          AND o.customer_state IS NOT NULL
          AND o.purchase_date BETWEEN ? AND ?
        GROUP BY o.purchase_date, o.customer_state
    """,
}


DIMENSION_SQL = {
    'seller': 'i.seller_id',
    'category': 'i.product_category_name_english',
    'region': 'o.customer_state',
}

DIMENSION_FILTER = {
    'seller': 'i.seller_id IS NOT NULL',
    'category': 'i.product_category_name_english IS NOT NULL',
    'region': 'o.customer_state IS NOT NULL',
}


def calculate_metrics(conn, mode, date_start=None, date_end=None, dimension_filter=None):
    now = datetime.datetime.now().isoformat()
    dimensions = dimension_filter if dimension_filter else ['seller', 'category', 'region']

    if mode == 'full':
        cur = conn.execute("SELECT MIN(purchase_date), MAX(purchase_date) FROM dwd_order_level WHERE purchase_date IS NOT NULL")
        date_min, date_max = cur.fetchone()
        for dim in dimensions:
            conn.execute("DELETE FROM metric_dimension_daily WHERE dimension_type = ?", (dim,))
    elif mode == 'range':
        date_min, date_max = date_start, date_end
        for dim in dimensions:
            conn.execute("DELETE FROM metric_dimension_daily WHERE dimension_type = ? AND metric_date BETWEEN ? AND ?",
                         (dim, date_start, date_end))
    else:
        raise ValueError(f"Unknown mode: {mode}")

    print(f"[dim_metrics] Computing metrics for {date_min} to {date_max}, dimensions: {dimensions}")
    total = 0

    for dim in dimensions:
        query = QUERIES[dim]
        cur = conn.execute(query, (date_min, date_max))
        rows = cur.fetchall()

        for row in rows:
            metric_date, dim_value = row[0], str(row[1]) or 'unknown'
            values = {METRICS[j]: row[2 + j] for j in range(len(METRICS))}

            for metric_name in METRICS:
                val = values.get(metric_name)
                sample = values.get('order_count', 0) or 0

                conn.execute("""
                    INSERT OR REPLACE INTO metric_dimension_daily
                    (metric_date, dimension_type, dimension_value, metric_name, metric_value, sample_size, created_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?)
                """, (
                    metric_date, dim, dim_value, metric_name,
                    round(val, 4) if val is not None else 0,
                    int(sample),
                    now
                ))
                total += 1

        print(f"  [{dim}] {len(rows)} (date,dim_value) groups -> {sum(1 for _ in rows) * len(METRICS)} metric rows")

    print(f"[dim_metrics] Computed {total} total rows")
    return total


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', required=True, choices=['full', 'range'])
    parser.add_argument('--start', default=None)
    parser.add_argument('--end', default=None)
    parser.add_argument('--dimension', default=None,
                        choices=['seller', 'category', 'region'])
    parser.add_argument('--db', default=None)
    args = parser.parse_args()

    if args.mode == 'range' and (not args.start or not args.end):
        print("[dim_metrics] ERROR: range mode requires --start and --end")
        sys.exit(1)

    db_path = args.db if args.db else config.DB_PATH
    dimensions = [args.dimension] if args.dimension else None

    conn = get_db(db_path)
    try:
        count = calculate_metrics(conn, args.mode, args.start, args.end, dimensions)
        conn.commit()
        print(f"[dim_metrics] Done. {count} rows in metric_dimension_daily")
    except Exception as e:
        conn.rollback()
        print(f"[dim_metrics] FAILED: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        conn.close()


if __name__ == '__main__':
    main()
