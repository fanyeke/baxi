# Phase 8: Waker Read/Write Contract
## Waker Agent Interaction Protocol for Feishu Decision Sandbox

**Contract Date**: 2026-05-20
**Version**: 1.0
**Scope**: Waker agent interaction with Feishu Bitable tables

---

## 1. Overview

This document defines how the Waker agent should read from and write to the Feishu Decision Sandbox tables. Waker is responsible for:
- Reading scenario parameters
- Running simulations
- Writing results back to the sandbox
- Checking validity and raising disclaimers when needed

---

## 2. Tables Waker Should Read

### 2.1 scenario_catalog (场景目录表)

**Purpose**: List all available scenarios.

**Read Columns**:
- `scenario_id` - Primary identifier
- `scenario_name` - Scene name
- `status` - Filter for ACTIVE scenarios only
- `category` - Group scenarios by type
- `description` - Understand what each scenario does

**Usage**:
```python
# Fetch all active scenarios where Waker can run simulations
active_scenarios = feishu_read("scenario_catalog", filter={"status": "ACTIVE"})
# Result: S01, S02, S07, S08 (4 scenarios)
```

**⚠️ Important**: Only process scenarios where `status == "ACTIVE"`. Ignore PAUSED or ARCHIVED scenarios.

---

### 2.2 scenario_parameters (场景参数表)

**Purpose**: Read input parameters for simulation.

**Read Columns**:
- `parameter_id` - Unique parameter identifier
- `scenario_id` - Foreign key to scenario_catalog
- `input_field_key` - Machine-readable key
- `baseline_value` - Current baseline from data
- `simulated_value` - User-defined target (what-if value)
- `editable_by_waker` - Whether Waker should modify this
- `data_type` / `input_unit` - For display and validation
- `min_value` / `max_value` - Validation constraints

**Usage**:
```python
# Fetch all parameters for a specific scenario
params = feishu_read("scenario_parameters", filter={"scenario_id": "S02"})
# For each param:
# - If editable_by_waker == False: baseline is system data; do NOT modify
# - If editable_by_waker == True: simulated_value is user input; use for simulation
```

**⚠️ Important**: 
- `editable_by_waker == False` means the parameter is **system-generated from analysis data**. Waker should NOT modify these.
- `editable_by_waker == True` means the parameter can be **adjusted by user/Waker** as the "what-if" input.

---

## 3. How Waker Locates Scenarios

### 3.1 Scenario Lookup Flow

```
1. User requests simulation of scenario → Get scenario_id (e.g., "S02")
2. Waker looks up scenario_id in scenario_catalog
3. Fetch all parameters from scenario_parameters where scenario_id = "S02"
4. Extract baseline_values and simulated_values
5. Run simulation with these parameters
6. Write results to scenario_simulation_results
7. Run validity checks and write to simulation_validity_checks
```

### 3.2 Parameter Resolution

For each parameter, Waker should:

1. **Identify editable parameters**: `editable_by_waker == true`
   - These represent the "what-if" inputs (e.g., target delivery days, target activation rate)
   - Use `simulated_value` as the target

2. **Identify read-only parameters**: `editable_by_waker == false`
   - These are system-derived from historical analysis (e.g., total closed sellers, avg GMV per seller)
   - Use `baseline_value` as the known constant

3. **Validate all inputs**:
   ```
   For each parameter:
     If simulated_value is not None:
       If simulated_value < min_value or simulated_value > max_value:
         Raise error: "Parameter {input_field_label} value {simulated_value} outside valid range [{min_value}, {max_value}]"
   ```

---

## 4. How to Read Input Parameters

### 4.1 Parameter Input Mapping (S02 Example)

For **S02: Delivery Time Reduction**, Waker should read:

| parameter_id | input_field_key | editable_by_waker | Value to Use | Explanation |
|-------------|-----------------|-------------------|-------------|-------------|
| P_S02_01 | delivery_days_current | true | baseline_value = 12.56 | Current delivery time (user can update) |
| P_S02_02 | delivery_days_target | true | simulated_value = 10.0 | Target delivery time (what-if input) |
| P_S02_03 | conservative_cap | true | simulated_value = 0.3 | Max allowed score increase (calibration) |

**Delta Calculation**: `delta = simulated_value - baseline_value = 10.0 - 12.56 = -2.56 days`

### 4.2 Parameter Input Mapping (S08 Example)

For **S08: Seller Activation Rate Improvement**, Waker should read:

