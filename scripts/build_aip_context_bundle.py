import sys, os, json, csv
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import *
from datetime import datetime, timezone, timedelta
import yaml


def load_json_if_exists(path):
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return None


def load_yaml_if_exists(path):
    if os.path.exists(path):
        with open(path) as f:
            return yaml.safe_load(f)
    return None


def load_csv_if_exists(path):
    if os.path.exists(path):
        with open(path, newline='') as f:
            reader = csv.DictReader(f)
            return list(reader)
    return []


def resolve_snapshot_date(state):
    date_str = state.get('last_completed_simulated_date') or state.get('current_simulated_date')
    if date_str:
        try:
            return datetime.strptime(date_str, '%Y-%m-%d').strftime('%Y-%m-%d')
        except ValueError:
            pass
    return 'unknown'


def get_run_id_for_date(target_date):
    if not os.path.exists(RUN_MANIFEST_FILE):
        return None
    rows = load_csv_if_exists(RUN_MANIFEST_FILE)
    for row in reversed(rows):
        if row.get('simulated_date') == target_date:
            return row.get('run_id')
    return None


def compute_time_windows(snapshot_date):
    try:
        end = datetime.strptime(snapshot_date, '%Y-%m-%d')
    except ValueError:
        return None, None

    current_end = end
    current_start = end - timedelta(days=6)
    baseline_end = current_start - timedelta(days=1)
    baseline_start = baseline_end - timedelta(days=13)

    current = {
        'label': '最近7天',
        'start': current_start.strftime('%Y-%m-%d'),
        'end': current_end.strftime('%Y-%m-%d'),
    }
    baseline = {
        'label': '前14天',
        'start': baseline_start.strftime('%Y-%m-%d'),
        'end': baseline_end.strftime('%Y-%m-%d'),
    }
    return current, baseline


def safe_float(val):
    if val is None or val == '':
        return None
    try:
        return float(val)
    except (ValueError, TypeError):
        return None


def compute_metric_summary(daily_rows, current_window, baseline_window):
    if not daily_rows or not current_window or not baseline_window:
        return []

    cs = current_window['start']
    ce = current_window['end']
    bs = baseline_window['start']
    be = baseline_window['end']

    from collections import defaultdict
    current_vals = defaultdict(list)
    baseline_vals = defaultdict(list)

    for row in daily_rows:
        metric_date = row.get('simulated_date')
        if not metric_date:
            continue
        if bs <= metric_date <= be:
            for col in row:
                if col in ('simulated_date', 'real_run_date'):
                    continue
                v = safe_float(row[col])
                if v is not None:
                    baseline_vals[col].append(v)
        elif cs <= metric_date <= ce:
            for col in row:
                if col in ('simulated_date', 'real_run_date'):
                    continue
                v = safe_float(row[col])
                if v is not None:
                    current_vals[col].append(v)

    all_metrics = set(list(current_vals.keys()) + list(baseline_vals.keys()))
    summary = []
    for metric_name in sorted(all_metrics):
        cvals = current_vals.get(metric_name, [])
        bvals = baseline_vals.get(metric_name, [])

        current_avg = sum(cvals) / len(cvals) if cvals else None
        baseline_avg = sum(bvals) / len(bvals) if bvals else None

        if current_avg is not None and baseline_avg is not None and baseline_avg != 0:
            change_pct = round((current_avg - baseline_avg) / abs(baseline_avg), 4)
        elif current_avg is not None:
            change_pct = None
        else:
            change_pct = None

        if change_pct is not None:
            if change_pct > 0.01:
                trend = 'up'
            elif change_pct < -0.01:
                trend = 'down'
            else:
                trend = 'stable'
        else:
            trend = 'unknown'

        entry = {
            'metric_name': metric_name,
            'current_window_value': round(current_avg, 4) if current_avg is not None else None,
            'baseline_value': round(baseline_avg, 4) if baseline_avg is not None else None,
            'change_pct': change_pct,
            'trend': trend,
        }
        summary.append(entry)

    return summary


