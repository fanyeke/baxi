# Olist 每日经营分析报告

**快照日期**: 2016-10-04
**时间窗口**: 最近7天 (2016-09-28 ~ 2016-10-04)
**基线窗口**: 前14天 (2016-09-14 ~ 2016-09-27)
**报告生成时间**: 2026-05-21 09:12:17 UTC
**运行 ID**: f4e1629b77f91d84f7921d2c22cfe675

## 核心指标概览

| 指标 | 当前值 | 基线值 | 变化 | 状态 |
|------|--------|--------|------|------|
| avg_order_value | R$ 105.24 | R$ 134.97 | ↓ -0.2% | ℹ️ |
| avg_review_score | 2.54 | 1.00 | ↑ +1.5% | ℹ️ |
| cancel_rate | 40.2% | 0.0% | baseline unavailable (early stage) | ℹ️ |
| customer_count | 24 | 1 | ↑ +23.0% | ⚠️ |
| freight_value | R$ 498.51 | R$ 8.49 | ↑ +57.7% | ⚠️ |
| gmv | R$ 3,501.48 | R$ 134.97 | ↑ +24.9% | ⚠️ |
| late_delivery_rate | 1.8% | 100.0% | ↓ -1.0% | ℹ️ |
| low_review_rate | 55.4% | 100.0% | ↓ -0.4% | ℹ️ |
| marketing_seller_share | 0.0% | 0.0% | baseline unavailable (early stage) | ℹ️ |
| order_count | 24 | 1 | ↑ +23.0% | ⚠️ |
| payment_installment_rate | 38.4% | 0.0% | baseline unavailable (early stage) | ℹ️ |
| seller_count | 15 | 1 | ↑ +14.7% | 📊 |

## 指标观察

### 新增事件 (3 条)

- **f47e6d6903a04999bba3fd391818527c**: N/A  [severity=N/A]
- **f9145dcb3adc47808f14aabaa002ff0d**: N/A  [severity=N/A]
- **8066d9fce1bf48af9b7fca7ee4e3d90d**: N/A  [severity=N/A]

### ⚠️ 需关注指标

- **customer_count**: 当前值 24, 变化 ↑ +23.0%
- **freight_value**: 当前值 R$ 498.51, 变化 ↑ +57.7%
- **gmv**: 当前值 R$ 3,501.48, 变化 ↑ +24.9%
- **order_count**: 当前值 24, 变化 ↑ +23.0%

## 建议与行动项

| 优先级 | 指标 | 当前值 | 负责人 | 建议行动 |
|--------|------|--------|--------|----------|
| P0 | customer_count | 24 | business_ops | 建立基线后持续监控 |
| P0 | freight_value | R$ 498.51 | logistics_ops | 建立基线后持续监控 |
| P0 | gmv | R$ 3,501.48 | business_ops | 建立基线后持续监控 |
| P0 | order_count | 24 | business_ops | 建立基线后持续监控 |
| P1 | seller_count | 15 | business_ops | 建立基线后持续监控 |

## 下一步

1. 持续采集数据，建立 14 天滚动基线
2. 配置 metric alert 规则，开启异常自动检测
3. 下一周期快照将根据基线对比输出环比变化
4. 当前平均评分较低 (avg_review_score=2.5)，建议重点关注卖家服务质量

---
*报告由 Wake Agent 自动生成 | Run ID: f4e1629b77f91d84f7921d2c22cfe675*