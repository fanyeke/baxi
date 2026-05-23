"""
Generate deterministic review_retro sample records for Phase I-Local verification.
Pairs strategies with historical hindsight from daily_metrics_full.csv.
"""
import sys, os, json
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import *
from datetime import datetime, timezone, timedelta
import pandas as pd


def load_daily_full():
    import pandas as pd
    if not os.path.exists(DAILY_METRICS_FULL_FILE):
        return None
    return pd.read_csv(DAILY_METRICS_FULL_FILE, parse_dates=['simulated_date'])


def gen_reviews():
    import uuid
    import pandas as pd
    reviews = []
    df = load_daily_full()

    samples = [
        {
            'metric': 'cancel_rate', 'title': '促销期取消率飙升排查',
            'target_date': '2017-11-15', 'expected_direction': 'down'
        },
        {
            'metric': 'late_delivery_rate', 'title': '延迟配送专项优化',
            'target_date': '2017-06-01', 'expected_direction': 'down'
        },
        {
            'metric': 'avg_review_score', 'title': '低评分品类专项治理',
            'target_date': '2018-03-15', 'expected_direction': 'up'
        },
        {
            'metric': 'gmv', 'title': 'GMV异常波动追踪',
            'target_date': '2017-01-10', 'expected_direction': 'up'
        },
        {
            'metric': 'cancel_rate', 'title': '高取消率卖家专项监控',
            'target_date': '2017-08-20', 'expected_direction': 'down'
        },
        {
            'metric': 'order_count', 'title': '订单量下降调查',
            'target_date': '2017-04-15', 'expected_direction': 'up'
        },
        {
            'metric': 'late_delivery_rate', 'title': '区域配送异常排查',
            'target_date': '2018-01-15', 'expected_direction': 'down'
        },
        {
            'metric': 'seller_count', 'title': '活跃卖家流失调查',
            'target_date': '2017-10-01', 'expected_direction': 'up'
        },
    ]

    for i, s in enumerate(samples, 1):
        review_id = f"rev_{i:03d}"
        target = pd.Timestamp(s['target_date'])
        outcome = '已完成异常排查'
        is_effective = True
        promote = False
        impact_text = ''
        lessons = ''

        if df is not None:
            future = df[(df['simulated_date'] > target) & (df['simulated_date'] <= target + timedelta(days=7))]
            if not future.empty:
                metric = s['metric']
                if metric in future.columns:
                    val_now = future[metric].iloc[0]
                    val_before_val = df[df['simulated_date'] <= target].tail(1)
                    if not val_before_val.empty and metric in val_before_val.columns:
                        val_before = val_before_val[metric].iloc[0]
                        if pd.notna(val_before) and pd.notna(val_now) and val_before != 0:
                            change = (val_now - val_before) / abs(val_before) * 100
                            direction_good = (change < 0 and s['expected_direction'] == 'down') or \
                                             (change > 0 and s['expected_direction'] == 'up')
                            if direction_good or abs(change) < 5:
                                is_effective = True
                            else:
                                is_effective = False
                            impact_text = f"指标{s['metric']}变化{change:.1f}%"

        if is_effective and i <= 2:
            promote = True

        if i % 3 == 0:
            is_effective = False

        lessons_map = {
            1: '促销峰值后取消率异常应设置7日观察窗口，避免单日误报',
            2: '延迟配送问题需结合区域仓储容量评估，单一卖家排查效果有限',
            3: '低评分品类治理需联合品类运营和卖家运营双线推进',
            4: 'GMV波动需结合促销日历分析，纯数据异常可能为业务事件',
            5: '高取消率卖家的持续监控比单次排查更有效',
            6: '订单量下降调查需区分季节性因素和结构性因素',
            7: '区域配送异常往往与物流基础设施变动相关',
            8: '活跃卖家流失应提前设置卖家生命周期预警',
        }
        lessons = lessons_map.get(i, '')

        target_dt = target + timedelta(days=7)
        reviews.append({
            'review_id': review_id,
            'strategy_id': f"rec_{i * 10:03d}",
            'outcome': outcome,
            'actual_impact': impact_text if impact_text else '指标有改善',
            'is_effective': is_effective,
            'lessons_learned': lessons,
            'promote_to_rule': promote,
            'reviewed_at': target_dt.strftime('%Y-%m-%d %H:%M:%S'),
            'review_type': 'simulated',
            'review_source': 'hindsight_rule',
        })

    return reviews


def main():
    reviews = gen_reviews()
    os.makedirs(os.path.dirname(os.path.join(OUTPUTS_DIR, 'ai', 'review_retro_draft.json')), exist_ok=True)
    os.makedirs(os.path.join(OUTPUTS_DIR, 'ai'), exist_ok=True)
    output = os.path.join(OUTPUTS_DIR, 'ai', 'review_retro_draft.json')
    with open(output, 'w') as f:
        json.dump(reviews, f, indent=2, ensure_ascii=False)

    effective = sum(1 for r in reviews if r['is_effective'])
    promoted = sum(1 for r in reviews if r['promote_to_rule'])
    print(f"[review] Generated {len(reviews)} review records at {output}")
    print(f"  effective: {effective}, ineffective: {len(reviews)-effective}, promoted: {promoted}")


if __name__ == '__main__':
    main()
