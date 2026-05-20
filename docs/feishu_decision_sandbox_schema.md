# Phase 8: Feishu Decision Sandbox Schema Design
## Data Structure for Multi-dimensional Tables

**Design Date**: 2026-05-20
**Based On**: Phase 6 scenario design + Phase 7 v1.1 calibrated simulation results
**Target Platform**: Feishu Bitable (多维表格)

---

## Overview

This document defines the data structure for a Decision Sandbox to be implemented in Feishu Bitable. The sandbox supports what-if scenario simulation and result management for the Olist Brazilian e-commerce business analysis project.

Four core tables are designed:
1. **scenario_catalog** - Scenario registry and metadata
2. **scenario_parameters** - Scenario input parameters
3. **scenario_simulation_results** - Simulation output results
4. **simulation_validity_checks** - Validity verification records

---

## Table 1: scenario_catalog (场景目录表)

### Purpose
Central registry of all what-if scenarios, including metadata, status, and priority.

### Fields

| Field Name | Field Label (CN) | Data Type | Primary Key | Description | Filled By |
|-----------|-----------------|-----------|-------------|-------------|-----------|
| scenario_id | 场景ID | Text (PK) | ✅ | Unique scenario identifier (S01, S02, etc.) | System / Script |
| scenario_name | 场景英文名称 | Text | | Scenario name in English | Analyst |
| scenario_display_name | 场景显示名称 | Text | | Display name with Chinese translation for UI | Analyst |
| category | 场景分类 | Single Select | | Category: Fulfillment / Regional / Activation / etc. | Analyst |
| priority | 优先级 | Single Select | | HIGH / MEDIUM / LOW | Analyst |
| status | 状态 | Single Select | | ACTIVE / PAUSED / ARCHIVED | Waker / Analyst |
| created_by | 创建人 | Text | | Who created the scenario | System |
| created_at | 创建时间 | Date | | Scenario creation date | System |
| last_simulated_at | 最后模拟时间 | DateTime | | Last time simulation was run | Waker |
| description | 场景描述 | Text | | Detailed description of the scenario | Analyst |
| corresponding_script | 对应脚本 | Text | | Python script used for simulation | System |

### CSV Location
`data/processed/feishu_scenario_catalog.csv`

### Relationships
- **1:N** with `scenario_parameters` (one scenario has multiple parameters)
- **1:N** with `scenario_simulation_results` (one scenario has multiple results)
- **1:N** with `simulation_validity_checks` (one scenario has multiple checks)

---

## Table 2: scenario_parameters (场景参数表)

### Purpose
Store input parameters for each scenario, including baseline values, target values, and constraints.

### Fields

| Field Name | Field Label (CN) | Data Type | Primary Key | Description | Filled By |
|-----------|-----------------|-----------|-------------|-------------|-----------|
| parameter_id | 参数ID | Text (PK) | ✅ | Unique parameter identifier (P_S01_01, etc.) | System |
| scenario_id | 关联场景ID | Text (FK) | 🔗 | Links to scenario_catalog.scenario_id | System |
| input_field_label | 字段显示名 | Text | | Human-readable parameter name | Analyst |
| input_field_key | 字段标识符 | Text | | Machine-readable key for script reference | System |
| data_type | 数据类型 | Single Select | | number / integer / string | System |
| input_unit | 单位 | Text | | Unit of measurement (percentage, days, BRL, etc.) | Analyst |
| baseline_value | 基准值 | Number | | Current/baseline value of this parameter | Script (from data) |
| simulated_value | 模拟目标值 | Number | | Target value for the "what-if" scenario | Analyst / Waker |
| editable_by_waker | Waker可编辑 | Checkbox | | Whether Waker can modify this parameter | Analyst |
| default_value | 默认值 | Number | | Default value if user doesn't specify | Script |
| min_value | 最小值 | Number | | Minimum allowed value (validation) | Analyst |
| max_value | 最大值 | Number | | Maximum allowed value (validation) | Analyst |
| validation_rule | 验证规则 | Text | | Description of validation constraints | Analyst |
| param_description | 参数说明 | Text | | Detailed description of this parameter | Analyst |

