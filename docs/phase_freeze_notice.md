# Phase 1-7 FROZEN 策略说明

> 生效日期: 2026-05-22

## 策略

`scripts/` 目录下的 Phase 1-7 分析脚本已标记为 **FROZEN**（历史归档资产）。

### FROZEN 的含义

- Phase 1-7 的脚本和产出（基础表、图表、报告）是项目的**历史分析资产**
- 这些产出已被后续阶段（Stage 2-4）消费和封装
- **不会再修改** FROZEN 脚本的原始代码
- 产出文件（CSV、PNG、MD）保持当前版本不变

### 已归档内容

| 类型 | 数量 | 位置 |
|------|------|------|
| 分析脚本 | 14 个 `.py` | `scripts/phase01_*` ~ `scripts/phase07_*` |
| 基础表 | 2 张 | `data/interim/order_level_base.csv`, `item_level_base.csv` |
| 分析图表 | ~36 张 | `outputs/charts/*.png` |
| 分析报告 | 8 份 | `reports/` 目录下 6 份 + `docs/` 下 2 份 |

### 已修复的脚本

以下 3 个脚本已修复路径，保留用于基础表重新生成：
1. `scripts/phase02_build_data_model.py` — 数据模型构建
2. `scripts/phase02_generate_docs.py` — 文档生成
3. `scripts/calculate_channel_thresholds.py` — 渠道阈值计算

其余所有 EDA（探索性数据分析）脚本保持 FROZEN 状态，不修改、不运行。

### 迁移指引

新功能的开发应基于 Stage 2-4（数据产品层、飞书工作台、决策闭环沙盘），直接阅读 FROZEN 脚本了解历史分析逻辑，但不要在 FROZEN 脚本上新增功能。

### 参考

- 完整 FROZEN 说明: `scripts/_FROZEN.md`
- 阶段规划: `.sisyphus/plans/phase-consolidation.md`