def append_to_manifest(run_id, simulated_date, status='success'):
    file_exists = os.path.exists(RUN_MANIFEST_FILE)
    now = datetime.now(timezone.utc).isoformat()
    with open(RUN_MANIFEST_FILE, 'a', newline='') as f:
        writer = csv.writer(f)
        if not file_exists:
            writer.writerow([
                'run_id', 'real_run_date', 'simulated_date',
                'pipeline_stage', 'input_row_count', 'output_row_count',
                'status', 'error_message', 'started_at', 'finished_at',
                'bundle_path', 'report_path'
            ])
        writer.writerow([
            run_id, now, simulated_date,
            'aip_bundle', 0, 1,
            status, '', now, now,
            AIP_CONTEXT_BUNDLE_LATEST_FILE, ''
        ])


def main():
    state = load_json_if_exists(INGESTION_STATE_FILE) or {}
    daily_rows = load_csv_if_exists(DAILY_METRICS_FILE)

    snapshot_date = resolve_snapshot_date(state)

    from hashlib import md5
    run_id = md5(f"aip_bundle_{snapshot_date}_{datetime.now().isoformat()}".encode()).hexdigest()

    current_window, baseline_window = compute_time_windows(snapshot_date)

    metric_summary = compute_metric_summary(daily_rows, current_window, baseline_window)

    events_data = load_json_if_exists(AIP_EVENTS_FILE)
    all_events = events_data.get('events', []) if events_data else []
    new_events = [e for e in all_events if e.get('properties', {}).get('status') == 'new']
    active_events = [e for e in all_events if e.get('properties', {}).get('status') in ('active', 'acknowledged')]

    rec_data = load_json_if_exists(AIP_ACTION_RECOMMENDATIONS_FILE)
    recommendations = rec_data.get('recommendations', []) if rec_data else []

    allowed_actions = load_yaml_if_exists(ACTION_REGISTRY_FILE) or {}
    owner_mapping = load_yaml_if_exists(OWNER_MAPPING_FILE) or {}

    bundle = {
        'snapshot_date': snapshot_date,
        'run_id': run_id,
        'ingestion_state': {
            'next_simulated_date': state.get('next_simulated_date', 'unknown'),
            'last_completed_simulated_date': snapshot_date,
        },
        'time_windows': {
            'current': current_window,
            'baseline': baseline_window,
        },
        'metric_summary': metric_summary,
        'new_events': new_events,
        'active_events': active_events,
        'recommendations': recommendations,
        'allowed_actions': allowed_actions,
        'owner_mapping': owner_mapping,
    }

    os.makedirs(AIP_DIR, exist_ok=True)

    with open(AIP_CONTEXT_BUNDLE_LATEST_FILE, 'w') as f:
        json.dump(bundle, f, indent=2, ensure_ascii=False)

    counts = {
        'metrics': len(metric_summary),
        'new_events': len(new_events),
        'active_events': len(active_events),
        'recommendations': len(recommendations),
        'allowed_actions': len(allowed_actions.get('actions', {})),
        'owners': len(owner_mapping.get('owners', [])),
    }
    print(f"[bundle] Created SLIM {AIP_CONTEXT_BUNDLE_LATEST_FILE}")
    print(f"  snapshot_date: {snapshot_date}")
    print(f"  run_id: {run_id}")
    if current_window and baseline_window:
        print(f"  time_windows: {current_window['start']} -> {current_window['end']} (current), {baseline_window['start']} -> {baseline_window['end']} (baseline)")
    print(f"  counts: {counts}")

    append_to_manifest(run_id, snapshot_date)
    print(f"[manifest] Appended run_id={run_id} to {RUN_MANIFEST_FILE}")

    metrics_data = load_json_if_exists(AIP_METRICS_FILE)
    if metrics_data is not None:
        print(f"[archive] Full historical metrics preserved at {AIP_METRICS_FILE} ({len(metrics_data.get('metrics', []))} records)")
    else:
        print(f"[warn] {AIP_METRICS_FILE} not found, no historical metrics to archive")


if __name__ == '__main__':
    main()