### CSV Location
`data/processed/feishu_scenario_parameters.csv`

### Relationships
- **N:1** with `scenario_catalog` (multiple parameters per scenario)

### Validation Rules
```
For each parameter:
  1. baseline_value and simulated_value must be within [min_value, max_value]
  2. If editable_by_waker = false, Waker cannot modify this field
  3. If editable_by_waker = true, Waker must validate against min/max before writing
```

---

## Table 3: scenario_simulation_results (模拟结果表)

### Purpose
Store simulation outputs including baseline, simulated values, changes, and evidence levels.

### Fields

| Field Name | Field Label (CN) | Data Type | Primary Key | Description | Filled By |
|-----------|-----------------|-----------|-------------|-------------|-----------|
| result_id | 结果ID | Text (PK) | ✅ | Unique result identifier (R_S01_01, etc.) | System |
| scenario_id | 关联场景ID | Text (FK) | 🔗 | Links to scenario_catalog.scenario_id | System |
| metric_key | 指标标识符 | Text | | Machine-readable metric name | System |
| metric_display_name | 指标显示名 | Text | | Human-readable metric name | Analyst |
| data_type | 数据类型 | Single Select | | number / percentage / BRL | System |
| metric_unit | 单位 | Text | | Unit of measurement | Analyst |
| baseline_value | 基准值 | Number | | Current metric value from historical data | Script |
| simulated_value | 模拟值 | Number | | Simulated metric value after scenario input | Script |
| raw_simulated_value | 原始模拟值 | Number | | Uncalibrated raw simulation output (may exceed bounds) | Script |
| change_value | 变化值 | Number | | simulated_value - baseline_value | Script |
| change_pct | 变化百分比 | Number | | Percentage change (%) | Script |
| change_direction | 变化方向 | Single Select | | positive / negative / neutral | Script |
| evidence_level | 证据级别 | Single Select | | ARITHMETIC / OBSERVATIONAL / EXPERIENTIAL | Script |
| confidence | 置信度 | Single Select | | HIGH / MEDIUM / LOW | Script |
| interpretation | 结果解读 | Text | | Human-readable explanation of the result | Script |
| causal_disclaimer | 因果声明 | Text | | Disclaimer about correlation vs causation | System |
| calculation_notes | 计算说明 | Text | | Technical notes on calculation method | Script |

### CSV Location
`data/processed/feishu_simulation_results.csv`

### Relationships
- **N:1** with `scenario_catalog` (multiple results per scenario)
- **1:N** with `simulation_validity_checks` (each result has corresponding checks)

### Evidence Level Definitions
| Level | CN | Description |
|-------|-----|----------------------------------------|
| ARITHMETIC_CALCULATION | 纯算术计算 | Direct calculation from known data |
| OBSERVATIONAL_CORRELATION | 观测相关性 | Based on historical statistical correlations |
| EXPERIENTIAL_ESTIMATE | 经验估算 | Based on domain expertise and approximations |

---

## Table 4: simulation_validity_checks (模拟有效性检查表)

### Purpose
Track validity verification for each simulation result, including boundary checks, extrapolation risks, and denominator clarity.

### Fields

