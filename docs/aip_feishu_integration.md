# AIP 项目与飞书集成指南

## 1. 飞书表字段映射

### 1.1 每日经营指标表
| 本地字段 | 飞书字段 |
|----------|----------|
| real_run_date | 执行日期 |
| simulated_date | 业务日期 |
| gmv | GMV(R$) |
| order_count | 订单数 |
| customer_count | 客户数 |
| seller_count | 卖家数 |
| avg_order_value | 客单价(R$) |
| avg_review_score | 平均评分 |
| low_review_rate | 差评率 |
| late_delivery_rate | 延迟配送率 |
| cancel_rate | 取消率 |
| payment_installment_rate | 分期支付率 |
| marketing_seller_share | 营销卖家占比 |

### 1.2 异常事件表
| 本地字段 | 飞书字段 |
|----------|----------|
| alert_id | 事件ID |
| simulated_date | 日期 |
| severity | 严重等级 |
| metric_name | 指标 |
| current_value | 当前值 |
| baseline_value | 基线值 |
| status | 状态 |
| owner_role | 负责人角色 |

### 1.3 策略建议表
| 本地字段 | 飞书字段 |
|----------|----------|
| recommendation_id | 建议ID |
| event_id | 关联事件ID |
| title | 策略标题 |
| detail | 策略详情 |
| expected_impact | 预期影响 |
| risk_level | 风险等级 |
| status | 状态 |

### 1.4 负责人任务表
| 本地字段 | 飞书字段 |
|----------|----------|
| task_id | 任务ID |
| title | 任务标题 |
| description | 任务说明 |
| owner | 负责人 |
| priority | 优先级 |
| status | 状态 |

### 1.5 执行复盘表
| 本地字段 | 飞书字段 |
|----------|----------|
| review_id | 复盘ID |
| strategy_id | 策略ID |
| outcome | 执行结果 |
| is_effective | 是否有效 |
| lessons_learned | 经验总结 |

## 2. 同步频率

| 表 | 频率 | 触发条件 |
|----|------|----------|
| daily_metrics | daily | 每日指标计算完成后 |
| alert_events | on_alert | 有新异常事件时 |
| recommendations | on_recommendation | 有新策略建议时 |
| action_tasks | on_task | 有新待执行任务时 |
| review_retro | manual | 人工填写 |

## 3. Wake Agent 消费流程

```
1. Wake 读取 aip_context_bundle.json
   - 包含：snapshot_date, metrics, events, recommendations, allowed_actions, owner_mapping
2. Wake 分析 events 中的异常事件
3. 根据 severity 和 owner_role 决定：
   - 创建飞书日报 → create_feishu_report
   - 通知负责人 → notify_owner
   - 创建任务 → create_followup_task
   - 推荐策略 → recommend_business_strategy
4. Wake 输出：
   - data/outputs/daily_report.md
   - data/outputs/feishu_message.json
   - data/outputs/strategy_recommendations.json
   - data/outputs/action_tasks.json
5. sync_to_feishu.py 将输出同步到飞书多维表格
```

## 4. 异常通知路由

| 严重等级 | owner_role | 通知方式 |
|----------|-----------|----------|
| high | business_ops | 实时通知 + 飞书群消息 |
| high | seller_ops | 1小时内创建任务 |
| high | logistics_ops | 实时通知 |
| medium | category_ops | 每日摘要 |
| medium | marketing_ops | 每周回顾 |
| low | 所有 | 仅写入日志 |

## 5. 后续接入飞书API步骤

1. 在飞书开放平台创建应用，获取 app_id 和 app_secret
2. 实现 tenant_access_token 获取
3. 通过 `Feishu Bitable API` `POST /open-apis/bitable/v1/apps/{app_token}/tables/{table_id}/records` 创建记录
4. 通过字段映射文件 `config/feishu_field_mapping.yml` 映射本地字段到飞书 field_id
5. 实现 `sync_to_feishu.py` 的实际同步逻辑（当前为骨架版）
6. 在 `scripts/sync_to_feishu.py` 中替换 `# TODO` 部分

---
**文档创建时间**: 2026-05-21
**参考**: `config/feishu_base_schema.yml`, `config/feishu_field_mapping.yml`, `docs/waker_read_write_contract.md`
