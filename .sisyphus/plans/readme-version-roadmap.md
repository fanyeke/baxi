# README 版本路线图更新

## TL;DR

> **Quick Summary**: 更新 README.md，新增版本演进小节，版本号从 v0.1 更新到 v0.5.2，加入 Phase I 状态。
>
> **Deliverables**:
> - README.md：新增"版本演进"小节，覆盖 v0.1→v0.5.2 + Phase I
> - 第 305 行"当前版本"从 v0.1 更新到 v0.5.2
>
> **Estimated Effort**: Quick (单文件，5-10 分钟)
> **Parallel Execution**: N/A（单任务）
> **Critical Path**: Task 1 → Done

---

## Context

### Original Request
用户希望更新文档，反映 v0.5.2 和 Phase I 的真实实现状态。

### Interview Summary
**Key Discussions**:
- v0.5.2: 实际已完成（5 commit，6/6 DoD），但 README 未记录
- Phase I: 核心 80% 完成，LLM 未激活
- 用户选择：新增版本路线图 + 更新当前版本号
- 用户选择：暂不激活 LLM
- 范围：仅 README.md，不改其他文件

### Metis Review
**Identified Gaps** (addressed):
- README 中无"v0.5.2"和"Phase I"标签——需要新增内容而非修改现有行
- 当前版本仍写 v0.1——需要更新到 v0.5.2
- 版本命名体系不一致（Phase vs v0.x）——新增小节统一说明

---

## Work Objectives

### Core Objective
在 README.md 中新增版本演进路线图，准确反映项目从 v0.1 到 v0.5.2 的演进历程和 Phase I 的当前状态。

### Concrete Deliverables
- README.md 第 305 行更新为 v0.5.2
- README.md "后续计划"节后新增"版本演进"小节

### Definition of Done
- [ ] `grep "v0.1-heuristic" README.md` 返回 0 匹配
- [ ] `grep "v0.5.2" README.md` 返回 ≥2 匹配（版本号 + 路线图）
- [ ] `grep "Phase I" README.md` 返回 ≥1 匹配
- [ ] README 中新增的版本演进小节格式与现有 Markdown 一致

### Must Have
- 版本号从 v0.1 更新到 v0.5.2
- v0.5.2 标注为 ✅ DONE
- Phase I 标注为 🟡 核心完成 / LLM 待激活

### Must NOT Have (Guardrails)
- 不改动 Phase 1-7 的 FROZEN 内容
- 不修改 pyproject.toml / package.json / 其他版本号文件
- 不删除现有的 "后续计划" 内容
- 不过度展开每个版本的细节（每个版本 1-2 行）

---

## Verification Strategy

### Test Decision
- **Infrastructure exists**: N/A（文档变更）
- **Automated tests**: None
- **QA Policy**: 手动检查 grep + Markdown 渲染

---

## Execution Strategy

### Single Task

```
Task 1: README 版本路线图更新 [quick]
├── 更新第 305 行"当前版本" v0.1 → v0.5.2
├── "后续计划"节后新增"版本演进"小节
└── commit
```

---

## TODOs

