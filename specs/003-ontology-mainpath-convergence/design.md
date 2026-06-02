# Ontology v2 主路径收敛 + Pi Agent 决策写回 — Ultrawork 设计

## 诊断摘要

### 当前状态
- **配置覆盖**: 12 objects, 18 metrics, 5 recipes — 完成
- **安全门禁**: execute_action validation-only, execute_proposal dry-run + env gate — 完成
- **Action 写回**: decision_json/evidence_refs/context_hash/recipe_id — schema 完成

### 核心问题
**E2E 测试明确写了** (`test/integration/ontology_v2_e2e_test.go:421-423`):
```
Repository without v2 compiler — uses the v1 hardcoded table mapping.
The v2 compiler path has known issues.
```

**根因分析**（从代码审计得出）:

1. **`metric_ref` 字段污染 SELECT** — `compiler.go:34-47` 遍历所有 properties 生成 SELECT，包括 `metric_ref` 字段（如 seller 的 `late_delivery_rate`），但这些字段没有 `source` 列，导致 SQL 语法错误
2. **无 `availability` 标记** — schema_v2.go 中 `ObjectPropertyV2` 没有 `Availability` 字段，planned 字段（如 review.review_text、payment.payment_type）与 real 字段混在一起
3. **expression 无安全校验** — `compiler.go:36` 直接拼接 `prop.Expression` 到 SQL，无注入防护
4. **query_ref 无安全校验** — `link_plan.go` 中 query_ref 策略的 SQL 模板缺乏 DML 检测
5. **v1 fallback 日志不完整** — `repository.go:300-301` 有 `slog.Warn` 但只覆盖 `QueryByObjectType`，`GetObjectByID` 和 `GetObjectMetrics` 的 fallback 日志未被观测

## 模型搭配策略

| 任务类型 | 模型 | 理由 |
|---------|------|------|
| TDD 测试编写 | Haiku 4.5 | 大量重复性测试生成，节省 3x 成本 |
| 编译器核心逻辑 | Sonnet/Opus | 安全关键路径，需要精确推理 |
| YAML 配置修改 | Haiku 4.5 | 模板化修改，低推理需求 |
| 安全校验器 | Opus | SQL 注入防护，不正确则灾难性 |
| E2E 验证 | Sonnet | 需要理解全链路 |
| 代码审查 | Sonnet | 平衡质量与成本 |

## 执行阶段

### 阶段 1 (P0): v2 QueryCompiler 主路径收敛
### 阶段 2 (P0): 字段真实性 / availability / maturity
### 阶段 3 (P1): Context Recipe 质量收敛
### 阶段 4 (P1): Pi Agent 只读决策
### 阶段 5 (P1): Pi Agent 受控写回
### 阶段 6 (P2): v1 fallback 收敛

每个阶段遵循: TDD 红灯 → 实现 → 绿灯 → 重构 → 验证 80%+
