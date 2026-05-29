# Order Governance Skill Comparison Report

**Generated:** 2026-05-29 23:54:05
**Test cases:** 10
**Versions compared:** V1 (Basic Rules), V2 (Context-Aware), V3 (Ontology-Aware)

## 1. Accuracy Summary

| Version | Correct | Total | Accuracy |
|---------|---------|-------|----------|
| V1 | 8 | 10 | **80%** |
| V2 | 9 | 10 | **90%** |
| V3 | 10 | 10 | **100%** |

## 2. Performance (Average Decision Time)

| Version | Avg Time (µs) | Min (µs) | Max (µs) | Total (µs) |
|---------|---------------|----------|----------|------------|
| V1 | 1.24 | 0.56 | 2.85 | 12.41 |
| V2 | 1.44 | 0.52 | 2.82 | 14.40 |
| V3 | 1.71 | 0.50 | 2.81 | 17.08 |

## 3. Per-Scenario Detailed Comparison

| ID | Scenario | Expected | V1 | V2 | V3 | V1 Match | V2 Match | V3 Match |
|----|----------|----------|----|----|----|----------|----------|----------|
| T1 | 低风险老客订单 | APPROVE | APPROVE (0) | APPROVE (0) | APPROVE (0) | ✅ | ✅ | ✅ |
| T2 | 低风险新客订单 | APPROVE | APPROVE (25) | APPROVE (25) | APPROVE (25) | ✅ | ✅ | ✅ |
| T3 | 中风险订单 | REVIEW | REVIEW (45) | REVIEW (45) | REVIEW (45) | ✅ | ✅ | ✅ |
| T4 | 高风险订单 | DECLINE | DECLINE (100) | DECLINE (115) | DECLINE (125) | ✅ | ✅ | ✅ |
| T5 | 边界值测试 (score≈30) | APPROVE | APPROVE (0) | APPROVE (0) | APPROVE (0) | ✅ | ✅ | ✅ |
| T6 | 边界值测试 (score≈60) | REVIEW | DECLINE (70) | DECLINE (85) | REVIEW (65) | ❌ | ❌ | ✅ |
| T7 | 硬规则触发 (已知欺诈) | DECLINE | DECLINE (25) | DECLINE (25) | DECLINE (25) | ✅ | ✅ | ✅ |
| T8 | 硬规则触发 (VIP 客户) | APPROVE | APPROVE (15) | APPROVE (15) | APPROVE (15) | ✅ | ✅ | ✅ |
| T9 | 深夜高价值订单 | REVIEW | REVIEW (40) | REVIEW (35) | REVIEW (35) | ✅ | ✅ | ✅ |
| T10 | 速度攻击 | DECLINE | REVIEW (55) | DECLINE (70) | DECLINE (60) | ❌ | ✅ | ✅ |

## 4. Risk Score Comparison

| ID | Scenario | V1 Score | V2 Score | V3 Score | Delta V2-V1 | Delta V3-V2 |
|----|----------|----------|----------|----------|-------------|-------------|
| T1 | 低风险老客订单 | 0 | 0 | 0 | +0 | +0 |
| T2 | 低风险新客订单 | 25 | 25 | 25 | +0 | +0 |
| T3 | 中风险订单 | 45 | 45 | 45 | +0 | +0 |
| T4 | 高风险订单 | 100 | 115 | 125 | +15 | +10 |
| T5 | 边界值测试 (score≈30) | 0 | 0 | 0 | +0 | +0 |
| T6 | 边界值测试 (score≈60) | 70 | 85 | 65 | +15 | -20 |
| T7 | 硬规则触发 (已知欺诈) | 25 | 25 | 25 | +0 | +0 |
| T8 | 硬规则触发 (VIP 客户) | 15 | 15 | 15 | +0 | +0 |
| T9 | 深夜高价值订单 | 40 | 35 | 35 | -5 | +0 |
| T10 | 速度攻击 | 55 | 70 | 60 | +15 | -10 |

## 5. Decision Reason Analysis

### V1

