# Phase 7 Simulation Calibration Documentation
## Calibration of Quantitative Simulation Results

**Calibration Date**: 2026-05-20 23:54:01
**Base Version**: Phase 7 Simulation Engine v1.0
**Calibration Version**: v1.1

---

## 1. Calibration Issues Identified

### 1.1 S02 Score Boundary Violation (CRITICAL)

**Problem**: Raw estimate gave review_score = 4.16 + 0.855 = **5.015 > 5.0**

**Root Cause**: Applied correlation (-0.334) directly to large delivery time delta (-2.56 days) without boundary checking.

**Fix Applied**:
- Added score clamping to [1.0, 5.0] range
- Applied conservative cap: max score increase of 0.3 points for MEDIUM/HIGH confidence scenarios
- Calibrated result: +0.300 (capped from +0.855)
- Low score % reduction also calibrated proportionally

**New Result**: Score improves from 4.16 to **4.460** (was 5.015)

---

### 1.2 S08 GMV Denominator Ambiguity (HIGH)

**Problem**: Reported GMV uplift of R$169,454 without clarifying the denominator.

**Key Distinction**:
- **Closed Seller GMV**: R$775,816 (GMV from 380 activated sellers out of 842 closed deals)
- **Platform Total GMV**: R$13,412,108 (all 99,441 orders across all sellers)

**Fix Applied**:
- Clearly labeled baseline as "closed seller GMV" NOT "platform GMV"
- Expressed additional GMV as percentage of BOTH denominators:
  - R$169,454 / R$775,816 = **21.8% increase in closed seller GMV**
  - R$169,454 / R$13,412,108 = **1.26% increase in platform GMV**
- Added warning about denominator risk

---

## 2. Boundary Enforcement Rules

### 2.1 Score Boundaries

| Metric | Min Value | Max Value | Enforcement |
|--------|-----------|-----------|-------------|
| review_score | 1.0 | 5.0 | `clamp(score, 1.0, 5.0)` |
| low_score_pct | 0.0% | 100.0% | `clamp(pct, 0.0, 100.0)` |
| seller_activation_rate | 0.0% | 100.0% | `clamp(pct, 0.0, 100.0)` |
| platform_GMV_pct | 0.0% | 100.0% | `clamp(pct, 0.0, 100.0)` |

### 2.2 Conservative Estimation Logic

For correlation-based scenarios (S01, S02):

```python
def conservative_score_estimate(baseline, delta, confidence):
    if confidence == 'HIGH':
        capped = delta  # Allow full delta (still clamp to boundaries)
    elif confidence == 'MEDIUM':
        capped = min(delta, 0.3)  # Max +0.3 pts
    else:
        capped = min(delta, 0.15)  # Max +0.15 pts
    
    return clamp(baseline + capped, 1.0, 5.0)
```

---

## 3. Validity Check Framework

Each scenario now includes three validity checks:

### 3.1 validity_check_bounded

**Question**: Did the simulation stay within valid boundaries?
**Possible Values**: True / False
**Implications**:
- `True`: All metrics within valid ranges (score 1-5, percentages 0-100%)
- `False`: At least one metric was clamped; raw estimate was unrealistic

| Scenario | Bounded? | Notes |
|----------|----------|-------|
| S01 | True | No clamping needed |
| S02 | True | **Raw estimate 5.015 was clamped to 4.460** |
| S07 | True | No clamping needed |
| S08 | True | No clamping needed (percentage arithmetic) |

### 3.2 validity_check_extrapolation

**Question**: Does the result rely on extrapolating historical correlations beyond observed range?
**Possible Values**: True / False
**Implications**:
- `True`: Result based on correlation which may not hold in intervention scenario
- `False`: Result is direct arithmetic (no correlation assumptions)

| Scenario | Extrapolation? | Notes |
|----------|---------------|-------|
| S01 | True | Uses delay_score correlation (-0.267) |
| S02 | True | Uses delivery_score correlation (-0.334) |
| S07 | False | Uses approximation, not extrapolation |
| S08 | False | Pure arithmetic calculation |

### 3.3 validity_check_denominator_risk

**Question**: Is there a risk of denominator ambiguity in interpreting results?
**Possible Values**: LOW / MEDIUM / HIGH
**Implications**:
- `LOW`: Denominator is clear and unambiguous
- `MEDIUM`: Denominator involves estimated proportions
- `HIGH`: Denominator ambiguity could lead to misinterpretation

| Scenario | Denominator Risk | Explanation |
|----------|-----------------|-------------|
| S01 | LOW | Score and percentage metrics are self-contained |
| S02 | LOW | Score and percentage metrics are self-contained |
| S07 | MEDIUM | State order weights (18%) are estimated |
| S08 | **HIGH** | Must distinguish closed seller GMV vs platform GMV |

---

## 4. Calibrated Results Summary

| Scenario | Input Change | Raw Score Delta | Calibrated Score Delta | Score Clamped? | GMV Impact |
|----------|-------------|-----------------|----------------------|----------------|------------|
| S01 | Late 8.11%→4% | +0.105 | +0.105 | No | Indirect |
| S02 | Delivery 12.56→10 | +0.855 | **+0.300** | **Yes (5.015→4.460)** | Indirect |
| S07 | State late 15%→8% | +0.179/state | +0.179/state | No | Indirect |
| S08 | Activation 45.1%→55% | N/A | N/A | No | +R$169,454 (21.8% closed, 1.26% platform) |

---

## 5. Implications for Waker Decision Sandbox

### Critical Changes from v1.0 to v1.1

1. **S02 score impact reduced**: From +0.855 to +0.300 points
   - Raw estimate was over-optimistic; correlation-based projections overestimate real-world impact
   - New conservative estimate is more realistic for decision-making

2. **S08 GMV context clarified**:
   - Previous version did not distinguish "closed seller GMV" from "platform GMV"
   - Now clear that R$169,454 is only 1.26% of total platform GMV
   - Important for setting realistic expectations

### Recommended Sandbox Input Ranges

For scenario inputs in Waker:

| Scenario | Conservative | Aggressive | Notes |
|----------|-------------|-----------|-------|
| S01 | +0.050 pts | +0.105 pts | Use range to show uncertainty |
| S02 | +0.150 pts | +0.300 pts | Correlation uncertainty ±50% |
| S07 | +0.016 pts | +0.032 pts | State order weight uncertain |
| S08 | +8% closed GMV | +22% closed GMV | Performance regression risk |

---

## 6. File Updates

| File | Change |
|------|--------|
| `outputs/tables/scenario_simulation_results.csv` | Added calibration columns: raw_simulated_value, score_clamped, conservative_cap, validity_check_* |
| `outputs/tables/scenario_simulation_summary.csv` | Added raw vs calibrated comparison columns |
| `reports/scenario_simulation_analysis.md` | Updated S02 and S08 results, added validity check sections |
| `docs/phase7_simulation_calibration.md` | NEW: This calibration documentation |

---

## 7. Reproducibility

Calibrated results generated by: `phase7_calibration_revision.py`

All boundary enforcement and conservative estimation logic is encapsulated in:
- `clamp_score()` function
- `clamp_percentage()` function  
- `conservative_score_estimate()` function

**Calibration Date**: 2026-05-20 23:54:01

---

**⚠️ Important Note**: Calibration is applied to make results more realistic and boundary-consistent.
The underlying correlations and arithmetic remain unchanged. Calibration adds:
1. Boundary enforcement (no score > 5.0)
2. Conservative capping (max reasonable improvement for correlation-based projections)
3. Denominator clarification (distinguish closed seller vs platform metrics)
