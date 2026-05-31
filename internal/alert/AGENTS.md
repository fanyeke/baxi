# alert: 维度级异常检测

**Branch:** main

## OVERVIEW
全局 + 维度级告警引擎，对 mart.metric_daily 和 mart.metric_dimension_daily 表执行规则评估。4 个生产文件。

## WHERE TO LOOK

| 任务 | 文件 | 说明 |
|------|------|------|
| 告警引擎 | `engine.go` | `Engine` 结构体 + `EvaluateGlobalRules` + 维度规则执行 |
| 规则定义 | `rule.go` | `AlertRule` 结构体、severity 类型、`GlobalRules()`、`GenerateAlertID` |
| 全局规则评估 | `engine.go` | evaluateGMVDrop, evaluateLateDeliverySpike, evaluateCancelRateSpike |
| 维度规则评估 | `engine.go` | `ExecuteDimensionalRule`, `EvaluateDimensionRules`, `SuppressAlerts` |

## KEY PATTERNS

- **两级规则**: 全局规则（基于 % 变化/阈值）+ 维度规则（seller/category/region 维度）
- **可配置死规则**: `Enabled=false` 规则（review_score_drop, seller_activation_gap）跳过执行但保留定义
- **告警抑制**: `SuppressAlerts` 按 severity → impact_score → sample_size 排序，上限 50 条/run
- **确定性 ID**: `GenerateAlertID` 使用 SHA-256(rule_id + date + dim) 生成可复现的 alert ID
- **规则条件**: 支持 `value_gt`, `value_lt`, `change_rate_gt`, `change_rate_lt` 四种条件格式
- **6 默认维度规则**: seller_late_delivery_spike, seller_review_score_drop, category_gmv_drop, category_low_review_cluster, region_cancel_rate_spike, region_late_delivery_spike

## ANTI-PATTERNS

- `evaluateGMVDrop` 等规则函数作为 `Engine{}` 的包级函数而非方法 — 可测试性受限
- 维度规则配置在 `engine.go` 中硬编码 — 未从 `config/dimensional_alert_rules.yml` 动态加载
- Engine 结构体为空（无状态）— 所有依赖通过函数参数传递
