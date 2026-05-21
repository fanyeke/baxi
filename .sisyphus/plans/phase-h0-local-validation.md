# Phase H-0：本地闭环验收

## TL;DR

> **Quick Summary**: 在接入真实飞书前，验证本地闭环稳定性。新增3个脚本（统一管道 + 连续回放 + 自动验收），连续模拟30天，生成验收报告。
>
> **Deliverables**:
> - `scripts/run_daily_pipeline.py` — 一键执行全部8个管道步骤
> - `scripts/replay_pipeline.py` — 连续模拟N天（`--days 30`）
> - `scripts/validate_local_loop.py` — 自动验收本地闭环
> - 30天 replay 运行产出
> - `reports/local_loop_validation_report.md` — 验收报告
>
> **Estimated Effort**: Medium（5任务，1阶段）
> **Parallel Execution**: 3脚本可并行编写，30天回放为串行验证
> **Critical Path**: 3脚本 → 30天replay → 验收报告

---

## Context

### 当前状态

| 指标 | 值 |
|------|-----|
| ingestion_state | next=2016-09-07, last_completed=2016-09-06 |
| run_manifest 条目 | 8 |
| 管道脚本 | 8个均存在且可运行 |
| 单日运行 | ✅ 稳定 |
| 连续多日运行 | ❌ 未验证 |
| 统一入口 | ❌ 无 |
| 验收机制 | ❌ 无 |

### 缺失的产物

| 文件 | 状态 |
|------|------|
| `scripts/run_daily_pipeline.py` | ❌ |
| `scripts/replay_pipeline.py` | ❌ |
| `scripts/validate_local_loop.py` | ❌ |
| `reports/local_loop_validation_report.md` | ❌ |

---

## Work Objectives

### Core Objective
验证本地闭环在连续30天模拟运行下的稳定性，确认系统可进入真实飞书API接入阶段。

### 5个具体目标
1. 连续模拟多天是否稳定（不崩溃、不丢数据）
2. 指标计算是否符合业务逻辑（as_of_date 持续生效）
3. 异常规则是否能在足够数据后触发
4. Wake Agent 输出是否稳定、可控、有证据
5. 飞书沙盘字段是否足够承接后续真实飞书同步

---

## TODOs

- [ ] 1. 创建 `scripts/run_daily_pipeline.py`

**What to do**:
- 创建统一的每日管道入口脚本
- 按顺序调用 8 个步骤：
  ```
  simulate_daily_ingestion     → run_data_quality_checks
  → calculate_daily_metrics     → run_alert_detection
  → build_aip_objects          → build_aip_context_bundle
  → run_wake_agent             → generate_feishu_sandbox
  ```
- 每步失败立即停止（`subprocess.run` + `check=True`）
- 每步运行前后记录时间戳，写入 run_manifest（pipeline_stage='pipeline_run'）
- 打印清晰的步骤进度条
- 使用 `from scripts.config import *` 管理路径
- 支持 `--skip-quality` 跳过数据质量校验（T4已知问题）

**Must NOT do**:
- 不并行执行（有严格顺序依赖）
- 不跳过中间任何步骤
- 不使用硬编码路径

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**Key references**:
- `scripts/simulate_daily_ingestion.py` — 步骤1
- `scripts/run_data_quality_checks.py` — 步骤2
- `scripts/calculate_daily_metrics.py` — 步骤3
- `scripts/run_alert_detection.py` — 步骤4
- `scripts/build_aip_objects.py` — 步骤5
- `scripts/build_aip_context_bundle.py` — 步骤6
- `scripts/run_wake_agent.py` — 步骤7
- `scripts/generate_feishu_sandbox.py` — 步骤8
- `scripts/config.py` — 路径常量

**Acceptance Criteria**:
- [ ] `python3 scripts/run_daily_pipeline.py` 执行成功（全部8步通过）
- [ ] 每步有清晰输出和时间戳
- [ ] run_manifest 新增 pipeline_run 条目
- [ ] 任一子步骤失败时，管道立即停止并报告失败原因

---

- [ ] 2. 创建 `scripts/replay_pipeline.py`

**What to do**:
- 创建连续回放脚本
- 用法：`python3 scripts/replay_pipeline.py --days 30`
- 内部循环调用 `run_daily_pipeline.py`
- 每次迭代：
  - 记录迭代序号和状态
  - 失败时记录错误并继续（不中断整个回放）
- 最终汇总：`total_days / success_count / fail_count`
- 输出进度条（`[15/30] OK 2016-09-21`）

**Must NOT do**:
- 不在子进程失败时退出整个回放
- 不跳过失败的日期
- 不使用硬编码路径

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**Key references**:
- `scripts/run_daily_pipeline.py` — 任务1产物

