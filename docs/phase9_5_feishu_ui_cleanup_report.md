# Phase 9.5: 飞书 Base 结构清理与验收记录更新报告

**执行时间**: 2026-05-21  
**Base**: [Olist经营模拟决策沙盘-Phase9](https://qcnwqbyu8d94.feishu.cn/base/P0m4bfs7da9Relsu2pKcvt6YnLb)  
**Base Token**: P0m4bfs7da9Relsu2pKcvt6YnLb  

---

## 1. 默认空表删除

| 操作 | 状态 | 说明 |
|------|------|------|
| 删除"数据表" | ✅ 成功 | 使用 `lark-cli base +table-delete --table-id tblitoXyXMDQ4zLr --yes` 删除成功 |

**清理后表清单**（仅保留4张业务表）:
1. scenario_catalog (tblpu4GcT3vman1C) - 4条记录
2. scenario_parameters (tbl89egfWEDzfJU2) - 12条记录
3. scenario_simulation_results (tblOcwGZt2DYJEvc) - 10条记录
4. simulation_validity_checks (tblT5iDLjF1UVeEp) - 12条记录

## 2. 字段类型转换（文本 → 单选）

### 尝试转换的字段清单

| 表 | 字段 | 当前类型 | 目标类型 | 结果 |
|----|------|---------|---------|------|
| scenario_catalog | category | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_catalog | priority | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_catalog | status | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_catalog | created_by | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_parameters | data_type | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_parameters | input_unit | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_simulation_results | data_type | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| scenario_simulation_results | metric_unit | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| simulation_validity_checks | check_type | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| simulation_validity_checks | check_result | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |
| simulation_validity_checks | risk_level | text | single_select | ❌ CLI 不支持，需人工在 UI 中完成 |

### 技术说明

飞书 CLI (`lark-cli`) 的 `+field-update` 命令对 `single_select` 类型转换存在格式限制。尝试了以下格式均失败:
- 使用 `property.options` 结构 → `Unrecognized key(s) in object: 'property'`
- 使用直接 `options` 数组 → `Request validation failed with 3 issues`

**结论**: 这些字段类型转换需要通过飞书 Web UI 手动完成，CLI 目前无法实现此操作。

### 人工操作指引

在飞书界面中，建议将以下字段改为单选型并配置对应选项:

**scenario_catalog**:
- `category`: 选项 → Fulfillment, Regional, Activation, Marketing, Category, Seller Quality, Retention
- `priority`: 选项 → HIGH, MEDIUM, LOW
- `status`: 选项 → ACTIVE, PAUSED, ARCHIVED
- `created_by`: 选项 → analyst, system, auto

**scenario_parameters**:
- `data_type`: 选项 → number, integer, string
- `input_unit`: 选项 → percentage, days, points, BRL, count

**scenario_simulation_results**:
- `data_type`: 选项 → number, percentage, BRL
- `metric_unit`: 选项 → points, percentage, BRL, index

**simulation_validity_checks**:
- `check_type`: 选项 → BOUNDED, EXTRAPOLATION, DENOMINATOR
- `check_result`: 选项 → PASS, WARNING
- `risk_level`: 选项 → LOW, MEDIUM, HIGH

## 3. scenario_id 字段保护

✅ 确认 `scenario_id` 在所有表中保持为文本字段类型，未进行任何类型转换。

## 4. 数据完整性校验

| 校验项 | 预期值 | 实际值 | 状态 |
|--------|--------|--------|------|
| scenario_catalog 记录数 | 4 | 4 | ✅ 通过 |
| scenario_parameters 记录数 | 12 | 12 | ✅ 通过 |
| scenario_simulation_results 记录数 | 10 | 10 | ✅ 通过 |
| simulation_validity_checks 记录数 | 12 | 12 | ✅ 通过 |
| S02 simulated_value | 4.460 | 4.46 | ✅ 通过（显示精度差异，值一致） |
| S02 raw_simulated_value | 5.015 | 5.015 | ✅ 通过 |
| S08 成交商家GMV (+21.8%) | 存在 | 存在 | ✅ 通过 |
| S08 全平台GMV (+1.26%) | 存在 | 存在 | ✅ 通过 |

## 5. 总结

| 操作 | 状态 | 备注 |
|------|------|------|
| 删除默认空表 | ✅ 完成 | 通过 CLI 执行 |
| 字段类型转换 | ⚠️ 未执行 | CLI 不支持，需飞书 UI 手动操作 |
| scenario_id 保护 | ✅ 确认 | 保持文本类型 |
| 数据完整性 | ✅ 全部通过 | 4/12/10/12 记录数一致，关键值未变 |

---

**最终状态**: ✅ 清理完成（字段类型转换列为待人工优化项）
**需要第二天人工处理**: 在飞书 UI 中将 10 个文本字段改为单选型
