# Baxi E2E 测试情景

## 概览

`baxi-cli e2e` 提供了一套自适应的端到端测试流程，根据实际数据库状态动态调整执行路径，覆盖告警、决策、治理、本体等核心域。

---

## 测试场景矩阵

### 场景 1：完整决策闭环（Happy Path）

**前提条件**：系统有活跃告警，LLM 服务可用

**执行步骤**：
```
1. 获取系统状态 → 确认数据库连接正常
2. 列出活跃告警 → 发现 seller_revenue_drop / unit_cost_anomaly 告警
3. 为告警创建决策案例 → case_id: dc_1780502896_MQeHqV
4. 获取案例详情 → 验证案例字段完整
5. 列出所有案例 → 确认案例出现在列表中
6. 构建 LLM 决策上下文 → 触发 info + 对象上下文 + 治理信息
7. 调用 decide → 生成 action_proposal（如 notify_owner）
8. 获取提案详情 → 验证 proposal_id 和推荐动作
9. 审批提案 → 状态变为 approved，生成 review_record
10. 执行提案（dry_run） → 返回模拟执行结果
11. 关闭案例 → 状态变为 closed
```

**预期结果**：全部步骤 PASS，形成完整闭环

---

### 场景 2：缺失告警时的数据准备路径

**前提条件**：系统无活跃告警，ontology 中有 seller 数据

**执行步骤**：
```
1. 获取系统状态 → alerts=0
2. 搜索 ontology 对象 → 找到 112,652 个 seller
3. 从真实数据创建模拟告警 → alert_id: st-alert-xxx
4. 用模拟告警创建案例 → 绑定到真实 seller 对象
5. 继续完整决策闭环
```

**自适应逻辑**：当 detectAlerts=0 时，自动从 ontology 数据生成测试告警

---

### 场景 3：Schema 缺失自动修复

**前提条件**：global 对象表缺少 baseline_value / snapshot_date 列

**执行步骤**：
```bash
go run ./cmd/baxi-cli e2e --auto-fix
```

```
1. 检查 ontology 定义 → 发现 global 类型缺少列
2. 自动执行 ALTER TABLE 添加缺失列
3. 重新验证 schema → PASS
4. 继续后续测试
```

**涉及修复**：
- `dwd.dim_global_daily`: +baseline_value, +snapshot_date
- `dwd.dim_seller_v2`: +order_level
- `ops.event_outbox`: 创建缺失表（如需要）

---

### 场景 4：LLM 决策失败回退

**前提条件**：LLM 服务不可用或返回无效决策

**执行步骤**：
```
1. 创建案例 → 成功
2. 构建上下文 → 成功
3. 调用 decide → FAIL（LLM error / validation error）
4. 回退：手动创建提案 → INSERT action_proposal
5. 继续审批 → 执行 → 结案
```

**预期结果**：decide 步骤 FAIL，但整体流程通过回退机制完成

---

### 场景 5：沙箱验证流程

**执行步骤**：
```
1. 创建沙箱 → sandbox_id: sbx_xxx
2. 将已审批提案加入沙箱 → 验证关联关系
3. 获取沙箱详情 → 确认包含提案
4. 对比沙箱与原始提案 → 验证一致性
```

**验证点**：sandbox_create, sandbox_add, sandbox_get 三个工具链

---

### 场景 6：治理数据验证

**执行步骤**：
```
1. 获取数据分类 → classification rules 返回
2. 获取治理状态 → governance_status 返回
3. 获取访问策略 → access policies 返回
4. 检查数据标记 → markings 返回
```

**验证点**： governance 配置是否正确加载到数据库

---

### 场景 7：Pipeline 数据流验证

**执行步骤**：
```
1. 检查 raw 层数据 → raw.listings / raw.orders 有数据
2. 运行 pipeline validate → 配置校验通过
3. 检查 mart 层指标 → mart.seller_daily_metrics 有聚合数据
4. 验证告警触发条件 → 指标异常能生成告警
```

**注意**：当前需要手动放置 CSV 文件到 data/ 目录以触发 pipeline

---

### 场景 8：Outbox 事件流验证

**执行步骤**：
```
1. 创建提案并审批 → 生成 outbox 事件
2. 列出 outbox 事件 → 发现 pending 事件
3. 验证事件格式 → 包含 channel_type, payload, retry_count
```

**已知限制**：当前无自动 dispatch worker，事件停留在 pending 状态

---

### 场景 9：安全边界测试

**执行步骤**：
```bash
# 尝试 live 执行（默认拒绝）
go run ./cmd/baxi-cli e2e
# → baxi_execute_proposal(live) = SKIP（安全保护）

# 显式开启 live
go run ./cmd/baxi-cli e2e --live
# → 需要 BAXI_ALLOW_LIVE_EXECUTION=true，否则 WARN
```

**验证点**：未设置环境变量时，live 执行被正确拦截

---

### 场景 10：多案例并发处理

**执行步骤**：
```
1. 为告警 A 创建案例 → case_1
2. 为告警 B 创建案例 → case_2
3. 列出所有案例 → 确认 2+ 个案例
4. 为 case_1 创建提案 → proposal_1
5. 为 case_2 创建提案 → proposal_2
6. 分别审批并执行
7. 分别结案
```

**验证点**：多案例隔离，proposal 正确关联到对应 case

---

## 命令行选项

| 选项 | 说明 | 场景 |
|------|------|------|
| `--auto-fix` | 自动修复 schema 缺失 | 场景 3 |
| `--live` | 允许真实执行（需环境变量） | 场景 9 |
| `--json` | JSON 格式输出结果 | 所有场景 |
| `--verbose` | 详细日志 | 调试所有场景 |

---

## 已知限制与绕过方案

| 问题 | 影响 | 绕过 |
|------|------|------|
| global 对象未注册 ontology | baxi_decide 失败 | 使用 --auto-fix 或手动注册 |
| ops.event_outbox 表缺失 | outbox 测试失败 | 手动创建表或跳过 |
| pipeline 无 CSV 数据 | search_objects 无结果 | 放置测试 CSV 到 data/ 目录 |
| live 执行需环境变量 | 真实执行被拦截 | 设置 BAXI_ALLOW_LIVE_EXECUTION=true |

---

## 运行示例

```bash
# 完整测试（推荐）
go run ./cmd/baxi-cli e2e --auto-fix --verbose

# CI 环境（JSON 输出）
go run ./cmd/baxi-cli e2e --json

# 安全测试（验证门禁）
go run ./cmd/baxi-cli e2e --live
```
