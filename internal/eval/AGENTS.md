# eval: LLM 决策评估与回放

**Branch:** main

## OVERVIEW
LLM 决策质量评估、回放与对比分析。6 个生产文件，覆盖 4 个核心能力：决策评分、回放、对比、指标收集。

## WHERE TO LOOK

| 任务 | 文件 | 说明 |
|------|------|------|
| 决策质量评估 | `decision_eval.go` | 9 维度评分（安全性/合规性/准确性等），持久化到 `ai.decision_eval_result` |
| 决策对比 | `comparison.go` | LLM 决策 vs 规则决策的 Jaccard 相似度、severity/type 匹配 |
| 决策回放 | `replay.go` | 用相同上下文重新调用 LLM，对比原决策与新决策差异 |
| 指标收集 | `metrics.go` | 线程安全的内存计数器：成功率、回退率、延迟、审批率 |

## KEY PATTERNS

- **9 维度评估**: schema_validity, governance_compliance, action_safety, human_review_required, context_grounding, rationale_completeness, not_overgeneralized, has_owner_role, action_relevance
- **评估分组**: Safety 维度和 Grounding/Usefulness 维度各有不同通过阈值
- **回放 + 对比联动**: `ReplayService` 调用 provider 重新生成决策，`computeDecisionDiff` 产出结构化 diff
- **线程安全收集器**: `MetricsCollector` 使用 `sync.RWMutex` 保护计数器，`GetMetrics` 返回快照副本
- **可选 DB 持久化**: `DecisionEvaluator` 传入 nil pool 跳过数据库写入（测试模式）

## ANTI-PATTERNS

- MetricsCollector 为纯内存实现 — 重启后丢失，未对接 DB 持久化
- 评估维度阈值硬编码（pass ≥ 0.75）— 不支持配置化调整
- `PGReplayRepository` 直接写 SQL 而不是通过 repository 层
