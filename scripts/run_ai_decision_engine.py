"""
AI Decision Engine: Generates structured strategy recommendations from full-mode data.
Uses LLM API if available; falls back to rule-based heuristics on failure.
"""
import sys, os, json, csv, argparse
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import *
import pandas as pd
from scripts.ai_schemas import StrategyRecommendation, ActionTask, validate_strategy_detail
from datetime import datetime, timezone, timedelta
import yaml


def load_llm_config():
    path = os.path.join(CONFIG_DIR, 'llm_config.yml')
    if os.path.exists(path):
        with open(path) as f:
            return yaml.safe_load(f)
    return {'fallback': 'rule_based'}


def load_config(name):
    path = os.path.join(CONFIG_DIR, name)
    if os.path.exists(path):
        yield path
        with open(path) as f:
            return yaml.safe_load(f)
    return {}


def build_fallback_strategies(alerts_df, metrics_df=None):
    strategies = []
    tasks = []

    alert_templates = {
        'gmv_drop': {
            'title': 'GMV下降专项排查',
            'detail': '【问题】GMV出现显著下降\n【证据】近7日均值较前14日均值下降超过15%\n【判断】可能受季节性因素、竞品促销或商品下架影响\n【建议动作】排查品类GMV趋势，确认是否存在系统性下降\n【预期收益】识别并应对GMV下降因素\n【风险】当前缺少竞品和外部数据，可能遗漏外部原因\n【验收指标】7日内GMV恢复至基线85%以上',
        },
        'gmv_spike': {
            'title': 'GMV飙升溯源分析',
            'detail': '【问题】GMV出现异常上升\n【证据】近7日均值较前14日均值上升超过20%\n【判断】可能受促销活动、季节性高峰或数据异常驱动\n【建议动作】核实促销日历，确认GMV增长是否可持续\n【预期收益】识别可持续增长来源\n【风险】促销脉冲可能不可持续\n【验收指标】确认增长来源，区分一次性 vs 持续性',
        },
        'order_drop': {
            'title': '订单量下降调查',
            'detail': '【问题】订单数量出现显著减少\n【证据】订单量7日均值较前14日均值下降超过20%\n【判断】可能受用户流失、转化率下降或流量减少影响\n【建议动作】分析转化漏斗，排查流量来源和转化环节\n【预期收益】恢复订单量至正常水平\n【风险】可能需长期运营调整\n【验收指标】7日内订单量恢复至基线80%以上',
        },
        'cancel_rate_spike': {
            'title': '取消率升高专项处理',
            'detail': '【问题】订单取消率显著升高\n【证据】取消率较基线变化超过50%且当前值大于5%\n【判断】可能存在库存不足、卖家发货超时或物流问题\n【建议动作】排查取消原因分类，优先处理高频取消的卖家和品类\n【预期收益】降低取消率，提升实收GMV\n【风险】数据缺少取消原因细粒度信息\n【验收指标】7日内取消率回落至5%以下',
        },
        'late_delivery_spike': {
            'title': '延迟配送专项治理',
            'detail': '【问题】配送延迟率异常升高\n【证据】延迟配送率超过25%\n【判断】可能存在物流商运力不足、仓储积压或地区性问题\n【建议动作】分析地区配送时效，排查物流商表现\n【预期收益】提升履约时效，改善客户体验\n【风险】物流基础设施调整周期较长\n【验收指标】7日内延迟率回落至20%以下',
        },
        'review_score_drop': {
            'title': '客户评分下降排查',
            'detail': '【问题】客户平均评分显著下降\n【证据】评分较基线下降超过0.3\n【判断】可能存在商品质量、配送体验或客服问题\n【建议动作】分析差评内容关键词，定位问题品类和卖家\n【预期收益】提升客户满意度\n【风险】缺乏评论文本数据，只能从评分维度分析\n【验收指标】7日内评分恢复至基线水平',
        },
        'low_review_cluster': {
            'title': '差评聚类分析',
            'detail': '【问题】差评率集中偏高\n【证据】差评率超过15%且评价样本充足\n【判断】存在系统性服务或商品质量问题\n【建议动作】对高差评率品类进行专项治理\n【预期收益】降低差评率，提升整体评分\n【风险】品类运营策略需时间见效\n【验收指标】差评率7日内回落至10%以下',
        },
        'seller_risk': {
            'title': '活跃卖家流失预警',
            'detail': '【问题】活跃卖家数量显著减少\n【证据】卖家数7日均值较前14日均值下降超过30%\n【判断】可能存在卖家流失或入驻停滞\n【建议动作】排查卖家生命周期状态，确认流失原因\n【预期收益】稳定卖家供给\n【风险】卖家流失可能涉及平台政策或市场竞争\n【验收指标】卖家数量7日内回升',
        },
    }

    decision_type_map = {
        'high': 'intervention',
        'medium': 'investigate',
        'low': 'monitor_only',
    }

    for _, alert in alerts_df.iterrows():
        rule_id = alert.get('rule_id', 'unknown')
        template = alert_templates.get(rule_id, {
            'title': f'{rule_id}异常处理',
            'detail': '【问题】异常\n【证据】检测到异常\n【判断】需进一步分析\n【建议动作】人工排查\n【预期收益】恢复正常\n【风险】暂无\n【验收指标】指标恢复正常',
        })
        owner = alert.get('owner_role', 'business_ops')
        sev = alert.get('severity', 'medium')
        impact = int(alert.get('impact_score', 1))

        strategies.append({
            'recommendation_id': f"rec_{len(strategies)+1:03d}",
            'event_id': alert.get('alert_id', ''),
            'title': template['title'],
            'detail': template['detail'],
            'target_object': alert.get('object_type', alert.get('dimension', '')),
            'expected_impact': template['detail'].split('【预期收益】')[1].split('【风险】')[0].strip() if '【预期收益】' in template['detail'] else '',
            'risk_level': sev,
            'requires_approval': sev == 'high',
            'owner_role': owner,
            'decision_type': decision_type_map.get(sev, 'investigate'),
            'confidence': 'medium' if impact >= 2 else 'low',
            'success_metric': template['detail'].split('【验收指标】')[1].strip() if '【验收指标】' in template['detail'] else '指标恢复正常',
            'impact_score': impact,
            'created_at': datetime.now(timezone.utc).isoformat(),
            'decision_source': 'heuristic',
            'model_name': None,
            'is_simulated': True,
        })

    if strategies:
        from scripts.config import OUTPUTS_DIR
        os.makedirs(os.path.join(OUTPUTS_DIR, 'ai'), exist_ok=True)
        with open(os.path.join(OUTPUTS_DIR, 'ai', 'strategy_recommendations.json'), 'w') as f:
            json.dump(strategies, f, indent=2, ensure_ascii=False)
        print(f"  [LLM fallback] Generated {len(strategies)} heuristic strategies")

    for s in strategies[:]:
        if s['decision_type'] in ('investigate', 'optimize', 'intervention'):
            tasks.append({
                'task_id': f"task_{len(tasks)+1:03d}",
                'title': s['title'],
                'description': s['detail'][:200] + ('...' if len(s['detail']) > 200 else ''),
                'owner_role': s['owner_role'],
                'source_event': s.get('event_id', ''),
                'source_strategy': s['recommendation_id'],
                'priority': 'high' if s['risk_level'] == 'high' else 'medium',
                'deadline': (datetime.now(timezone.utc) + timedelta(days=7)).strftime('%Y-%m-%d'),
                'status': 'todo',
                'task_source': 'heuristic',
                'source_rule': s.get('event_id'),
                'requires_human_confirmation': True,
            })

    with open(os.path.join(OUTPUTS_DIR, 'ai', 'action_tasks.json'), 'w') as f:
        json.dump(tasks, f, indent=2, ensure_ascii=False)

    return strategies, tasks


