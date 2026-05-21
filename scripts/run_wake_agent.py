"""
Wake Agent: Reads SLIM bundle and generates 4 standard outputs.

Inputs:
  - data/aip/aip_context_bundle_latest.json
  - config/owner_mapping.yml
  - config/action_registry.yml
  - config/wake_io_contract.yml

Outputs (under outputs/wake/):
  - daily_report.md
  - feishu_message.json
  - strategy_recommendations.json
  - action_tasks.json
"""

import sys, os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import json
import yaml
import uuid
from datetime import datetime, timezone
from scripts.config import *


def load_json(path):
    with open(path, 'r', encoding='utf-8') as f:
        return json.load(f)


def load_yaml(path):
    with open(path, 'r', encoding='utf-8') as f:
        return yaml.safe_load(f)


def save_json(path, data):
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, indent=2, ensure_ascii=False)


def save_text(path, text):
    with open(path, 'w', encoding='utf-8') as f:
        f.write(text)


def gen_id():
    return uuid.uuid4().hex[:12]


def get_owner_for_metric(metric_name, owner_mapping):
    for owner in owner_mapping.get('owners', []):
        if metric_name in owner.get('metric_scope', []):
            return owner['owner_role']
    return 'business_ops'


def analyze_metric_changes(metric_summary):
    findings = []
    for m in metric_summary:
        name = m['metric_name']
        current = m['current_window_value']
        baseline = m['baseline_value']
        change = m['change_pct']
        trend = m['trend']

        if current is None:
            continue

        severity = 'info'
        note = ''
        if change is not None:
            if abs(change) > 20:
                severity = 'high'
            elif abs(change) > 10:
                severity = 'medium'
            if change > 0:
                note = f'↑ {change:+.1f}%'
            elif change < 0:
                note = f'↓ {change:+.1f}%'
        elif trend and trend != 'unknown':
            severity = 'medium'
            note = f'trend: {trend}'
        else:
            note = 'baseline unavailable (early stage)'

        findings.append({
            'metric_name': name,
            'current_value': current,
            'baseline_value': baseline,
            'change_pct': change,
            'trend': trend,
            'severity': severity,
            'note': note,
        })
    return findings


def format_metric_value(name, value):
    if value is None:
        return 'N/A'
    if name in ('gmv', 'avg_order_value', 'freight_value', 'price', 'payment_value'):
        return f'R$ {value:,.2f}'
    if name.endswith('_rate') or name.endswith('_share'):
        return f'{value * 100:.1f}%'
    if name.endswith('_count'):
        return f'{int(value):,}'
    return f'{value:,.2f}'


