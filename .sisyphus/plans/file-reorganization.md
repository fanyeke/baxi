# Olist 电商项目文件整理方案

## TL;DR

> **核心目标**: 将经过 10 个阶段累积的 Olist 电商分析项目重新整理为清晰的结构，清理冗余文件、消除重复版本、补全 README 文档、创建 .gitignore。
> 
> **关键决策**: 整理分为 Phase A（文件移动/清理）和 Phase B（脚本路径修复）两个独立阶段。本次计划仅执行 Phase A，脚本标记为"frozen"暂不修路径。
> 
> **预计工作量**: 中等 (~15 个并行任务，3 个 Wave)
> **并行执行**: YES - 3 个 Wave

---

## Context

### 问题发现

| # | 问题类型 | 具体描述 | 影响 |
|---|---------|---------|------|
| 1 | 冗余版本 | 6 对 original + `_corrected` 文件并存 | 不知道该用哪个版本 |
| 2 | 备份文件 | `project_overview.html.bak` (43K)、`project_overview_partial.html` (43K) | 占用空间 |
| 3 | 验证产物混放 | `phase9_*` 和 `phase10_*` CSV 混在 `outputs/tables/` 里 | 分析结果不清晰 |
| 4 | README 不完整 | 只写了 Phase 1-2，Phase 3-10 无文档 | 新项目看不懂 |
| 5 | 目录混乱 | `doc/` (1文件) 和 `docs/` 并存 | 文件归属不明 |
| 6 | 无 .gitignore | 123MB CSV 全被 git 跟踪 | 仓库体积大 |
| 7 | 命名不一致 | `top10_*`、`top15_*`、`*_4metrics` | 命名规范不统一 |

### Metis 关键发现

- 所有脚本硬编码路径 565+ 处 — 移动文件会导致脚本失效
- **Phase A/B 拆分必须**：只移文件不修路径，否则变成 2 天路径重构
- `_corrected` 文件必须先验证行数再删除原版

---

## 目标目录结构（摘要）

```
baxi/
├── .gitignore                         ← 新建
├── README.md                          ← 更新：含 Phase 3-10
├── data/
│   ├── raw/                           ← 新建，11 CSV + archive.zip
│   ├── interim/                       ← 中间表 (保留)
│   └── processed/                     ← 飞书产物 (不动)
├── scripts/                           ← 所有 Python 脚本 (加 phaseXX_ 前缀)
│   ├── _FROZEN.md                     ← 新建，说明脚本冻结状态
│   └── phase01_*.py ... phase07_*.py
├── outputs/
│   ├── charts/                        ← 图表 (清理 corrected)
│   ├── tables/                        ← 分析结果表 (清理 phase9/10)
│   └── validation/                    ← 新建，phase9/10 验证产物
├── reports/                           ← 分析报告 (清理 corrected)
└── docs/                              ← 技术文档 (删除 doc/)
```

---

## Work Objectives

### Definition of Done
- [ ] 零 `_corrected`/`.bak`/`_partial` 文件
- [ ] CSV 集中在 data/raw/
- [ ] 脚本集中在 scripts/ 并加 phaseXX_ 前缀
- [ ] phase9/10 移至 outputs/validation/
- [ ] .gitignore 存在
- [ ] README 含 Phase 1-10
- [ ] git status clean

### Must NOT Have (Guardrails)
- ❌ 不修改任何 Python 脚本代码（Phase A 边界）
- ❌ 不删除 _corrected 文件前不验证行数
- ❌ 不改动 .claude/ 目录
- ❌ 不做 git history rewrite

---

## 执行策略

```
Wave 1 (5个并行)          Wave 2 (7个并行)              Wave 3 (3个并行)
├─ T1: .gitignore         ├─ T6: CSV→data/raw/          ├─ T13: README更新
├─ T2: _corrected验证      ├─ T7: _corrected替换原版      ├─ T14: 完整性验证
├─ T3: 目录创建            ├─ T8: phase9/10→validation    └─ T15: Git提交
├─ T4: backup清理          ├─ T9: 脚本移至scripts/
└─ T5: _FROZEN.md         ├─ T10: 报告corrected替换
                          ├─ T11: 图表corrected替换
                          └─ T12: doc/处理
```

---

## TODOs

详见各 Wave 任务。共 15 个任务 + 3 个 Final Review。

### Wave 1 - 验证与准备

- [ ] W1-T1: 创建 .gitignore（忽略 CSV/ZIP/HTML/PNG/bak/pycache）
- [ ] W1-T2: `_corrected` 文件行数验证（必须 ≥ 原版才可安全替换）
- [ ] W1-T3: 创建 data/raw/ 和 outputs/validation/
- [ ] W1-T4: 删除 .bak 和 _partial 文件
- [ ] W1-T5: 创建 scripts/_FROZEN.md（说明路径冻结状态）

### Wave 2 - 文件移动与清理

- [ ] W2-T6: 11 CSV + archive.zip 移入 data/raw/
- [ ] W2-T7: _corrected 替换原版 → 删除 _corrected（tables/reports/charts/interim 共 7 对）
- [ ] W2-T8: 4 个 phase9/10 CSV 移至 outputs/validation/
- [ ] W2-T9: 13 个脚本移至 scripts/ 并重命名 phaseXX_ 前缀
- [ ] W2-T10: 报告 corrected 版替换（1 个）
- [ ] W2-T11: 图表 corrected 版替换（1 个）
- [ ] W2-T12: 删除 doc/ 目录

### Wave 3 - 验证与文档

- [ ] W3-T13: README.md 更新（添加 Phase 3-10 + 新结构树 + frozen 说明）
- [ ] W3-T14: 文件完整性验证（grep 确认零残留 + 计数确认）
- [ ] W3-T15: git 提交

### Final Verification

- [ ] F1: 结构完整性
- [ ] F2: 文档引用验证
- [ ] F3: README 质量

---

## Commit Strategy

- **一次性提交**: `refactor(project): 整理文件结构 - 10阶段产物规范化`

---

## Phase B 展望（后续）

本次不执行，Phase B 将：
1. 创建 `scripts/config.py` 集中管理路径常量
2. 更新所有脚本使用 config.py
3. 测试所有脚本可运行
4. 可选：修复 `explore_data.py` 的 os.listdir 路径
