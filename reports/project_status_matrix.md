# 项目状态矩阵

> 最后更新: 2026-05-22

## 模块组总览

| 模块组 | 包含内容 | 状态 |
|--------|---------|------|
| Stage 1: 数据分析资产 | Phase 1-7 原始分析、基础表 (order/item_level_base)、图表 (36 PNG)、报告 (8 MD) | FROZEN |
| Stage 2: 数据产品层 | AIP 层 (6 JSON)、12 核心指标 (metrics.yml)、异常检测 (8 规则)、Context Bundle、治理配置 (14 YAML) | DONE |
| Stage 3: 飞书工作台 | 5 张表 (daily_metrics/alert_events/recommendations/action_tasks/review_retro)、字段映射、同步脚本 (FeishuClient SDK)、视图/仪表盘 | PARTIAL |
| Stage 4: 决策闭环沙盘 | 规则建议 (run_wake_agent/run_ai_decision_engine)、任务生成、复盘样例 (generate_review_retro_samples)、状态回流 (pull_feishu_status) | PARTIAL |
| AI 大模型决策 | Qoder/LLM 接入、真实 LLM 策略生成 | NOT_STARTED |
| 自动化调度 | 定时任务 (cron)、自动触发、实时监控 | NOT_STARTED |
| 环境与测试 | requirements.txt、runbook、pytest 测试集 | NOT_STARTED |
| 文档与入口 | README、RUNBOOK、4 入口命令标准化 | IN_PROGRESS |

## 阶段说明

- **第一阶段 (Phase 1-2)**: 原始数据理解 + 数据模型搭建 → FROZEN（历史资产，已归档）
- **第二阶段 (Phase 3-5)**: 全局业务分析 + 履约分析 + 营销漏斗 → FROZEN（历史资产，已归档）
- **第三阶段 (Phase 7)**: 决策沙盘模拟器 → FROZEN（历史资产，已归档）
- **当前版本**: v0.1-heuristic-decision-sandbox（规则驱动的 AI-ready 决策沙盘，不含真实 LLM 决策）

## 状态定义

| 状态标签 | 含义 |
|----------|------|
| DONE | 该模块所有开发工作已完成，产出物已验证可用 |
| PARTIAL | 该模块核心功能已完成，但存在已知待补充项（见各模块详情） |
| FROZEN | 该模块为历史资产，已归档。代码和产出保持不变，不再修改 |
| NOT_STARTED | 该模块尚未开始开发，处于规划阶段 |
| BLOCKED | 该模块因外部依赖或技术障碍被阻塞，需特定条件解除 |
| READY_FOR_TEST | 该模块开发完成，等待测试验证 |
| IN_PROGRESS | 该模块正在开发中，尚未完成 |