def main():
    from scripts.config import OUTPUTS_DIR
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument('--mode', choices=['daily', 'full'], default='daily')
    parser.add_argument('--top-alerts', type=int, default=20)
    parser.add_argument('--dry-run', action='store_true')
    args = parser.parse_args()

    config = load_llm_config()
    api_key = os.environ.get('LLM_API_KEY', '')

    alerts_file = METRIC_ALERTS_FULL_FILE if args.mode == 'full' else METRIC_ALERTS_FILE
    if not os.path.exists(alerts_file):
        print(f"[ai] No alerts file found at {alerts_file}")
        return

    alerts_df = pd.read_csv(alerts_file)
    if alerts_df.empty:
        print(f"[ai] No alerts in {alerts_file}")
        return

    alerts_df = alerts_df.head(args.top_alerts)

    print(f"[ai] Decision engine (mode={args.mode}, fallback={config.get('fallback', 'rule_based')})")
    print(f"  Alerts loaded: {len(alerts_df)}")

    if args.dry_run:
        print("\n[DRY RUN] System prompt:")
        print('You are an e-commerce operations AI. Analyze the provided business metrics and alerts, generate structured strategy recommendations with format: 【问题】【证据】【判断】【建议动作】【预期收益】【风险】【验收指标】')
        print("\n[DRY RUN] User context (excerpt):")
        for _, a in alerts_df.head(3).iterrows():
            print(f"  Rule: {a.get('rule_id')}, Severity: {a.get('severity')}, Description: {a.get('description', '')[:80]}")
        return

    if api_key and api_key != 'sk-your-key-here':
        try:
            from openai import OpenAI
            client = OpenAI(api_key=api_key, base_url=config.get('api_base', 'https://api.openai.com/v1'))
            print(f"  [LLM] Calling {config.get('model', 'gpt-4o')}...")
            context = []
            for _, a in alerts_df.iterrows():
                context.append(f"- {a.get('rule_id')}: {a.get('description', '')}")
            prompt = f"Analyze these alerts and generate strategies with this format: 【问题】【证据】【判断】【建议动作】【预期收益】【风险】【验收指标】\n\nAlerts:\n" + "\n".join(context)
            resp = client.chat.completions.create(
                model=config.get('model', 'gpt-4o'),
                messages=[{"role": "system", "content": "You are an e-commerce operations AI analyst. Return JSON array of strategies."},
                          {"role": "user", "content": prompt}],
                response_format={"type": "json_object"},
                temperature=config.get('temperature', 0.3),
                max_tokens=config.get('max_tokens', 2000),
            )
            data = json.loads(resp.choices[0].message.content)
            print("  [LLM] Success")
        except Exception as e:
            print(f"  [LLM] Failed: {e}. Falling back to rules.")
            build_fallback_strategies(alerts_df)
    else:
        print(f"  [LLM] No API key configured. Using heuristic fallback.")
        build_fallback_strategies(alerts_df)

    report_md = "# AI 决策报告\n\n"
    report_md += f"生成时间: {datetime.now(timezone.utc).isoformat()}\n\n"
    report_md += f"模式: {args.mode}\n"
    report_md += f"分析异常: {len(alerts_df)} 条\n\n"
    report_md += "## 异常概览\n"
    for _, a in alerts_df.iterrows():
        report_md += f"- **{a.get('rule_id')}** (severity: {a.get('severity')}): {a.get('description', '')}\n"
    with open(os.path.join(OUTPUTS_DIR, 'ai', 'decision_report.md'), 'w') as f:
        f.write(report_md)

    if not os.path.exists(os.path.join(OUTPUTS_DIR, 'ai', 'review_retro_draft.json')):
        with open(os.path.join(OUTPUTS_DIR, 'ai', 'review_retro_draft.json'), 'w') as f:
            json.dump([], f)

    print(f"[ai] Done. Outputs in {os.path.join(OUTPUTS_DIR, 'ai')}/")


if __name__ == '__main__':
    main()
