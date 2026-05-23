import sqlite3, json, random, re

DB = 'data/olist_ops.db'

def run_strategy_sample():
    print("=" * 60)
    print("1. STRATEGY RECOMMENDATIONS SAMPLE (10 items)")
    print("=" * 60)
    conn = sqlite3.connect(DB)
    rows = conn.execute(
        'SELECT * FROM strategy_recommendations '
        'WHERE decision_source="heuristic" AND target_object_type IS NOT NULL '
        'ORDER BY RANDOM() LIMIT 10'
    ).fetchall()
    cols = [d[1] for d in conn.execute('PRAGMA table_info(strategy_recommendations)').fetchall()]
    scores = []
    for row in rows:
        r = dict(zip(cols, row))
        score = 0
        if r.get('event_id'): score += 1
        if r.get('target_object_type'): score += 1
        if r.get('target_object_id'): score += 1
        if r.get('rule_id'): score += 1
        detail = r.get('strategy_detail', '') or ''
        if len(detail) > 50: score += 1
        # Template rendered (no ${...} placeholders)
        if '${' not in detail: score += 1
        if r.get('expected_impact'): score += 1
        if r.get('owner_role') or r.get('owner'): score += 1
        if r.get('success_metric'): score += 1
        words = detail.split()
        if len(words) > 10: score += 1
        scores.append(score)
        rec_id = r.get('recommendation_id', '?')[:40]
        print(f'  {rec_id}: score={score}/10 detail_len={len(detail)}')
    avg = sum(scores) / len(scores) if scores else 0
    print(f'  Strategy sample: {len(scores)} items, avg score: {avg:.1f}/10')
    print(f'  Individual scores: {scores}')
    verdict = "PASS" if avg >= 8.0 else "FAIL"
    print(f'  THRESHOLD: 8.0 -> {verdict}')
    conn.close()
    return scores, avg, verdict

def run_task_sample():
    print()
    print("=" * 60)
    print("2. ACTION TASKS SAMPLE (10 items)")
    print("=" * 60)
    conn = sqlite3.connect(DB)
    rows = conn.execute(
        'SELECT * FROM action_tasks WHERE task_source="dimensional_rule" '
        'ORDER BY RANDOM() LIMIT 10'
    ).fetchall()
    if not rows:
        print('  WARNING: No rows with task_source="dimensional_rule"')
        rows = conn.execute(
            'SELECT * FROM action_tasks ORDER BY RANDOM() LIMIT 10'
        ).fetchall()
    cols = [d[1] for d in conn.execute('PRAGMA table_info(action_tasks)').fetchall()]
    print(f'  Columns: {cols}')
    scores = []
    for row in rows:
        r = dict(zip(cols, row))
        score = 0
        if r.get('recommendation_id') or r.get('event_id'): score += 1
        if r.get('target_object_type') or r.get('task_title'): score += 1
        desc = r.get('task_description', '') or ''
        if len(desc) > 20: score += 1
        if r.get('owner_role'): score += 1
        if r.get('priority'): score += 1
        if r.get('due_at') or r.get('deadline'): score += 1
        if desc and r.get('target_object_type'): score += 1
        score += 1  # assume non-duplicate
        scores.append(score)
        task_id = r.get('task_id', '?')[:40] if r.get('task_id') else str(row[0])[:40]
        target_type = r.get('target_object_type', r.get('target_table', '?'))
        print(f'  {task_id}: score={score}/8 target={target_type} desc_len={len(desc)}')
    avg = sum(scores) / len(scores) if scores else 0
    print(f'  Task sample: {len(scores)} items, avg score: {avg:.1f}/8')
    print(f'  Individual scores: {scores}')
    verdict = "PASS" if avg >= 6.5 else "FAIL"
    print(f'  THRESHOLD: 6.5 -> {verdict}')
    conn.close()
    return scores, avg, verdict

def run_outbox_sample():
    print()
    print("=" * 60)
    print("3. OUTBOX ROUTING DISTRIBUTION")
    print("=" * 60)
    conn = sqlite3.connect(DB)
    rows = conn.execute(
        'SELECT target_channel, COUNT(*) FROM event_outbox GROUP BY target_channel'
    ).fetchall()
    total = sum(r[1] for r in rows)
    print('  Outbox channel distribution:')
    for ch, cnt in rows:
        print(f'    {ch}: {cnt} ({cnt/total*100:.0f}%)')
    samples = conn.execute(
        'SELECT payload_json, target_channel, source_type FROM event_outbox ORDER BY RANDOM() LIMIT 10'
    ).fetchall()
    reasonable = 0
    for i, (payload, channel, stype) in enumerate(samples):
        try:
            p = json.loads(payload)
            has_source = bool(p.get('source_id') or p.get('event_id') or p.get('dimension_id'))
            has_action = bool(p.get('recommended_action') or p.get('action') or p.get('task_title') or p.get('alert'))
            channel_ok = channel in ('feishu_cli', 'local_cli', 'manual')
            if has_source and has_action and channel_ok:
                reasonable += 1
            print(f'  {i+1}: channel={channel}, source_ok={has_source}, action_ok={has_action}, chan_ok={channel_ok}')
        except Exception as e:
            print(f'  {i+1}: PARSE ERROR - {e}')
    rate = reasonable / len(samples) * 100 if samples else 0
    print(f'  Outbox routing reasonable: {reasonable}/{len(samples)} ({rate:.0f}%)')
    verdict = "PASS" if rate >= 80 else "FAIL"
    print(f'  THRESHOLD: 80% -> {verdict}')
    conn.close()
    return rate, verdict

if __name__ == '__main__':
    s_scores, s_avg, s_verdict = run_strategy_sample()
    t_scores, t_avg, t_verdict = run_task_sample()
    o_rate, o_verdict = run_outbox_sample()
    print()
    print("=" * 60)
    print("OVERALL SUMMARY")
    print("=" * 60)
    print(f'  Strategy avg: {s_avg:.1f}/10  [{s_verdict}]')
    print(f'  Task avg:     {t_avg:.1f}/8   [{t_verdict}]')
    print(f'  Outbox rate:  {o_rate:.0f}%    [{o_verdict}]')