- [x] 1. README 版本路线图更新

  **What to do**:
  1. 编辑 README.md 第 305 行，将 `v0.1-heuristic-decision-sandbox（规则驱动的 AI-ready 决策沙盘，不含真实 LLM 决策）` 更新为 `v0.5.2（含规则决策引擎、SQLite 后端、FastAPI 网关、React 控制台、飞书集成）`
  2. 在 "后续计划" 的 Phase 10 之后，"运行说明" 之前，新增 "### 版本演进" 小节，包含：

  ```markdown
  ### 版本演进

  | 版本 | 核心能力 | 状态 |
  |------|---------|------|
  | v0.1 | 规则驱动决策沙盘 (heuristic) | ✅ DONE |
  | v0.2 | SQLite 后端 + 12 表 Schema + 配置化治理 | ✅ DONE |
  | v0.3 | 维度级异常检测 (seller/category/region) | ✅ DONE |
  | v0.3.1 | 飞书沙盘集成 + 决策质量校准 | ✅ DONE |
  | v0.4 | 分发适配器 (Feishu/GitHub/Local/Manual) | ✅ DONE |
  | v0.5 | API 网关 (FastAPI:8765, OpenAPI, Bearer Token) | ✅ DONE |
  | v0.5.1 | React 控制台 Alpha (7 pages, TanStack Query) | ✅ DONE |
  | v0.5.2 | 控制台 Beta 硬化 (P0修复 + 日志诊断) | ✅ DONE |
  | Phase I | 全量数据 + AI 决策引擎 (LLM 代码就绪，待激活) | 🟡 核心完成 |
  | Phase II+ | 维度告警扩展 / 真实 LLM 决策 / 自动调度 | ❌ 未启动 |
  ```

  **Must NOT do**:
  - 不要修改 Phase 3-10 的任何内容
  - 不要修改 pyproject.toml / package.json 中的版本号
  - 不要添加超过 2 行的版本描述

  **Recommended Agent Profile**:
  > 简单文档编辑，无需 specialist agent
  - **Category**: `quick`
  - **Skills**: `[]`
  - **Reason**: 单文件 markdown 编辑，2 处修改

  **References** (CRITICAL - Be Exhaustive):
  **Pattern Reference**:
  - `README.md:303-349` - 现有"后续计划"格式，新小节应匹配此风格

  **Evidence Reference**:
  - `.sisyphus/evidence/v0.5.2-acceptance.md` - v0.5.2 完成证据（6/6 DoD PASS）
  - `.sisyphus/plans/phase-i-full-decision-mode.md:1-10` - Phase I 计划范围
  - `.sisyphus/plans/v0.5.2-hardening.md:1-10` - v0.5.2 计划范围

  **Acceptance Criteria**:

  **QA Scenarios (MANDATORY)**:

  ```
  Scenario: 版本号正确更新
    Tool: Bash (grep)
    Steps:
      1. grep "v0.1-heuristic" /home/zzz/project/baxi/README.md
      2. 确认返回 0 行（旧版本号已替换）
    Expected Result: 0 matches
    Evidence: .sisyphus/evidence/task-1-version-updated.txt

  Scenario: 新增版本路线图存在
    Tool: Bash (grep)
    Steps:
      1. grep -c "v0.5.2" /home/zzz/project/baxi/README.md
      2. 确认返回 ≥2（版本号行 + 路线图行）
      3. grep "Phase I" /home/zzz/project/baxi/README.md
      4. 确认返回 ≥1
    Expected Result: v0.5.2 >=2 matches, Phase I >=1 match
    Evidence: .sisyphus/evidence/task-1-roadmap-exists.txt

  Scenario: 已有内容未被破坏
    Tool: Bash (grep)
    Steps:
      1. grep -c "Phase 3.*全局业务分析.*FROZEN" /home/zzz/project/baxi/README.md
      2. 确认返回 =1
      3. grep -c "Phase 10.*Waker" /home/zzz/project/baxi/README.md
      4. 确认返回 =1
    Expected Result: 现有 Phase 3-10 内容完整保留
    Evidence: .sisyphus/evidence/task-1-existing-intact.txt
  ```

  **Evidence to Capture**:
  - [ ] `.sisyphus/evidence/task-1-version-updated.txt` — grep 结果确认旧版本号已移除
  - [ ] `.sisyphus/evidence/task-1-roadmap-exists.txt` — grep 结果确认新路线图存在
  - [ ] `.sisyphus/evidence/task-1-existing-intact.txt` — grep 结果确认原有内容完整

  **Commit**: YES
  - Message: `docs(readme): add version roadmap v0.1-v0.5.2 + Phase I status`
  - Files: `README.md`

---

## Final Verification Wave

- [x] F1. grep 验证：v0.1-heuristic 已消失，v0.5.2 出现，Phase I 出现
- [x] F2. Markdown 渲染验证：read README.md 确认格式正确

---

## Commit Strategy

- **1**: `docs(readme): add version roadmap v0.1-v0.5.2 + Phase I status`
  - Files: `README.md`

---

## Success Criteria

### Verification Commands
```bash
grep "v0.1-heuristic" README.md          # Expected: 0 matches
grep "v0.5.2" README.md                   # Expected: >=2 matches
grep "Phase I" README.md                  # Expected: >=1 matches
grep "✅ DONE" README.md | grep "v0.5.2"  # Expected: 1 match
```

### Final Checklist
- [x] 当前版本显示 v0.5.2
- [x] v0.5.2 标注 ✅ DONE
- [x] Phase I 标注 🟡 核心完成
- [x] 现有 Phase 3-10 内容未改动