def main():
    bundle = load_json(AIP_CONTEXT_BUNDLE_LATEST_FILE)
    owner_mapping = load_yaml(OWNER_MAPPING_FILE)
    action_registry = load_yaml(ACTION_REGISTRY_FILE)
    wake_contract = load_yaml(WAKE_IO_CONTRACT_FILE)

    snapshot_date = bundle.get('snapshot_date', 'unknown')
    time_windows = bundle.get('time_windows', {})
    metric_summary = bundle.get('metric_summary', [])
    new_events = bundle.get('new_events', [])
    active_events = bundle.get('active_events', [])
    bundle_recommendations = bundle.get('recommendations', [])

    now_utc = datetime.now(timezone.utc)
    real_run_date = now_utc.strftime('%Y-%m-%d %H:%M:%S UTC')

    analysis = analyze_metric_changes(metric_summary)

    wake_dir = os.path.join(OUTPUTS_DIR, 'wake')
    os.makedirs(wake_dir, exist_ok=True)

    tw_current = time_windows.get('current', {})
    tw_baseline = time_windows.get('baseline', {})

    md_lines = []
    md_lines.append('# Olist 每日经营分析报告')
    md_lines.append('')
    md_lines.append(f'**快照日期**: {snapshot_date}')
    md_lines.append(f'**时间窗口**: {tw_current.get("label", "N/A")} ({tw_current.get("start", "?")} ~ {tw_current.get("end", "?")})')
    md_lines.append(f'**基线窗口**: {tw_baseline.get("label", "N/A")} ({tw_baseline.get("start", "?")} ~ {tw_baseline.get("end", "?")})')
    md_lines.append(f'**报告生成时间**: {real_run_date}')
    md_lines.append(f'**运行 ID**: {bundle.get("run_id", "N/A")}')
    md_lines.append('')

    md_lines.append('## 核心指标概览')
    md_lines.append('')
    md_lines.append('| 指标 | 当前值 | 基线值 | 变化 | 状态 |')
    md_lines.append('|------|--------|--------|------|------|')
    for f in analysis:
        name = f['metric_name']
        current_str = format_metric_value(name, f['current_value'])
        baseline_str = format_metric_value(name, f['baseline_value'])
        change_str = f['note'] if f['change_pct'] is not None else f['note']
        status = {'high': '⚠️', 'medium': '📊', 'info': 'ℹ️'}.get(f['severity'], '')
        md_lines.append(f'| {name} | {current_str} | {baseline_str} | {change_str} | {status} |')
    md_lines.append('')
    md_lines.append('## 指标观察')
    md_lines.append('')

    high_items = [f for f in analysis if f['severity'] == 'high']
    medium_items = [f for f in analysis if f['severity'] == 'medium']
    no_baseline = [f for f in analysis if f['baseline_value'] is None]

    if new_events:
        md_lines.append(f'### 新增事件 ({len(new_events)} 条)')
        md_lines.append('')
        for ev in new_events:
            md_lines.append(f'- **{ev.get("event_id", "?")}**: {ev.get("title", ev.get("description", "N/A"))}  '
                            f'[severity={ev.get("severity", "N/A")}]')
        md_lines.append('')
    else:
        md_lines.append('### 新增事件: 无')
        md_lines.append('')

    if high_items:
        md_lines.append('### ⚠️ 需关注指标')
        md_lines.append('')
        for item in high_items:
            md_lines.append(f'- **{item["metric_name"]}**: 当前值 {format_metric_value(item["metric_name"], item["current_value"])}, '
                            f'变化 {item["note"]}')
        md_lines.append('')

    if no_baseline:
        md_lines.append(f'### 基线数据不足 ({len(no_baseline)} 个指标)')
        md_lines.append('')
        md_lines.append('当前处于数据采集早期阶段，以下指标尚无有效基线值进行对比：')
        md_lines.append('')
        for item in no_baseline:
            md_lines.append(f'- `{item["metric_name"]}` = {format_metric_value(item["metric_name"], item["current_value"])}')
        md_lines.append('')
        md_lines.append('> **建议**: 在积累至少 14 天数据后建立稳定基线，届时可自动计算环比变化并触发异常告警。')
        md_lines.append('')

    if active_events:
        md_lines.append(f'### 活跃事件 ({len(active_events)} 条)')
        md_lines.append('')
        for ev in active_events:
            md_lines.append(f'- **{ev.get("event_id", "?")}**: {ev.get("title", ev.get("description", "N/A"))}  '
                            f'[status={ev.get("status", "active")}]')
        md_lines.append('')

    md_lines.append('')

    action_items = []
    for f in analysis:
        if f['severity'] in ('high', 'medium'):
            owner = get_owner_for_metric(f['metric_name'], owner_mapping)
            action_items.append({
                'metric': f['metric_name'],
                'owner': owner,
                'severity': f['severity'],
                'value': f['current_value'],
                'note': f['note'],
            })

    if action_items:
        md_lines.append('| 优先级 | 指标 | 当前值 | 负责人 | 建议行动 |')
        md_lines.append('|--------|------|--------|--------|----------|')
        for item in sorted(action_items, key=lambda x: 0 if x['severity'] == 'high' else 1):
            priority = 'P0' if item['severity'] == 'high' else 'P1'
            md_lines.append(f'| {priority} | {item["metric"]} | {format_metric_value(item["metric"], item["value"])} | {item["owner"]} | 建立基线后持续监控 |')
        md_lines.append('')
    else:
        md_lines.append('当前无需要立即行动的异常指标。')
        md_lines.append('')

    md_lines.append('## 下一步')
    md_lines.append('')
    md_lines.append('1. 持续采集数据，建立 14 天滚动基线')
    md_lines.append('2. 配置 metric alert 规则，开启异常自动检测')
    md_lines.append('3. 下一周期快照将根据基线对比输出环比变化')
    md_lines.append('4. 当前平均评分较低 (avg_review_score={:.1f})，建议重点关注卖家服务质量'.format(
        next((f['current_value'] for f in analysis if f['metric_name'] == 'avg_review_score'), 0)
    ))
    md_lines.append('')
    md_lines.append(f'---')
    md_lines.append(f'*报告由 Wake Agent 自动生成 | Run ID: {bundle.get("run_id", "N/A")}*')

    daily_report_md = '\n'.join(md_lines)
    save_text(os.path.join(wake_dir, 'daily_report.md'), daily_report_md)

    total_metrics = len(metric_summary)
    no_bl_count = len(no_baseline)
    high_count = len(high_items)
    event_count = len(new_events) + len(active_events)

    briefing_parts = []
    briefing_parts.append(f'Olist 经营日报 {snapshot_date}')
    briefing_parts.append('')
    briefing_parts.append(f'核心指标: GMV {format_metric_value("gmv", next((f["current_value"] for f in analysis if f["metric_name"] == "gmv"), 0))} | '
                          f'订单 {format_metric_value("order_count", next((f["current_value"] for f in analysis if f["metric_name"] == "order_count"), 0))} | '
                          f'客单价 {format_metric_value("avg_order_value", next((f["current_value"] for f in analysis if f["metric_name"] == "avg_order_value"), 0))}')
    briefing_parts.append(f'评价均分 {format_metric_value("avg_review_score", next((f["current_value"] for f in analysis if f["metric_name"] == "avg_review_score"), 0))} | '
                          f'取消率 {format_metric_value("cancel_rate", next((f["current_value"] for f in analysis if f["metric_name"] == "cancel_rate"), 0))}')
    briefing_parts.append('')
    briefing_parts.append(f'本期共跟踪 {total_metrics} 个指标，其中 {no_bl_count} 个指标基线数据不足（平台上线初期）。')
    if high_count > 0:
        briefing_parts.append(f'⚠️ {high_count} 个指标需要关注。')
    if event_count > 0:
        briefing_parts.append(f'📢 新增/活跃事件共 {event_count} 条。')
    else:
        briefing_parts.append('📢 本期无新增异常事件。')
    briefing_parts.append('')
    briefing_parts.append('完整报告: daily_report.md')

    feishu_message = {
        'title': f'Olist日报 {snapshot_date}',
        'content': '\n'.join(briefing_parts),
        'attachments': ['daily_report.md'],
    }
    save_json(os.path.join(wake_dir, 'feishu_message.json'), feishu_message)

    recommendations = []

    if no_baseline:
        metric_names = [f['metric_name'] for f in no_baseline]
        recommendations.append({
            'recommendation_id': gen_id(),
            'event_id': None,
            'title': '建立指标基线体系',
            'detail': f'当前 {len(no_baseline)} 个核心指标缺少基线数据，包括 {", ".join(metric_names[:5])} 等。建议运行 14 天后建立 rolling baseline，启用自动 anomaly detection。',
            'expected_impact': '启用基线后，系统可自动检测 10% 以上的环比波动并告警，提前发现业务异常。',
            'risk_level': 'low',
            'requires_approval': False,
            'owner_role': 'business_ops',
            'status': 'draft',
        })

    avg_review = next((f for f in analysis if f['metric_name'] == 'avg_review_score'), None)
    if avg_review and avg_review['current_value'] is not None and avg_review['current_value'] < 3.0:
        recommendations.append({
            'recommendation_id': gen_id(),
            'event_id': None,
            'title': '关注卖家服务质量 - 评价均分偏低',
            'detail': (f'当前平均评价分数为 {avg_review["current_value"]:.1f}（满分 5 分），该数值偏低的 '
                       f'主要原因可能是：平台刚启动、样本量小（仅 {next((f["current_value"] for f in analysis if f["metric_name"] == "order_count"), 1):.0f} 单）。'
                       f'建议重点跟踪早期卖家的履约表现和客户反馈。'),
            'expected_impact': '早期建立服务质量标杆有助于后续卖家治理，避免低质量卖家影响平台口碑。',
            'risk_level': 'medium',
            'requires_approval': True,
            'owner_role': 'seller_ops',
            'status': 'draft',
        })

    cancel = next((f for f in analysis if f['metric_name'] == 'cancel_rate'), None)
    if cancel and cancel['current_value'] is not None and cancel['current_value'] > 0:
        recommendations.append({
            'recommendation_id': gen_id(),
            'event_id': None,
            'title': '监控订单取消率',
            'detail': (f'当前订单取消率为 {cancel["current_value"] * 100:.1f}%。'
                       f'虽然样本量较小，但取消订单直接影响客户体验，建议持续跟踪。'),
            'expected_impact': '早期识别取消模式可降低后续运营阶段的取消率到 1% 以下。',
            'risk_level': 'low',
            'requires_approval': False,
            'owner_role': 'seller_ops',
            'status': 'draft',
        })

    marketing = next((f for f in analysis if f['metric_name'] == 'marketing_seller_share'), None)
    if marketing and marketing['current_value'] == 0:
        recommendations.append({
            'recommendation_id': gen_id(),
            'event_id': None,
            'title': '评估营销获客渠道效果',
            'detail': ('当前通过营销渠道入驻的卖家占比为 0%。平台启动初期尚未看到营销漏斗转化效果，'
                       '建议跟踪 marketing qualified leads 到 closed deals 的转化效率，'
                       '评估获客渠道 ROI。'),
            'expected_impact': '营销渠道优化可提升高质量卖家入驻比例，带动 GMV 增长。',
            'risk_level': 'medium',
            'requires_approval': True,
            'owner_role': 'marketing_ops',
            'status': 'draft',
        })

    valid_owners = {o['owner_role'] for o in owner_mapping.get('owners', [])}
    for rec in recommendations:
        assert rec['risk_level'] in ('low', 'medium', 'high'), f"Invalid risk_level: {rec['risk_level']}"
        assert rec['owner_role'] in valid_owners, f"Invalid owner_role: {rec['owner_role']}"
        if rec['risk_level'] == 'high':
            rec['requires_approval'] = True

    save_json(os.path.join(wake_dir, 'strategy_recommendations.json'), recommendations)

    tasks = []

    for rec in recommendations:
        priority_map = {'high': 'high', 'medium': 'medium', 'low': 'low'}
        tasks.append({
            'task_id': gen_id(),
            'title': rec['title'],
            'description': rec['detail'],
            'owner_role': rec['owner_role'],
            'source_event': None,
            'source_strategy': rec['recommendation_id'],
            'priority': priority_map.get(rec['risk_level'], 'medium'),
            'status': 'todo',
        })

    for ev in new_events + active_events:
        severity = ev.get('severity', 'medium')
        priority = 'high' if severity == 'high' else ('medium' if severity == 'medium' else 'low')
        owner = 'business_ops'
        for o in owner_mapping.get('owners', []):
            if ev.get('dimension_scope') == o.get('dimension_scope'):
                owner = o['owner_role']
                break

        tasks.append({
            'task_id': gen_id(),
            'title': ev.get('title', ev.get('description', '处理事件')),
            'description': ev.get('description', ''),
            'owner_role': owner,
            'source_event': ev.get('event_id'),
            'source_strategy': None,
            'priority': priority,
            'status': 'todo',
        })

    for task in tasks:
        assert task['owner_role'] in valid_owners, f"Invalid owner_role: {task['owner_role']}"
        assert task['priority'] in ('high', 'medium', 'low'), f"Invalid priority: {task['priority']}"
        assert task['status'] == 'todo'

    save_json(os.path.join(wake_dir, 'action_tasks.json'), tasks)

    print(f'Wake Agent completed for snapshot_date={snapshot_date}')
    print(f'  Metrics analyzed: {len(metric_summary)}')
    print(f'  New events: {len(new_events)}')
    print(f'  Active events: {len(active_events)}')
    print(f'  Recommendations generated: {len(recommendations)}')
    print(f'  Action tasks created: {len(tasks)}')
    print(f'  Output directory: {wake_dir}')


if __name__ == '__main__':
    main()
