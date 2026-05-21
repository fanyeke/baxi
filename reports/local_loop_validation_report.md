# 本地闭环验收报告

**生成时间**: 2026-05-21T09:08:18.251973+00:00

## 1. 连续运行概况

| 指标 | 值 |
|------|----|
| ingestion_state next | 2016-10-04 |
| ingestion_state last_completed | 2016-10-03 |
| daily_metrics 行数 | 6 |
| metric_alerts 总数 | 2 |
| run_manifest 条目 | 180 |

## 2. 各检查项结果

| 检查项 | 结果 | 详情 |
|--------|------|------|
| ingestion_state推进 | PASS | next=2016-10-04, last=2016-10-03 |
| daily_metrics无未来数据 | PASS | 0 future rows, 6 total |
| metric_alerts无重复alert_id | PASS | 2 alerts |
| bundle日期一致 | PASS | snapshot=2016-10-03, last_completed=2016-10-03 |
| Wake输出完整 | PASS | 4/4 |
| Wake feishu_message结构 | PASS |  |
| Wake recommendations结构 | PASS | 3 items |
| 飞书沙盘5张CSV完整 | PASS |  |
| run_manifest有记录 | PASS | 180 entries |
| run_manifest覆盖多阶段 | PASS | stages: ['', 'aip_bundle', 'alert_detection', 'data_quality_check', 'ingestion', 'pipeline_run'] |
| 原始数据不变 | PASS | 99442 lines |

## 3. 验收结论

**判定: ✅ PASS** — 所有 11 项检查通过，系统可进入下一阶段。

## 4. 下一步建议

- 如全部 PASS: 可进入 Phase H 真实飞书 API 接入阶段
- 如存在 FAIL: 修复具体问题后重新运行 `replay_pipeline.py --days 30` + 本验收脚本