| parameter_id | input_field_key | editable_by_waker | Value to Use | Explanation |
|-------------|-----------------|-------------------|-------------|-------------|
| P_S08_01 | activation_rate_current | true | baseline_value = 45.1 | Current activation rate |
| P_S08_02 | activation_rate_target | true | simulated_value = 55.0 | Target activation rate |
| P_S08_03 | total_closed_sellers | false | baseline_value = 842 | System data; do not modify |
| P_S08_04 | avg_gmv_per_seller | true | baseline_value = 2041.62 | Can be adjusted if market conditions change |

**Calculation**:
```
additional_sellers = (842 * 55/100) - (842 * 45.1/100) = 83
additional_gmv = 83 * 2041.62 = R$169,454
```

---

## 5. How to Interpret Simulation Results

### 5.1 Reading Results from scenario_simulation_results

After running a simulation, results are written to `scenario_simulation_results`. Waker should interpret them as:

| Field | How to Use |
|-------|-----------|
| `simulated_value` | The main result to display to users |
| `raw_simulated_value` | Shows raw (uncalibrated) estimate; compare with calibrated to see adjustments |
| `change_value` | Absolute difference from baseline |
| `change_pct` | Percentage change |
| `evidence_level` | **Critical**: Determines confidence in the result |
| `causal_disclaimer` | **Critical**: Must display this text if `causal_disclaimer` is not empty |

### 5.2 Evidence Level Interpretation

| evidence_level | Interpretation | Waker Action |
|---------------|----------------|-------------|
| **ARITHMETIC_CALCULATION** | Direct calculation from known data | **No disclaimer required**; result is mathematically certain given the inputs |
| **OBSERVATIONAL_CORRELATION** | Based on historical statistical correlations | **⚠️ MUST display causal_disclaimer**; correlation ≠ causation |
| **EXPERIENTIAL_ESTIMATE** | Based on domain expertise and approximations | **⚠️ MUST display causal_disclaimer**; estimate, not measurement |

### 5.3 Example Result Interpretation (S02)

```json
{
  "result_id": "R_S02_01",
  "scenario_id": "S02",
  "metric_key": "review_score",
  "baseline_value": 4.16,
  "simulated_value": 4.460,
  "raw_simulated_value": 5.015,
  "change_value": 0.3000,
  "evidence_level": "OBSERVATIONAL_CORRELATION",
  "causal_disclaimer": "⚠️ 基于历史相关性推演，不代表真实因果关系。原始估算5.015已截断至4.460。"
}
```

**Waker should display**:
```
📊 S02 Simulation Result - Review Score

Baseline: 4.16
Simulated: 4.460 (+0.300 points)
⚠️ Based on historical correlation; raw estimate was 5.015 (calibrated to 4.460)

⚠️ Disclaimer: 基于历史相关性推演，不代表真实因果关系。原始估算5.015已截断至4.460。保守上限0.3分已应用。
```

---

## 6. Results That MUST Display "不代表严格因果" Disclaimer

The following results represent **correlation-based or estimated** outcomes and MUST include a causal disclaimer when displayed:

### 6.1 Disclaimer Rules

| Scenario | Metric | Evidence Level | Requires Disclaimer? | Why |
|----------|--------|---------------|---------------------|------|
| **S01** | review_score | OBSERVATIONAL_CORRELATION | ✅ YES | Uses Phase 4 delay_score correlation |
| **S01** | low_score_pct | OBSERVATIONAL_CORRELATION | ✅ YES | Derived from score correlation |
| **S02** | review_score | OBSERVATIONAL_CORRELATION | ✅ YES | Uses Phase 4 delivery_score correlation (-0.334); calibrated |
| **S02** | low_score_pct | OBSERVATIONAL_CORRELATION | ✅ YES | Derived from score correlation |
| **S02** | cancellation_risk_factor | OBSERVATIONAL_CORRELATION | ✅ YES | Based on experience estimate |
| **S07** | overall_score_impact | EXPERIENTIAL_ESTIMATE | ✅ YES | Uses approximation, not hard data |
| **S08** | seller_activation_rate | ARITHMETIC_CALCULATION | ❌ NO | Pure arithmetic; no correlation |
| **S08** | closed_seller_GMV | ARITHMETIC_CALCULATION | ❌ NO | Pure arithmetic; but needs denominator clarification |
| **S08** | platform_GMV_pct | ARITHMETIC_CALCULATION | ❌ NO | Pure arithmetic; but must clarify denominator |