| ID | Decision | Score | Reason |
|----|----------|-------|--------|
| T1 | APPROVE | 0 | Low risk (score=0) |
| T2 | APPROVE | 25 | Low risk (score=25) |
| T3 | REVIEW | 45 | Medium risk (score=45) |
| T4 | DECLINE | 100 | High risk (score=100) |
| T5 | APPROVE | 0 | Low risk (score=0) |
| T6 | DECLINE | 70 | High risk (score=70) |
| T7 | DECLINE | 25 | Known fraud pattern detected |
| T8 | APPROVE | 15 | Long-standing customer |
| T9 | REVIEW | 40 | Medium risk (score=40) |
| T10 | REVIEW | 55 | Medium risk (score=55) |

### V2

| ID | Decision | Score | Reason |
|----|----------|-------|--------|
| T1 | APPROVE | 0 | VIP low risk (score=0) |
| T2 | APPROVE | 25 | New customer low risk (score=25) |
| T3 | REVIEW | 45 | Standard medium risk (score=45) |
| T4 | DECLINE | 115 | New customer high risk (score=115) |
| T5 | APPROVE | 0 | Standard low risk (score=0) |
| T6 | DECLINE | 85 | Standard high risk (score=85) |
| T7 | DECLINE | 25 | Known fraud pattern detected |
| T8 | APPROVE | 15 | Long-standing customer |
| T9 | REVIEW | 35 | Standard medium risk (score=35) |
| T10 | DECLINE | 70 | New customer high risk (score=70) |

### V3

| ID | Decision | Score | Reason |
|----|----------|-------|--------|
| T1 | APPROVE | 0 | VIP ontology-adjusted low risk (score=0) |
| T2 | APPROVE | 25 | New customer ontology-adjusted low risk (score=25) |
| T3 | REVIEW | 45 | Standard ontology-adjusted medium risk (score=45) |
| T4 | DECLINE | 125 | New customer ontology-adjusted high risk (score=125) |
| T5 | APPROVE | 0 | Standard ontology-adjusted low risk (score=0) |
| T6 | REVIEW | 65 | Standard ontology-adjusted medium risk (score=65) |
| T7 | DECLINE | 25 | Known fraud pattern detected |
| T8 | APPROVE | 15 | Long-standing customer |
| T9 | REVIEW | 35 | Standard ontology-adjusted medium risk (score=35) |
| T10 | DECLINE | 60 | New customer ontology-adjusted high risk (score=60) |

## 6. Failures Analysis

### V1 Failures (2)

- **T6** (边界值测试 (score≈60)): expected `REVIEW`, got `DECLINE` (score=70)
  - Reason: High risk (score=70)
- **T10** (速度攻击): expected `DECLINE`, got `REVIEW` (score=55)
  - Reason: Medium risk (score=55)

### V2 Failures (1)

- **T6** (边界值测试 (score≈60)): expected `REVIEW`, got `DECLINE` (score=85)
  - Reason: Standard high risk (score=85)

### V3 — No failures!

## 7. Recommendation

**Recommended version: V3**

### Rationale

- **V1 (Basic Rules)**: 80% accuracy. Simple weighted scoring with fixed thresholds. Misses context like time-of-day, customer history, and product domain risk. Fast but imprecise.
- **V2 (Context-Aware)**: 90% accuracy. Adds customer tier segmentation (VIP/NEW/STANDARD), time-based risk adjustments, and purchase history analysis. Fixes velocity-attack detection that V1 misses.
- **V3 (Ontology-Aware)**: 100% accuracy. Builds on V2 with seller health scoring, product risk levels, and cross-domain coherence discounting. Correctly handles mixed-signal borderline cases by weighing the overall context rather than treating each signal independently.

### Key Improvements

| Feature | V1 | V2 | V3 |
|---------|----|----|----|
| Weighted risk signals | ✅ | ✅ | ✅ |
| Hard rules (fraud/disposable email) | ✅ | ✅ | ✅ |
| Customer tier (VIP/NEW/STANDARD) | ❌ | ✅ | ✅ |
| Dynamic thresholds by tier | ❌ | ✅ | ✅ |
| Time-of-day risk adjustment | ❌ | ✅ | ✅ |
| Purchase history analysis | ❌ | ✅ | ✅ |
| Seller health scoring | ❌ | ❌ | ✅ |
| Product risk level | ❌ | ❌ | ✅ |
| Cross-domain coherence | ❌ | ❌ | ✅ |
| Feedback loop recording | ❌ | ❌ | ✅ |

**V3 achieves the highest accuracy (100%) with acceptable performance overhead.**

---

*Report generated by simulate_skills.py at 2026-05-29 23:54:05*