**Acceptance Criteria**:
- [ ] `python3 scripts/replay_pipeline.py --days 30` 跑完30天
- [ ] 输出 success=30, fail=0
- [ ] run_manifest 有30组管道条目
- [ ] ingestion_state 推进到约 2016-10-06

---

- [ ] 3. 创建 `scripts/validate_local_loop.py`

**What to do**:
- 创建本地闭环验收脚本
- **不修改任何数据，只读检查**
- 检查清单：

| # | 检查项 | 验证逻辑 |
|---|--------|----------|
| 1 | ingestion_state推进 | next_simulated_date > 初始值 |
| 2 | daily_metrics无未来数据 | 所有 simulated_date <= last_completed |
| 3 | metric_alerts无重复 | alert_id 唯一 |
| 4 | bundle日期一致 | aip_context_bundle_latest.snapshot_date == last_completed |
| 5 | Wake输出符合合同 | 4个JSON结构匹配 wake_io_contract.yml |
| 6 | 飞书沙盘完整 | 5张CSV字段与 feishu_base_schema.yml 一致 |
| 7 | run_manifest连续 | 无间隔缺失的 simulated_date |
| 8 | 原始数据不变 | olist_orders_dataset.csv 行数不变 |

- 生成 `reports/local_loop_validation_report.md`，包含：
  ```markdown
  # 本地闭环验收报告
  ## 1. 连续运行概况
  ## 2. 各检查项结果
  ## 3. daily_metrics 行数变化
  ## 4. metric_alerts 触发统计
  ## 5. Wake 输出样例
  ## 6. 飞书沙盘验证
  ## 7. 已知问题
  ## 8. 下一步建议
  ```

**Must NOT do**:
- 不修改任何数据
- 不做任何分析/计算
- 不使用硬编码路径

**Recommended Agent Profile**:
- **Category**: `quick`
- **Skills**: `[]`

**Key references**:
- `config/wake_io_contract.yml` — Wake输出格式规范
- `config/feishu_base_schema.yml` — 飞书表字段规范
- `config/status_enums.yml` — 状态枚举
- `scripts/config.py` — 路径常量

**Acceptance Criteria**:
- [ ] `python3 scripts/validate_local_loop.py` 执行成功
- [ ] 生成完整的验收报告
- [ ] 报告包含所有8项检查结果
- [ ] 报告清晰标注 PASS/FAIL/WARN

---

- [ ] 4. 连续回放30天 + 验收

**What to do**:
```bash
# 先重置状态（从2016-09-04开始）
# 编辑 ingestion_state.json: next_simulated_date=2016-09-04, last_completed=null

# 连续回放30天
python3 scripts/replay_pipeline.py --days 30

# 执行验收
python3 scripts/validate_local_loop.py
```

**Acceptance Criteria**:
- [ ] `--days 30` 全部完成，success_rate >= 90%
- [ ] `validate_local_loop.py` 核心检查项全部 PASS
- [ ] `reports/local_loop_validation_report.md` 生成
- [ ] ingestion_state 推进到约 2016-10-04
- [ ] metric_alerts 在数据充足后开始触发

---

- [ ] 5. 输出验收摘要

**What to synthesize**:
- run_manifest.csv 摘要（成功/失败统计）
- daily_metrics.csv 前后几行
- metric_alerts.csv 是否触发
- aip_context_bundle_latest.json 结构摘要
- Wake 输出样例（1份日报 + 前3条建议）
- 飞书沙盘行数统计
- 验收判断：是否可进入真实飞书API阶段

---

## Success Criteria

### 验收命令
```bash
python3 scripts/run_daily_pipeline.py          # 单日管道
python3 scripts/replay_pipeline.py --days 30   # 连续30天
python3 scripts/validate_local_loop.py          # 自动验收
```

### Final Checklist
- [ ] `run_daily_pipeline.py` 可一键执行全部8步
- [ ] `replay_pipeline.py --days 30` 30天成功率 >= 90%
- [ ] `validate_local_loop.py` 生成验收报告
- [ ] daily_metrics 持续 as_of_date 过滤生效
- [ ] metric_alerts 在数据充足后正确触发
- [ ] Wake 4个输出格式稳定
- [ ] 飞书5张沙盘CSV字段完整
- [ ] 原始数据未被修改
- [ ] `reports/local_loop_validation_report.md` 包含完整验收结论

## Commit Strategy

- 单次提交：`feat(Phase H-0): 本地闭环验收 - 统一管道+连续回放+自动验收`
  - Files: scripts/run_daily_pipeline.py, scripts/replay_pipeline.py, scripts/validate_local_loop.py, reports/local_loop_validation_report.md