### 6.2 Disclaimer Text to Use

For each result row, use the **exact** text from the `causal_disclaimer` column in `feishu_simulation_results.csv`.

**Generic disclaimer template** (if `causal_disclaimer` is empty but evidence is not ARITHMETIC):
```
⚠️ 此结果基于历史观测相关性或经验估算推演，不代表严格因果关系。
实际业务中的干预效果可能因混杂变量、非线性关系或实施质量而与模拟结果不同。
```

### 6.3 Special Note for S08 (Denominator Clarification)

S08 results are ARITHMETIC (no correlation), but have a **high denominator risk**. Waker must always clarify:

```
💡 注意：R$169,454 是「营销漏斗成交商家」的GMV提升（占这部分GMV的21.8%），
而非全平台GMV提升（仅占全平台GMV的1.26%）。请根据具体上下文选择正确的分母口径。
```

---

## 7. Waker Write Operations

### 7.1 Writing to scenario_simulation_results

After running a simulation, Waker writes results to `scenario_simulation_results`:

```python
for metric in simulation.metrics:
    feishu_write("scenario_simulation_results", {
        "result_id": f"R_{scenario_id}_{metric_index}",
        "scenario_id": scenario_id,
        "metric_key": metric.key,
        "baseline_value": metric.baseline,
        "simulated_value": metric.calibrated,
        "raw_simulated_value": metric.raw,
        "change_value": metric.delta,
        "evidence_level": metric.evidence_level,
        "causal_disclaimer": metric.disclaimer,
        # ... other fields
    })
```

### 7.2 Writing to simulation_validity_checks

After simulation, Waker validates results and writes checks:

```python
for check in simulation.validity_checks:
    feishu_write("simulation_validity_checks", {
        "validity_id": f"V_{scenario_id}_{check.metric}_{check.type}",
        "scenario_id": scenario_id,
        "metric_key": check.metric,
        "check_type": check.type,
        "check_result": check.result,  # PASS or WARNING
        "risk_level": check.risk_level,
        "requires_disclaimer": check.requires_disclaimer,
        "disclaimer_text": check.disclaimer_text,
    })
```

### 7.3 Updating scenario_catalog

After successful simulation, update `last_simulated_at`:

```python
feishu_update("scenario_catalog", 
    filter={"scenario_id": scenario_id},
    values={"last_simulated_at": datetime.now().isoformat()}
)
```

---

## 8. Error Handling

### 8.1 Parameter Validation Errors

```
If user sets simulated_value outside [min_value, max_value]:
  → Return error: "参数 '{input_field_label}' 的值 {simulated_value} 超出有效范围 [{min_value}, {max_value}]"
```

### 8.2 Simulation Script Errors

```
If simulation script fails:
  → Write error to simulation_validity_checks with check_result = "ERROR"
  → Do NOT update scenario_catalog.last_simulated_at
  → Return error message with traceback
```

### 8.3 Missing Data Errors

```
If required parameter is missing from scenario_parameters:
  → Return error: "场景 {scenario_id} 缺少必要参数 {parameter_id}"
```

---

## 9. Quick Reference Summary

### Waker Read Flow
1. Read `scenario_catalog` → Get `scenario_id` (ACTIVE only)
2. Read `scenario_parameters` → Get inputs (editable=true for what-if)
3. Run simulation script
4. Write `scenario_simulation_results` → Results with evidence levels
5. Write `simulation_validity_checks` → Validity checks
6. Update `scenario_catalog.last_simulated_at`

### Mandatory Disclaimer Display
- **ANY result with evidence_level ≠ ARITHMETIC_CALCULATION** → Show `causal_disclaimer`
- **S08 GMV results** → Show denominator clarification note
- Use exact text from `causal_disclaimer` column in results table

---

## 10. Reproducibility

All Waker interactions reference:
- `data/processed/feishu_scenario_catalog.csv` - Scenario registry
- `data/processed/feishu_scenario_parameters.csv` - Parameters
- `data/processed/feishu_simulation_results.csv` - Results
- `data/processed/feishu_validity_checks.csv` - Validity checks

This contract ensures Waker correctly distinguishes between:
- **System-generated data** (editable_by_waker = false) - do not modify
- **User-defined what-if inputs** (editable_by_waker = true) - use for simulation
- **ARITHMETIC results** - certain, no disclaimer
- **OBSERVATIONAL/EXPERIENTIAL results** - uncertain, MUST show disclaimer

**Contract Date**: 2026-05-20
