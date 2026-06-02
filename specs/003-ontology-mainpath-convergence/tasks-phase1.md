# Phase 1 (P0): v2 QueryCompiler 主路径收敛 — 任务清单

## TDD 红灯测试（先写，预期失败）

### 1.1 metric_ref 字段不进入 SELECT
- **文件**: `internal/ontology/compiler_test.go` (新建)
- **测试**: `TestCompileGetObject_SkipsMetricRefFields`
  - 构造一个含 `metric_ref` 属性的 ObjectTypeV2
  - 调用 CompileGetObject
  - 断言: SQL 不含 metric_ref 列名，MetricRefs 列表包含该引用
- **测试**: `TestCompileObjectMetrics_SkipsMetricRefFields`
  - 同上针对 CompileObjectMetrics

### 1.2 planned 字段不进入 SELECT
- **测试**: `TestCompileGetObject_SkipsPlannedFields`
  - 构造含 `availability: planned` 的属性
  - 断言: SQL 不包含 planned 字段

### 1.3 expression 安全校验
- **文件**: `internal/ontology/expression_validator_test.go` (新建)
- **测试**: `TestValidateExpression_RejectsDML`
  - 输入: "DROP TABLE dwd.item_level"
  - 断言: 返回 error
- **测试**: `TestValidateExpression_RejectsSemicolon`
  - 输入: "AVG(price); DELETE FROM dwd.item_level"
  - 断言: 返回 error
- **测试**: `TestValidateExpression_AllowsWhitelistFunctions`
  - 输入各种 AVG/SUM/COUNT/MIN/MAX/CASE/COALESCE

### 1.4 query_ref 安全校验
- **测试**: `TestValidateQueryRef_RejectsDML`
- **测试**: `TestValidateQueryRef_RequiresSelect`
- **测试**: `TestValidateQueryRef_RequiresParam`

### 1.5 v1 fallback 可观测化
- **测试**: `TestFallbackLogsWarning`
  - 验证当 v2 compilation 失败时产生 WARN 日志
  - 验证日志包含 object_type 和 reason

## 实现任务

### 1.A 扩展 CompiledQuery 结构体
- 在 `schema_v2.go` 的 `CompiledQuery` 中增加:
  - `MetricRefs []string`
  - `VirtualProperties []string`

### 1.B 扩展 ObjectPropertyV2 结构体
- 增加 `Availability string` 字段 (real/virtual/planned)

### 1.C 修改 QueryCompiler
- `CompileGetObject`: 跳过 metric_ref 和 planned 字段
- `CompileSearchObjects`: 同上
- `CompileObjectMetrics`: 同上
- metric_ref 字段收集到 MetricRefs

### 1.D 新建 ExpressionValidator
- 文件: `internal/ontology/expression_validator.go`
- DML 检测 (INSERT/UPDATE/DELETE/DROP/ALTER)
- 分号检测
- 注释检测 (-- 和 /* */)
- 白名单函数校验
- 引用字段校验

### 1.E 新建 QueryRefValidator
- SELECT 开头检测
- $1 参数检测
- DML 检测
- 表白名单校验

### 1.F 增强 v1 fallback 日志
- 所有 fallback 打 `slog.Warn("ontology_v1_fallback", ...)`
- 统一日志字段: object_type, reason
- 可选 metric 计数

### 1.G 更新 YAML 配置
- 给所有 property 加 `availability` 字段

## 验证标准
1. `go test ./internal/ontology/... -v` 全部通过
2. E2E test `TestOntologyV2E2E` 不再跳过 v2 compiler
3. seller/order/product/customer/category/region 的 get_object 走 v2
4. metric_ref 不进入 SELECT
5. planned 字段不进入 SELECT
6. query_ref validator 拦截危险 SQL
7. fallback 有明确 WARN 日志
