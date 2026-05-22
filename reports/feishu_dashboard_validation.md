# 飞书看板核心口径校准报告

**生成日期**: 2026-05-22
**数据源**: daily_metrics_full.csv (634行), metric_alerts_full.csv, action_tasks_for_feishu_full.csv

## 核心组件校验

| # | 组件名称 | 数据源表 | 聚合方式 | 当前本地计算值 | 期望值参考 | 状态 |
|---|---------|---------|---------|--------------|-----------|------|
| 1 | 累计 GMV | daily_metrics_full | SUM(gmv) | 13,591,643.70 | > 1,000,000 | ✅ PASS |
| 2 | 累计订单数 | daily_metrics_full | SUM(order_count) | 99,441 | ≈99,441 | ✅ PASS |
| 3 | 最新业务日 GMV | daily_metrics_full | LAST(gmv) | 145.00 (日期: 2018-09-03) | > 0 | ⚠️ CHECK |
| 4 | 异常事件总数 | metric_alerts_full | COUNT(alert_id) | 19 | 10-200 | ✅ PASS |
| 5 | 待处理任务数 | action_tasks_for_feishu_full | COUNT(status=todo) | 19 | ≥0 | ✅ PASS |
| 6 | 平均评分范围 | daily_metrics_full | AVG(review_score) | 1.00 - 5.00 (均值: 4.01) | 3.5-4.5 | ⚠️ CHECK |

## 已知问题分析

1. **"配置数据发生变更"提示**: 可能原因：飞书仪表盘组件绑定的字段名与最新 `feishu_field_mapping.yml` 不一致。建议在飞书后台检查每个组件的字段绑定。

2. **"今日订单数显示 COUNT_ALL"**: 聚合方式可能误设为 `COUNT_ALL` 而非 `SUM`。确认飞书仪表盘组件的数据源和聚合配置。

3. **异常数与异常表不一致**: 飞书看板显示的异常数可能统计了不同范围的数据（如仅显示 high severity）。建议检查看板的筛选条件设置。

4. **闭环验证表空白**: `review_retro` 表的数据来自 `execution_reviews_for_feishu_full.csv`（当前为采样数据5-10行），确认飞书表是否已同步该数据。

## 异常细节

- **GMV 尾部数据缺失**: 本地 CSV 最后一行 (2018-10-17) 的 GMV 为 NaN，最后一个有效 GMV 日期为 2018-09-03。飞书看板若显示最新日期 GMV，需确认是否使用了正确的 LAST 聚合或需要过滤空值。
- **评分范围异常宽**: avg_review_score 实际范围 1.00-5.00，均值 4.01 在预期范围内，但分布极宽，建议检查数据中是否存在异常低分数据点。

## 建议

- 所有数值均来自本地 CSV 计算，飞书看板若显示不同值，应在飞书后台检查数据源绑定
- 优先修复累计 GMV 和累计订单数（用户最关注的 KPI）
- review_retro 表可在有真实运营反馈后补充真实数据
- 建议清洗 daily_metrics_full.csv 末尾的 NaN 行或添加 LASTNonNull 逻辑
