# Pipeline 管道系统

> `internal/pipeline/` — Go Step 接口编排的数据处理管道

## 设计原则

- **Step 接口**: 每个步骤实现 `Step.Run(ctx, tx, input)` 接口
- **事务隔离**: 每步运行在独立数据库事务中，成功提交、失败回滚
- **审计追踪**: 自动记录 input/output 行数到 `audit.pipeline_step_run`
- **可测试**: 每个 Step 可独立测试，通过 mock 事务隔离

## Step 接口

```go
type Step interface {
    Name() string
    Run(ctx context.Context, tx pgx.Tx, input StepInput) (*StepOutput, error)
}
```

## 管道步骤（7 步）

| 步骤 | 职责 | 输入 | 输出 |
|------|------|------|------|
| `ingest_raw` | CSV → 原始表 | CSV 数据 | 摄入行数 |
| `build_dwd` | DWD 层构建 | 原始表 | 宽表行数 |
| `build_metrics` | 指标聚合 | DWD 层 | 指标行数 |
| `detect_alerts` | 异常检测 | 指标表 | 告警数 |
| `generate_recommendations` | 策略建议 | 告警数据 | 建议数 |
| `generate_tasks` | 任务创建 | 建议数据 | 任务数 |
| `create_outbox` | 发件箱事件 | 任务数据 | 事件数 |

## 运行方式

```bash
# 完整管道
make pipeline

# 单步执行
make pipeline-ingest
make pipeline-dwd
make pipeline-metrics
make pipeline-compare

# CLI 直接运行
go run ./cmd/baxi-cli pipeline run --data-dir ./data/raw

# CLI 验证
go run ./cmd/baxi-cli pipeline validate --data-dir ./data/raw
```

## Runner 编排

`Runner` 负责：
1. 创建 `pipeline_run` 审计记录
2. 按顺序执行每个 Step
3. 每步：开启事务 → 执行 Step → 成功提交/失败回滚
4. 完成时更新审计记录状态
