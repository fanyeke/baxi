"""Validate Feishu Dashboard data by computing expected values from local CSVs.

Reads local feishu/ CSV files and prints expected dashboard values for manual
comparison against Feishu Dashboard components. No Feishu API calls needed.
"""

import os
import sys

import pandas as pd

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from scripts.config import FEISHU_DIR


def fmt_num(v, decimals=2):
    if v is None or (isinstance(v, float) and pd.isna(v)):
        return "N/A"
    if isinstance(v, float):
        return f"{v:.{decimals}f}"
    return str(v)


def main():
    print("=" * 60)
    print("  飞书 Dashboard 预期值校验")
    print("=" * 60)

    # --- daily_metrics ---
    dm_path = os.path.join(FEISHU_DIR, "daily_metrics_for_feishu.csv")
    if os.path.exists(dm_path):
        dm = pd.read_csv(dm_path)
        if "simulated_date" in dm.columns:
            latest = dm.loc[dm["simulated_date"].idxmax()]
            latest_date = latest.get("simulated_date", "N/A")
        else:
            latest = dm.iloc[-1] if len(dm) > 0 else None
            latest_date = "N/A"
    else:
        dm = pd.DataFrame()
        latest = None
        latest_date = "N/A"

    print(f"\n  最新业务日期: {latest_date}")
    print(f"  每日经营指标行数: {len(dm)}")
    print()

    if latest is not None:
        print("  --- 经营概览 KPI (最新业务日) ---")
        print(f"  GMV:              R$ {fmt_num(latest.get('gmv'))}")
        print(f"  订单数:             {fmt_num(latest.get('order_count'), 0)}")
        print(f"  客户数:             {fmt_num(latest.get('customer_count'), 0)}")
        print(f"  卖家数:             {fmt_num(latest.get('seller_count'), 0)}")
        print(f"  客单价:            R$ {fmt_num(latest.get('avg_order_value'))}")
        print(f"  平均评分:           {fmt_num(latest.get('avg_review_score'))}")
        print(f"  差评率:             {fmt_num(latest.get('low_review_rate'))}")
        print(f"  延迟配送率:         {fmt_num(latest.get('late_delivery_rate'))}")
        print(f"  取消率:             {fmt_num(latest.get('cancel_rate'))}")
        print(f"  异常事件数:         {fmt_num(latest.get('alert_count'), 0)}")

    # --- alert_events ---
    ae_path = os.path.join(FEISHU_DIR, "alert_events_for_feishu.csv")
    if os.path.exists(ae_path):
        ae = pd.read_csv(ae_path)
    else:
        ae = pd.DataFrame()

    total_alerts = len(ae)
    new_alerts = len(ae[ae["status"] == "new"]) if "status" in ae.columns else 0
    high_alerts = len(ae[ae["severity"] == "high"]) if "severity" in ae.columns else 0

    print(f"\n  --- 异常事件 ---")
    print(f"  异常事件表行数:     {total_alerts}")
    print(f"  全部异常数 (COUNT): {total_alerts}")
    print(f"  待处理异常数 (new):  {new_alerts}")
    print(f"  高优先级异常 (high): {high_alerts}")

    # --- strategy_recommendations ---
    sr_path = os.path.join(FEISHU_DIR, "strategy_recommendations_for_feishu.csv")
    if os.path.exists(sr_path):
        sr = pd.read_csv(sr_path)
    else:
        sr = pd.DataFrame()

    total_recs = len(sr)
    pending = len(sr[sr["approval_status"] == "pending_review"]) if "approval_status" in sr.columns else 0
    requires_approval_count = len(sr[sr["requires_approval"] == True]) if "requires_approval" in sr.columns else 0

    print(f"\n  --- 策略建议 ---")
    print(f"  策略建议表行数:     {total_recs}")
    print(f"  需审批建议数:       {pending}")
    print(f"  需审批 (True):      {requires_approval_count}")

    # --- action_tasks ---
    at_path = os.path.join(FEISHU_DIR, "action_tasks_for_feishu.csv")
    if os.path.exists(at_path):
        at = pd.read_csv(at_path)
    else:
        at = pd.DataFrame()

    total_tasks = len(at)
    todo_tasks = len(at[at["status"] == "todo"]) if "status" in at.columns else 0

    print(f"\n  --- 负责人任务 ---")
    print(f"  负责人任务表行数:   {total_tasks}")
    print(f"  待处理任务 (todo):   {todo_tasks}")

    # --- review_retro ---
    rr_path = os.path.join(FEISHU_DIR, "execution_reviews_for_feishu.csv")
    if os.path.exists(rr_path):
        rr = pd.read_csv(rr_path)
    else:
        rr = pd.DataFrame()

    print(f"\n  --- 执行复盘 ---")
    print(f"  执行复盘表行数:     {len(rr)}")
    if len(rr) == 0:
        print("  (空 — 闭环验证看板空白属于预期行为)")

    # --- latest_daily_metrics ---
    ldm_path = os.path.join(FEISHU_DIR, "latest_daily_metrics_for_feishu.csv")
    has_latest = os.path.exists(ldm_path)

    print(f"\n  --- 辅助表 ---")
    print(f"  latest_daily_metrics: {'✓ 已生成' if has_latest else '✗ 缺失'}")

    print(f"\n{'=' * 60}")
    print("  对照以上预期值，手动检查飞书 Dashboard 组件显示是否正确。")
    print("  不一致项说明：")
    print("    - 值匹配 → 数据层正常，问题在仪表盘配置")
    print("    - 值不匹配 → 检查数据生成或同步是否正常")
    print("=" * 60)


if __name__ == "__main__":
    main()
