# Phase 10: Waker 飞书决策沙盘只读验证 - 决策解释报告

**执行时间**: 2026-05-21
**Base**: [Olist经营模拟决策沙盘-Phase9](https://qcnwqbyu8d94.feishu.cn/base/P0m4bfs7da9Relsu2pKcvt6YnLb)
**Base Token**: P0m4bfs7da9Relsu2pKcvt6YnLb
**访问方式**: `lark-cli base +record-list` (只读，零写入)
**验证状态**: ✅ **SUCCESS** — 四张表全量只读读取通过，所有校验项 PASS

---

## 读取操作记录（只读）

| 操作 | 表 | 命令 | 结果 |
|------|-----|------|------|
| 只读读取 | scenario_catalog | `+record-list --table-id tblpu4GcT3vman1C` | 4 条记录 |
| 只读读取 | scenario_parameters | `+record-list --table-id tbl89egfWEDzfJU2` | 12 条记录 |
| 只读读取 | scenario_simulation_results | `+record-list --table-id tblOcwGZt2DYJEvc` | 10 条记录 |
| 只读读取 | simulation_validity_checks | `+record-list --table-id tblT5iDLjF1UVeEp` | 12 条记录 |

**所有操作均为 read-only，无任何写入操作。**

---

## S02 场景解释：配送时长缩短

### 场景概览

| 属性 | 值 |
|------|-----|
| 场景 ID | S02 |
| 场景名称 | Delivery Time Reduction |
| 显示名称 | S02 配送时长缩短 |
| 分类 | Fulfillment |
| 优先级 | HIGH |
| 状态 | ACTIVE |
| 执行脚本 | phase7_calibration_revision.py |

### 输入参数（从 scenario_parameters 读取）

| 参数 | 基准值 | 模拟值 | 单位 | Waker可编辑 |
|------|--------|--------|------|------------|
| Avg Delivery Days (Current) | 12.56 | 12.56 | days | true |
| Avg Delivery Days (Target) | 12.56 | 10.0 | days | true |
| Conservative Cap | 0.3 | 0.3 | points | true |

**操作描述**: 将平均配送时长从 **12.56 天**缩短到 **10.0 天**（-20.4%）。

### 模拟输出（从 scenario_simulation_results 读取）

| 指标 | 基准值 | 原始模拟值 | 校准后值 | 变化 | 证据等级 |
|------|--------|-----------|---------|------|---------|
| Review Score (Mean) | 4.16 | **5.015** | **4.460** | +0.300 (+7.21%) | OBSERVATIONAL_CORRELATION |
| Low Score Percentage | 12.71 | 10.88 | 10.88 | -1.83pp | OBSERVATIONAL_CORRELATION |
| Cancellation Risk Factor | 1.019 | 0.661 | 0.661 | -0.358 (-35.1%) | OBSERVATIONAL_CORRELATION |

### 校准原因

**S02 的评分模拟结果经过了校准处理**：

1. **原始外推**：基于配送时长与评分的相关系数 r = -0.334，将配送时长减少 2.56 天外推得到评分 = 5.015
2. **边界违规**：5.015 超过评分上限 5.0，需要截断
3. **保守调整**：应用 +0.3 保守上限（针对 HIGH 置信度的相关性推演），最终校准值为 **4.460**

### 证据等级

**OBSERVATIONAL_CORRELATION (历史相关性推演)**

- 基于 Phase 4 观测到的历史数据中的相关性
- 配送时长与评分相关系数：-0.334（负相关：配送时间越长，评分越低）
- 置信度：HIGH

### ⚠️ 非严格因果声明

> **基于历史相关性推演，不代表真实因果关系。**
>
> 原始估算 5.015 已截断至 4.460。保守上限 0.3 分已应用。
>
> 实际业务中，缩短配送时长可能因实施质量、混杂变量（如卖家处理速度、物流基础设施差异）而产生不同的效果。相关性不等同于因果性。

### 有效性检查（从 simulation_validity_checks 读取）

| 检查ID | 检查类型 | 结果 | 风险等级 | 说明 |
|--------|----------|------|---------|------|
| V_S02_01 | BOUNDED | PASS | LOW | 4.460 在有效范围 [1.0, 5.0] 内 |
| V_S02_02 | EXTRAPOLATION | WARNING | HIGH | 使用最强相关性 (-0.334) 进行外推 |
| V_S02_03 | DENOMINATOR | PASS | LOW | 评分指标自身完整，无分母问题 |

---

## S08 场景解释：商家激活率提升

### 场景概览

| 属性 | 值 |
|------|-----|
| 场景 ID | S08 |
| 场景名称 | Seller Activation Rate Improvement |
| 显示名称 | S08 商家激活率提升 |
| 分类 | Activation |
| 优先级 | HIGH |
| 状态 | ACTIVE |
| 执行脚本 | phase7_calibration_revision.py |

### 输入参数（从 scenario_parameters 读取）

| 参数 | 基准值 | 模拟值 | 单位 | Waker可编辑 |
|------|--------|--------|------|------------|
| Current Activation Rate | 45.1 | 45.1 | percentage | true |
| Target Activation Rate | 45.1 | 55.0 | percentage | true |
| Total Closed Sellers | 842 | 842 | count | false |
| Avg GMV per Active Seller | 2041.62 | 2041.62 | BRL | true |

**操作描述**: 将商家激活率从 **45.1%** 提升到 **55.0%**（+9.9 个百分点）。

**计算过程**:
- 额外激活商家数 = 842 × 55% - 380 = **83 家**
- 新增 GMV = 83 × R$2,041.62 = **R$169,454**

### 模拟输出（从 scenario_simulation_results 读取）

| 指标 | 基准值 | 模拟值 | 变化 | 证据等级 |
|------|--------|--------|------|---------|
| Seller Activation Rate | 45.1 | 55.0 | +9.9pp | ARITHMETIC_CALCULATION |
| Closed Seller GMV | 775,815.63 | 945,270.09 | +R$169,454.46 | ARITHMETIC_CALCULATION |
| As % of Platform GMV | 5.78 | 7.05 | +1.26pp | ARITHMETIC_CALCULATION |

### 分母口径说明

**S08 同时保留两种 GMV 口径**：

| 口径 | 基准 | 模拟 | 增幅 |
|------|------|------|------|
| 漏斗成交商家 GMV | R$775,816 | R$945,270 | **+21.8%** |
| 全平台 GMV 占比 | 5.78% | 7.05% | **+1.26pp** |

**关键区别**：
- **21.8%** 是相对于"漏斗成交商家 GMV"的增幅（局部口径，R$775,816 → R$945,270）
- **+1.26pp** 是相对于"全平台总 GMV"的占比增幅（全局口径，5.78% → 7.05%）

> ⚠️ **分母误解风险（HIGH）**：R$169,454 仅为漏斗成交商家 GMV 提升的 21.8%，占全平台 GMV 的 1.26%。在汇报和决策时，必须明确使用哪种口径，否则可能严重高估或低估场景的实际影响。

### 证据等级

**ARITHMETIC_CALCULATION (直接算术计算)**

- 基于已知商家数量和平均 GMV 进行确定性计算
- 不涉及任何相关性推演或假设
- 置信度：HIGH

### 有效性检查（从 simulation_validity_checks 读取）

| 检查ID | 检查类型 | 结果 | 风险等级 | 说明 |
|--------|----------|------|---------|------|
| V_S08_01 | BOUNDED | PASS | LOW | 55% 激活率在有效范围 [0%, 100%] 内 |
| V_S08_02 | EXTRAPOLATION | PASS | LOW | 纯算术计算，无相关性外推 |
| V_S08_03 | DENOMINATOR | WARNING | HIGH | 必须区分漏斗商家 GMV vs 全平台 GMV |

---

## 数据验证汇总

### 记录数校验

| 表 | 预期记录数 | 实际读取数 | 状态 |
|----|-----------|-----------|------|
| scenario_catalog | 4 | 4 | PASS |
| scenario_parameters | 12 | 12 | PASS |
| scenario_simulation_results | 10 | 10 | PASS |
| simulation_validity_checks | 12 | 12 | PASS |

### 关键字段校验

| 校验项 | 期望值 | 实际读取值 | 状态 |
|--------|--------|-----------|------|
| S02 raw_simulated_value (R_S02_01) | 5.015 | 5.015 | PASS |
| S02 simulated_value (R_S02_01) | 4.460 | 4.46 | PASS |
| S08 漏斗GMV (R_S08_02) | 存在 | 945270.09 BRL | PASS |
| S08 全平台占比 (R_S08_03) | 存在 | 7.05% | PASS |
| S02 证据等级 | OBSERVATIONAL_CORRELATION | OBSERVATIONAL_CORRELATION | PASS |
| S08 证据等级 | ARITHMETIC_CALCULATION | ARITHMETIC_CALCULATION | PASS |
| S02 免责声明 | 存在 | 有 | PASS |
| S08 免责声明 | 存在 | 有 | PASS |

### 跨表一致性校验

| 场景 | scenario_catalog | scenario_parameters | scenario_simulation_results | simulation_validity_checks | 状态 |
|------|-----------------|---------------------|---------------------------|---------------------------|------|
| S01 | 1 | 2 | 2 | 3 | PASS |
| S02 | 1 | 3 | 3 | 3 | PASS |
| S07 | 1 | 3 | 2 | 3 | PASS |
| S08 | 1 | 4 | 3 | 3 | PASS |

---

## 最终状态

**✅ SUCCESS** — Phase 10 只读验证完成。

- 四张表全部可访问，记录数与预期完全一致（4/12/10/12 = 38 条）
- S02 原始模拟值 5.015 和校准值 4.460 均正确读取
- S08 同时保留了漏斗 GMV 口径与全平台 GMV 占比口径
- 所有证据等级、免责声明、有效性检查均完整
- 所有操作为 read-only，无任何写入