| Field Name | Field Label (CN) | Data Type | Primary Key | Description | Filled By |
|-----------|-----------------|-----------|-------------|-------------|-----------|
| validity_id | 检查ID | Text (PK) | ✅ | Unique validity check identifier | System |
| scenario_id | 关联场景ID | Text (FK) | 🔗 | Links to scenario_catalog.scenario_id | System |
| metric_key | 关联指标 | Text | | Links to scenario_simulation_results.metric_key | System |
| check_type | 检查类型 | Single Select | | BOUNDED / EXTRAPOLATION / DENOMINATOR | System |
| check_description | 检查描述 | Text | | Description of what is being checked | System |
| check_result | 检查结果 | Single Select | | PASS / WARNING | Script |
| value_or_status | 检查值/状态 | Text | | Specific value or status message | System |
| risk_level | 风险等级 | Single Select | | LOW / MEDIUM / HIGH | Script |
| explanation | 说明 | Text | | Detailed explanation of the check result | Script |
| requires_disclaimer | 需要免责声明 | Checkbox | | Whether a disclaimer must be shown to users | Script |
| disclaimer_text | 免责声明文本 | Text | | Displayed to users when disclaimer is required | Script |

### CSV Location
`data/processed/feishu_validity_checks.csv`

### Relationships
- **N:1** with `scenario_catalog`
- **N:1** with `scenario_simulation_results` (on scenario_id + metric_key)

### Check Type Definitions
| Type | Description |
|------|-------------|
| BOUNDED | Checks if simulated value is within valid range (e.g., score 1-5) |
| EXTRAPOLATION | Checks if result relies on extrapolating historical correlations |
| DENOMINATOR | Checks if there is risk of denominator ambiguity in interpretation |

---

## Data Flow Diagram

```
┌─────────────────────┐
│  Analyst / User     │
│  (Feishu UI)        │
└─────────┬───────────┘
          │ Sets target values
          ▼
┌─────────────────────┐     ┌─────────────────────┐
│ scenario_parameters │────▶│ scenario_simulation  │
│ (editable params)   │     │   results table      │
└─────────────────────┘     └──────────┬──────────┘
                                       │
                                       ▼
                              ┌─────────────────────┐
                              │ simulation_validity  │
                              │    checks table      │
                              └─────────────────────┘

Waker reads from:    scenario_catalog, scenario_parameters
Waker writes to:     scenario_simulation_results (via simulation script)
Waker validates:     simulation_validity_checks (auto-generated)
```

---

## Feishu Bitable Configuration Recommendations

### Field Types Mapping

| CSV Type | Feishu Bitable Field Type |
|----------|--------------------------|
| Text | Text |
| Number | Number |
| Single Select | Select |
| Checkbox | Checkbox |
| Date | Date |
| DateTime | Date (with time) |

### Recommended Views

1. **Scenario Overview View**: scenario_catalog (all scenarios, grouped by category)
2. **Parameter Input View**: scenario_parameters (editable fields for Waker)
3. **Results Dashboard View**: scenario_simulation_results (grouped by scenario_id)
4. **Validity Check View**: simulation_validity_checks (filtered by requires_disclaimer = true)

### Automation Rules

1. When Waker updates a parameter in `scenario_parameters`:
   - Re-run simulation script
   - Update `scenario_simulation_results`
   - Re-evaluate `simulation_validity_checks`
   - Update `scenario_catalog.last_simulated_at`

2. When `check_result` in `simulation_validity_checks` is WARNING:
   - Display `disclaimer_text` to users
   - Flag scenario for manual review

---

## CSV Files for Import

| Table Name | CSV File Path | Rows |
|-----------|---------------|------|
| scenario_catalog | `data/processed/feishu_scenario_catalog.csv` | 4 |
| scenario_parameters | `data/processed/feishu_scenario_parameters.csv` | 12 |
| scenario_simulation_results | `data/processed/feishu_simulation_results.csv` | 10 |
| simulation_validity_checks | `data/processed/feishu_validity_checks.csv` | 12 |

**Total**: 4 tables, 38 rows, ready for direct Feishu import.

---

## Reproducibility

All CSV files generated from:
- Phase 6 scenario design (`outputs/tables/scenario_design_catalog.csv`)
- Phase 7 v1.1 calibrated results (`outputs/tables/scenario_simulation_results.csv`)
- This schema documentation: `docs/feishu_decision_sandbox_schema.md`

**Design Date**: 2026-05-20
