import sys, os, json, csv, yaml
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from datetime import datetime, timezone
from scripts.config import *


def load_json(path):
    if os.path.exists(path):
        with open(path) as f:
            return json.load(f)
    return None


def load_yaml(path):
    if os.path.exists(path):
        with open(path) as f:
            return yaml.safe_load(f)
    return None


def check(label, condition, detail=""):
    status = "PASS" if condition else "FAIL"
    print(f"  [{status}] {label}{' - ' + detail if detail else ''}")
    return {"label": label, "status": status, "detail": detail}


def main():
    print("=== Local Loop Validation ===\n")
    results = []
    now = datetime.now(timezone.utc).isoformat()

    # --- 1. ingestion_state 推进 ---
    state = load_json(INGESTION_STATE_FILE) or {}
    next_date = state.get("next_simulated_date", "unknown")
    last_date = state.get("last_completed_simulated_date")
    results.append(check("ingestion_state推进", next_date > "2016-09-04", f"next={next_date}, last={last_date}"))

    # --- 2. daily_metrics 无未来数据 ---
    import pandas as pd
    future_count = 0
    dm = pd.DataFrame()
    if os.path.exists(DAILY_METRICS_FILE):
        dm = pd.read_csv(DAILY_METRICS_FILE)
        if last_date and len(dm) > 0:
            cutoff = pd.Timestamp(last_date).date()
            dm_dates = pd.to_datetime(dm["simulated_date"]).dt.date
            future_count = (dm_dates > cutoff).sum()
    results.append(check("daily_metrics无未来数据", future_count == 0, f"{future_count} future rows, {len(dm)} total"))

    alerts_ok = True
    alerts_count = 0
    if os.path.exists(METRIC_ALERTS_FILE):
        ma = pd.read_csv(METRIC_ALERTS_FILE)
        alerts_count = len(ma)
        if "alert_id" in ma.columns:
            dupes = ma["alert_id"].duplicated().sum()
            alerts_ok = dupes == 0
    results.append(check("metric_alerts无重复alert_id", alerts_ok, f"{alerts_count} alerts"))

    bundle = load_json(AIP_CONTEXT_BUNDLE_LATEST_FILE) or {}
    snap = bundle.get("snapshot_date", "")
    bundle_match = (snap == last_date)
    results.append(check("bundle日期一致", bundle_match, f"snapshot={snap}, last_completed={last_date}"))

    wake_dir = os.path.join(OUTPUTS_DIR, "wake")
    wake_files = {
        "daily_report.md": os.path.exists(os.path.join(wake_dir, "daily_report.md")),
        "feishu_message.json": os.path.exists(os.path.join(wake_dir, "feishu_message.json")),
        "strategy_recommendations.json": os.path.exists(os.path.join(wake_dir, "strategy_recommendations.json")),
        "action_tasks.json": os.path.exists(os.path.join(wake_dir, "action_tasks.json")),
    }
    wake_ok = all(wake_files.values())
    missing = [k for k, v in wake_files.items() if not v]
    results.append(check("Wake输出完整", wake_ok, f"missing: {missing}" if missing else "4/4"))

    fmsg = load_json(os.path.join(wake_dir, "feishu_message.json"))
    msg_ok = bool(fmsg and "title" in fmsg and "content" in fmsg)
    results.append(check("Wake feishu_message结构", msg_ok))

    srecs = load_json(os.path.join(wake_dir, "strategy_recommendations.json"))
    rec_ok = isinstance(srecs, list) and all(
        "recommendation_id" in r and "owner_role" in r for r in srecs
    ) if srecs else True
    results.append(check("Wake recommendations结构", rec_ok, f"{len(srecs) if srecs else 0} items"))

    feishu_all = True
    for fname in [
        "daily_metrics_for_feishu.csv", "metric_alerts_for_feishu.csv",
        "strategy_recommendations_for_feishu.csv", "action_tasks_for_feishu.csv",
        "execution_reviews_for_feishu.csv",
    ]:
        if not os.path.exists(os.path.join(FEISHU_DIR, fname)):
            feishu_all = False
            break
    results.append(check("飞书沙盘5张CSV完整", feishu_all))

    entries = 0
    if os.path.exists(RUN_MANIFEST_FILE):
        with open(RUN_MANIFEST_FILE) as f:
            rows = list(csv.reader(f))
        entries = len(rows) - 1
        stages = set()
        for r in rows[1:]:
            if len(r) > 3:
                stages.add(r[3])
        results.append(check("run_manifest有记录", entries > 0, f"{entries} entries"))
        results.append(check("run_manifest覆盖多阶段", len(stages) >= 3, f"stages: {sorted(stages)}"))
    else:
        results.append(check("run_manifest存在", False))

    orders_path = os.path.join(RAW_DIR, "olist_orders_dataset.csv")
    with open(orders_path) as f:
        lines = sum(1 for _ in f)
    raw_ok = lines == 99442
    results.append(check("原始数据不变", raw_ok, f"{lines} lines"))

    # --- Summary ---
    passed = sum(1 for r in results if r["status"] == "PASS")
    failed = sum(1 for r in results if r["status"] == "FAIL")
    total = len(results)

    print(f"\n=== Summary: {passed}/{total} PASS, {failed} FAIL ===\n")

    # --- Generate report ---
    report_path = os.path.join(REPORTS_DIR, "local_loop_validation_report.md")
    os.makedirs(REPORTS_DIR, exist_ok=True)
    with open(report_path, "w") as f:
        f.write(f"# 本地闭环验收报告\n\n")
        f.write(f"**生成时间**: {now}\n\n")
        f.write(f"## 1. 连续运行概况\n\n")
        f.write(f"| 指标 | 值 |\n|------|----|\n")
        f.write(f"| ingestion_state next | {next_date} |\n")
        f.write(f"| ingestion_state last_completed | {last_date} |\n")
        f.write(f"| daily_metrics 行数 | {len(dm) if 'dm' in dir() and 'dm' in locals() else 'N/A'} |\n")
        f.write(f"| metric_alerts 总数 | {alerts_count} |\n")
        f.write(f"| run_manifest 条目 | {entries if 'entries' in dir() else 'N/A'} |\n\n")

        f.write(f"## 2. 各检查项结果\n\n")
        f.write(f"| 检查项 | 结果 | 详情 |\n|--------|------|------|\n")
        for r in results:
            f.write(f"| {r['label']} | {r['status']} | {r['detail']} |\n")

        f.write(f"\n## 3. 验收结论\n\n")
        if failed == 0:
            f.write(f"**判定: ✅ PASS** — 所有 {total} 项检查通过，系统可进入下一阶段。\n\n")
        else:
            f.write(f"**判定: ❌ FAIL** — {failed}/{total} 项未通过，需修复后重新验收。\n\n")

        f.write(f"## 4. 下一步建议\n\n")
        f.write(f"- 如全部 PASS: 可进入 Phase H 真实飞书 API 接入阶段\n")
        f.write(f"- 如存在 FAIL: 修复具体问题后重新运行 `replay_pipeline.py --days 30` + 本验收脚本\n")

    print(f"Report: {report_path}")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
