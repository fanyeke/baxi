# Baxi AIP-Inspired Architecture Migration Plan

## TL;DR

> **Quick Summary**: 基于Palantir AIP理念，对Baxi项目进行渐进式架构增强，包括身份认证、决策血缘、Ontology层、Marking系统、Actions模式，同时完成Python→Go的全面迁移。
> 
> **Deliverables**:
> - 身份认证系统（JWT/token-based）
> - 决策血缘追踪系统
> - 轻量级Ontology层
> - 动态Marking系统
> - 增强Actions模式
> - 完成Python→Go迁移
> 
> **Estimated Effort**: 10周（约2.5个月）
> **Parallel Execution**: YES - 6个阶段，部分可并行
> **Critical Path**: 身份认证 → 决策血缘 → Ontology → Marking → Actions → 迁移完成

---

## Context

### Original Request
用户要求基于Palantir AIP理念对Baxi项目进行改造和重构，不包含LLM决策集成，实现渐进式迁移。

### Interview Summary
**Key Discussions**:
- 项目当前状态：双系统架构（Python/SQLite + Go/PostgreSQL）
- 目标：借鉴Palantir AIP理念增强治理能力
- 约束：不包含LLM决策集成，渐进式迁移

**Research Findings**:
- Palantir AIP核心：Ontology（语义层）、Actions（写API）、Markings（数据分类）、Decision Lineage（决策血缘）
- Baxi项目已有基础：Ontology包、治理层、Actions注册表、审计日志
- 关键差距：身份传播、决策血缘、标记强制执行、Ontology统一

### Metis Review
**Identified Gaps** (addressed):
- 身份认证：当前硬编码"qoder"，需要真实用户身份
- 决策血缘：当前审计日志无correlation_id，无法端到端追踪
- Marking系统：当前仅在LLM上下文中应用，普通查询不检查
- Ontology：三处定义（YAML、Go、DB）可能漂移

---

## Work Objectives

### Core Objective
基于Palantir AIP理念，对Baxi项目进行渐进式架构增强，完成Python→Go迁移。

### Concrete Deliverables
- 身份认证系统（JWT/token-based，替换硬编码"qoder"）
- 决策血缘追踪系统（event-sourced，端到端追踪）
- 轻量级Ontology层（DB驱动，统一定义源）
- 动态Marking系统（查询时强制执行，行级安全）
- 增强Actions模式（动态注册，对象作用域）
- 完成Python→Go迁移（五阶段渐进式）

### Definition of Done
- [ ] 所有新表创建完成，数据迁移完成
- [ ] 所有新接口实现完成，适配器测试通过
- [ ] 特性标志系统工作正常
- [ ] 回滚方案验证通过
- [ ] 前端成功切换到Go API
- [ ] Python API退役

### Must Have
- 身份认证：真实用户身份传播，替换硬编码"qoder"
- 决策血缘：端到端追踪（alert → case → context → decision → proposal → review → action）
- Ontology层：DB驱动，统一定义源，支持关系遍历
- Marking系统：查询时强制执行，行级安全
- Actions模式：动态注册，对象作用域限制
- 渐进式迁移：每个阶段可回滚，零停机

### Must NOT Have (Guardrails)
- 不包含LLM决策集成
- 不破坏现有功能（向后兼容）
- 不强制一次性切换（渐进式）
- 不引入外部依赖（如Palantir平台）
- 不过度设计（80%准确度即可部署）

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** - ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES（Go test + pytest）
- **Automated tests**: YES（Tests-after）
- **Framework**: go test + pytest
- **Testing approach**: 每个阶段完成后运行完整测试套件

### QA Policy
Every task MUST include agent-executed QA scenarios.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Database**: 使用psql执行SQL验证
- **Go代码**: 使用go test执行单元测试
- **API**: 使用curl执行端点测试
- **前端**: 使用Playwright执行E2E测试

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Start Immediately - foundation):
├── Task 1: 身份认证系统 [deep]
├── Task 2: 数据库迁移（ontology tables） [quick]
├── Task 3: 数据库迁移（marking tables） [quick]
├── Task 4: 数据库迁移（lineage tables） [quick]
└── Task 5: 特性标志系统 [quick]

Wave 2 (After Wave 1 - core services):
├── Task 6: OntologyAwareRepo接口+适配器 [deep]
├── Task 7: MarkingService接口+适配器 [deep]
├── Task 8: DecisionLineageService接口+适配器 [deep]
└── Task 9: YAML种子数据迁移 [quick]

Wave 3 (After Wave 2 - integration):
├── Task 10: ContextBuilderV2实现 [deep]
├── Task 11: 向后兼容视图 [quick]
├── Task 12: 验证脚本 [quick]
└── Task 13: 回滚脚本 [quick]

Wave 4 (After Wave 3 - progressive migration):
├── Task 14: 阶段1验证（并行运行） [unspecified-high]
├── Task 15: 阶段2验证（Go为主读） [unspecified-high]
├── Task 16: 阶段3验证（双写） [unspecified-high]
├── Task 17: 阶段4验证（Go为主写） [unspecified-high]
└── Task 18: 阶段5验证（Python退役） [unspecified-high]

Wave FINAL (After ALL tasks - verification):
├── Task F1: 计划合规审计 (oracle)
├── Task F2: 代码质量审查 (unspecified-high)
├── Task F3: 完整QA验证 (unspecified-high)
└── Task F4: 范围保真检查 (deep)
-> Present results -> Get explicit user okay

Critical Path: Task 1 → Task 6 → Task 10 → Task 14 → Task 18 → F1-F4 → user okay
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Wave 1)
```

### Dependency Matrix

| Task | Depends On | Blocks |
|------|------------|--------|
| 1 | - | 6, 7, 8, 10 |
| 2 | - | 6, 9 |
| 3 | - | 7, 9 |
| 4 | - | 8, 9 |
| 5 | - | 10, 14-18 |
| 6 | 1, 2 | 10, 14-18 |
| 7 | 1, 3 | 10, 14-18 |
| 8 | 1, 4 | 10, 14-18 |
| 9 | 2, 3, 4 | 14-18 |
| 10 | 1, 5, 6, 7, 8 | 14-18 |
| 11 | 2, 3, 4 | 14-18 |
| 12 | - | 14-18 |
| 13 | - | 14-18 |
| 14 | 5, 6, 7, 8, 9, 10, 11, 12, 13 | 15 |
| 15 | 14 | 16 |
| 16 | 15 | 17 |
| 17 | 16 | 18 |
| 18 | 17 | F1-F4 |

### Agent Dispatch Summary

- **Wave 1**: 5 tasks - T1 → `deep`, T2-T5 → `quick`
- **Wave 2**: 4 tasks - T6-T8 → `deep`, T9 → `quick`
- **Wave 3**: 4 tasks - T10 → `deep`, T11-T13 → `quick`
- **Wave 4**: 5 tasks - T14-T18 → `unspecified-high`
- **Wave FINAL**: 4 tasks - F1 → `oracle`, F2 → `unspecified-high`, F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.

- [x] 1. 身份认证系统（JWT/token-based）

  **What to do**:
  - 实现JWT/token-based身份认证，替换硬编码"qoder"
  - 创建UserIdentity结构体（UserID, Username, Roles, Email）
  - 修改internal/api/middleware/auth.go，从token提取真实用户身份
  - 修改api/auth.py，同步Python API的用户身份
  - 通过context传播身份到所有service/repository层
  - 实现RBACMiddleware，强制执行access_policy.yml

  **Must NOT do**:
  - 不破坏现有Bearer Token认证流程
  - 不强制要求JWT（支持token映射表回退）

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 2, 3, 4, 5)
  - **Blocks**: Tasks 6, 7, 8, 10
  - **Blocked By**: None

  **References**:
  - `internal/api/middleware/auth.go` - 当前Go auth实现
  - `api/auth.py` - 当前Python auth实现
  - `config/access_policy.yml` - RBAC角色定义
  - `internal/ontology/registry.go` - ObjectRegistry参考

  **Acceptance Criteria**:
  - [ ] Go测试通过：`go test ./internal/api/middleware/...`
  - [ ] Python测试通过：`pytest tests/test_auth.py`
  - [ ] 身份传播验证：API日志包含真实用户ID

  **QA Scenarios**:
  ```
  Scenario: JWT身份提取
    Tool: Bash (curl)
    Preconditions: Go API运行中
    Steps:
      1. curl -H "Authorization: Bearer <valid-jwt>" http://localhost:8080/api/v1/status
      2. 检查响应header包含X-User-ID
      3. 检查audit日志记录真实用户
    Expected Result: 身份正确传播
    Evidence: .sisyphus/evidence/task-1-jwt-identity.txt

  Scenario: 回退到token映射
    Tool: Bash (curl)
    Preconditions: Go API运行中，无JWT
    Steps:
      1. curl -H "Authorization: Bearer <legacy-token>" http://localhost:8080/api/v1/status
      2. 检查响应正常返回
      3. 检查audit日志记录token映射用户
    Expected Result: 向后兼容正常工作
    Evidence: .sisyphus/evidence/task-1-token-fallback.txt
  ```

  **Commit**: YES
  - Message: `feat(auth): implement JWT identity propagation`
  - Files: `internal/api/middleware/auth.go`, `api/auth.py`
  - Pre-commit: `go test ./internal/api/middleware/...`

- [x] 2. 数据库迁移（ontology tables）

  **What to do**:
  - 创建migrations/017_ontology_tables.sql
  - 创建gov.object_type_registry表
  - 创建gov.object_property表
  - 创建gov.object_relationship表
  - 创建索引和约束
  - 编写DOWN脚本

  **Must NOT do**:
  - 不修改现有表结构
  - 不影响现有数据

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 3, 4, 5)
  - **Blocks**: Tasks 6, 9
  - **Blocked By**: None

  **References**:
  - `migrations/006_gov_tables.sql` - 现有gov表参考
  - `config/aip_object_schema.yml` - 对象类型定义
  - `internal/ontology/schema.go` - ObjectType结构体

  **Acceptance Criteria**:
  - [ ] 迁移成功：`goose up`无错误
  - [ ] 表创建成功：`psql -c "\dt gov.*"`显示新表
  - [ ] DOWN脚本工作：`goose down`成功回滚

  **QA Scenarios**:
  ```
  Scenario: 迁移执行
    Tool: Bash (psql)
    Preconditions: PostgreSQL运行中
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" up
      2. psql -c "SELECT COUNT(*) FROM gov.object_type_registry;"
      3. psql -c "\d gov.object_type_registry"
    Expected Result: 表创建成功，结构正确
    Evidence: .sisyphus/evidence/task-2-migration.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): add ontology tables`
  - Files: `migrations/017_ontology_tables.sql`
  - Pre-commit: `goose status`

- [x] 3. 数据库迁移（marking tables）

  **What to do**:
  - 创建migrations/018_marking_tables.sql
  - 创建gov.marking_definition表
  - 创建gov.marking_assignment表
  - 创建gov.pipeline_stage_marking表
  - 创建索引和约束
  - 编写DOWN脚本

  **Must NOT do**:
  - 不修改现有表结构
  - 不影响现有数据

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 4, 5)
  - **Blocks**: Tasks 7, 9
  - **Blocked By**: None

  **References**:
  - `migrations/006_gov_tables.sql` - 现有gov表参考
  - `config/data_markings.yml` - Marking定义
  - `config/data_classification.yml` - 分类定义

  **Acceptance Criteria**:
  - [ ] 迁移成功：`goose up`无错误
  - [ ] 表创建成功：`psql -c "\dt gov.*"`显示新表
  - [ ] DOWN脚本工作：`goose down`成功回滚

  **QA Scenarios**:
  ```
  Scenario: 迁移执行
    Tool: Bash (psql)
    Preconditions: PostgreSQL运行中
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" up
      2. psql -c "SELECT COUNT(*) FROM gov.marking_definition;"
      3. psql -c "\d gov.marking_definition"
    Expected Result: 表创建成功，结构正确
    Evidence: .sisyphus/evidence/task-3-migration.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): add marking tables`
  - Files: `migrations/018_marking_tables.sql`
  - Pre-commit: `goose status`

- [x] 4. 数据库迁移（lineage tables）

  **What to do**:
  - 创建migrations/019_decision_lineage.sql
  - 扩展ai.decision_case表（添加config版本字段）
  - 创建ai.decision_lineage_event表
  - 创建ai.decision_data_snapshot表
  - 创建索引和约束
  - 编写DOWN脚本

  **Must NOT do**:
  - 不丢失现有decision_case数据
  - 不影响现有审计日志

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 5)
  - **Blocks**: Tasks 8, 9
  - **Blocked By**: None

  **References**:
  - `migrations/007_ai_tables.sql` - 现有ai表参考
  - `migrations/010_ai_tables_enhance.sql` - 表增强参考
  - `internal/decision/engine.go` - 决策引擎参考

  **Acceptance Criteria**:
  - [ ] 迁移成功：`goose up`无错误
  - [ ] 表创建成功：`psql -c "\dt ai.*"`显示新表
  - [ ] DOWN脚本工作：`goose down`成功回滚

  **QA Scenarios**:
  ```
  Scenario: 迁移执行
    Tool: Bash (psql)
    Preconditions: PostgreSQL运行中
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" up
      2. psql -c "SELECT COUNT(*) FROM ai.decision_lineage_event;"
      3. psql -c "\d ai.decision_lineage_event"
    Expected Result: 表创建成功，结构正确
    Evidence: .sisyphus/evidence/task-4-migration.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): add decision lineage tables`
  - Files: `migrations/019_decision_lineage.sql`
  - Pre-commit: `goose status`

- [x] 5. 特性标志系统

  **What to do**:
  - 创建internal/feature/flags.go
  - 实现FeatureFlags结构体
  - 实现LoadFlags()从环境变量读取
  - 实现IsEnabled()检查函数
  - 添加单元测试

  **Must NOT do**:
  - 不强制要求所有标志（默认false）
  - 不在运行时修改标志（重启生效）

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 1 (with Tasks 1, 2, 3, 4)
  - **Blocks**: Tasks 10, 14-18
  - **Blocked By**: None

  **References**:
  - `internal/config/config.go` - 配置加载参考
  - `os.Getenv` - 环境变量读取

  **Acceptance Criteria**:
  - [ ] 单元测试通过：`go test ./internal/feature/...`
  - [ ] 默认值正确：所有标志默认false
  - [ ] 环境变量读取正确

  **QA Scenarios**:
  ```
  Scenario: 特性标志加载
    Tool: Bash (go test)
    Preconditions: 无
    Steps:
      1. go test ./internal/feature/... -v
      2. 检查测试输出显示所有标志默认false
      3. 设置环境变量后重新测试
    Expected Result: 测试全部通过
    Evidence: .sisyphus/evidence/task-5-flags.txt
  ```

  **Commit**: YES
  - Message: `feat(feature): add feature flag system`
  - Files: `internal/feature/flags.go`, `internal/feature/flags_test.go`
  - Pre-commit: `go test ./internal/feature/...`

- [x] 6. OntologyAwareRepo接口+适配器

  **What to do**:
  - 创建internal/repository/ontology_aware_repo.go（接口定义）
  - 实现OntologyAwareAdapter（包装现有OntologyRepo+ObjectRegistry）
  - 实现GetObjectByID、QueryByObjectType、GetObjectTypeSchema方法
  - 实现schema验证（返回属性必须在ontology schema中）
  - 添加单元测试

  **Must NOT do**:
  - 不修改现有OntologyRepo接口
  - 不破坏现有ontology查询

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 7, 8, 9)
  - **Blocks**: Tasks 10, 14-18
  - **Blocked By**: Tasks 1, 2

  **References**:
  - `internal/repository/ontology_repository.go` - 现有OntologyRepo实现
  - `internal/ontology/registry.go` - ObjectRegistry实现
  - `internal/ontology/schema.go` - ObjectType定义
  - `config/aip_object_schema.yml` - 对象类型YAML定义

  **Acceptance Criteria**:
  - [ ] 接口定义完整：OntologyAwareRepo接口包含所有方法
  - [ ] 适配器实现正确：OntologyAwareAdapter包装现有实现
  - [ ] 单元测试通过：`go test ./internal/repository/...`

  **QA Scenarios**:
  ```
  Scenario: 对象查询验证
    Tool: Bash (go test)
    Preconditions: PostgreSQL运行中，ontology表有数据
    Steps:
      1. go test ./internal/repository/... -run TestOntologyAwareAdapter -v
      2. 检查GetObjectByID返回正确对象
      3. 检查schema验证拒绝未知属性
    Expected Result: 测试全部通过
    Evidence: .sisyphus/evidence/task-6-ontology-repo.txt
  ```

  **Commit**: YES
  - Message: `feat(ontology): add OntologyAwareRepo interface and adapter`
  - Files: `internal/repository/ontology_aware_repo.go`, `internal/repository/ontology_aware_adapter.go`
  - Pre-commit: `go test ./internal/repository/...`

- [x] 7. MarkingService接口+适配器

  **What to do**:
  - 创建internal/governance/marking_service.go（接口定义）
  - 实现MarkingAdapter（包装现有ClassificationService+ObjectRegistry）
  - 实现GetFieldMarking、GetObjectMarkings、IsLLMAllowed方法
  - 实现从ontology schema回退获取sensitivity
  - 添加单元测试

  **Must NOT do**:
  - 不修改现有ClassificationService接口
  - 不破坏现有分类查询

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 8, 9)
  - **Blocks**: Tasks 10, 14-18
  - **Blocked By**: Tasks 1, 3

  **References**:
  - `internal/governance/classification.go` - 现有ClassificationService
  - `internal/ontology/registry.go` - ObjectRegistry参考
  - `config/data_classification.yml` - 分类定义
  - `config/data_markings.yml` - Marking定义

  **Acceptance Criteria**:
  - [ ] 接口定义完整：MarkingService接口包含所有方法
  - [ ] 适配器实现正确：MarkingAdapter包装现有实现
  - [ ] 单元测试通过：`go test ./internal/governance/...`

  **QA Scenarios**:
  ```
  Scenario: 字段标记查询
    Tool: Bash (go test)
    Preconditions: PostgreSQL运行中，classification表有数据
    Steps:
      1. go test ./internal/governance/... -run TestMarkingAdapter -v
      2. 检查GetFieldMarking返回正确分类
      3. 检查IsLLMAllowed正确判断PII字段
    Expected Result: 测试全部通过
    Evidence: .sisyphus/evidence/task-7-marking-service.txt
  ```

  **Commit**: YES
  - Message: `feat(governance): add MarkingService interface and adapter`
  - Files: `internal/governance/marking_service.go`, `internal/governance/marking_adapter.go`
  - Pre-commit: `go test ./internal/governance/...`

- [x] 8. DecisionLineageService接口+适配器

  **What to do**:
  - 创建internal/decision/lineage_service.go（接口定义）
  - 实现DecisionLineageAdapter（包装现有LineageService+DecisionRepository）
  - 实现GetDecisionLineage、GetContextLineage、RecordDecisionLineage方法
  - 实现从alert到action的完整lineage链
  - 添加单元测试

  **Must NOT do**:
  - 不修改现有LineageService接口
  - 不破坏现有lineage查询

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 9)
  - **Blocks**: Tasks 10, 14-18
  - **Blocked By**: Tasks 1, 4

  **References**:
  - `internal/governance/lineage.go` - 现有LineageService
  - `internal/repository/decision_repository.go` - DecisionRepository
  - `internal/decision/case_service.go` - CaseService
  - `config/data_lineage.yml` - 血缘定义

  **Acceptance Criteria**:
  - [ ] 接口定义完整：DecisionLineageService接口包含所有方法
  - [ ] 适配器实现正确：DecisionLineageAdapter包装现有实现
  - [ ] 单元测试通过：`go test ./internal/decision/...`

  **QA Scenarios**:
  ```
  Scenario: 决策血缘查询
    Tool: Bash (go test)
    Preconditions: PostgreSQL运行中，decision_case有数据
    Steps:
      1. go test ./internal/decision/... -run TestDecisionLineageAdapter -v
      2. 检查GetDecisionLineage返回完整链
      3. 检查GetContextLineage返回源表列表
    Expected Result: 测试全部通过
    Evidence: .sisyphus/evidence/task-8-lineage-service.txt
  ```

  **Commit**: YES
  - Message: `feat(decision): add DecisionLineageService interface and adapter`
  - Files: `internal/decision/lineage_service.go`, `internal/decision/lineage_adapter.go`
  - Pre-commit: `go test ./internal/decision/...`

- [x] 9. YAML种子数据迁移

  **What to do**:
  - 创建migrations/020_seed_ontology_data.sql（从aip_object_schema.yml导入）
  - 创建migrations/021_seed_marking_data.sql（从data_markings.yml导入）
  - 创建migrations/022_seed_lineage_data.sql（从data_lineage.yml导入）
  - 使用ON CONFLICT实现幂等插入
  - 编写DOWN脚本（TRUNCATE）

  **Must NOT do**:
  - 不修改现有YAML文件
  - 不丢失现有gov表数据

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 2 (with Tasks 6, 7, 8)
  - **Blocks**: Tasks 14-18
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `config/aip_object_schema.yml` - 对象类型定义
  - `config/data_markings.yml` - Marking定义
  - `config/data_lineage.yml` - 血缘定义
  - `migrations/006_gov_tables.sql` - 现有gov表参考

  **Acceptance Criteria**:
  - [ ] 种子数据成功：`goose up`无错误
  - [ ] 数据完整性：`SELECT COUNT(*) FROM gov.object_type_registry`返回8
  - [ ] DOWN脚本工作：`goose down`成功清空数据

  **QA Scenarios**:
  ```
  Scenario: 种子数据验证
    Tool: Bash (psql)
    Preconditions: 迁移017-019已执行
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" up
      2. psql -c "SELECT COUNT(*) FROM gov.object_type_registry;"
      3. psql -c "SELECT COUNT(*) FROM gov.marking_definition;"
      4. psql -c "SELECT COUNT(*) FROM gov.lineage_node;"
    Expected Result: 数据正确导入
    Evidence: .sisyphus/evidence/task-9-seed-data.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): seed ontology/marking/lineage data from YAML`
  - Files: `migrations/020_seed_ontology_data.sql`, `migrations/021_seed_marking_data.sql`, `migrations/022_seed_lineage_data.sql`
  - Pre-commit: `goose status`

- [x] 10. ContextBuilderV2实现

  **What to do**:
  - 创建internal/decision/context_builder_v2.go
  - 实现ContextBuilderV2结构体（使用OntologyAwareRepo+MarkingService+DecisionLineageService）
  - 实现BuildDecisionContext方法（使用新接口）
  - 实现场景：通过特性标志切换新旧ContextBuilder
  - 添加单元测试

  **Must NOT do**:
  - 不修改现有ContextBuilder实现
  - 不破坏现有决策上下文构建

  **Recommended Agent Profile**:
  - **Category**: `deep`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 11, 12, 13)
  - **Blocks**: Tasks 14-18
  - **Blocked By**: Tasks 1, 5, 6, 7, 8

  **References**:
  - `internal/decision/context_builder.go` - 现有ContextBuilder实现
  - `internal/repository/ontology_aware_repo.go` - 新OntologyAwareRepo接口
  - `internal/governance/marking_service.go` - 新MarkingService接口
  - `internal/decision/lineage_service.go` - 新DecisionLineageService接口

  **Acceptance Criteria**:
  - [ ] 实现完整：ContextBuilderV2包含所有方法
  - [ ] 特性标志切换：通过USE_NEW_CONTEXT_BUILDER控制
  - [ ] 单元测试通过：`go test ./internal/decision/...`

  **QA Scenarios**:
  ```
  Scenario: 新ContextBuilder验证
    Tool: Bash (go test)
    Preconditions: PostgreSQL运行中，所有新接口可用
    Steps:
      1. go test ./internal/decision/... -run TestContextBuilderV2 -v
      2. 检查BuildDecisionContext返回正确上下文
      3. 检查使用新接口（OntologyAwareRepo等）
    Expected Result: 测试全部通过
    Evidence: .sisyphus/evidence/task-10-context-builder-v2.txt
  ```

  **Commit**: YES
  - Message: `feat(decision): add ContextBuilderV2 with new interfaces`
  - Files: `internal/decision/context_builder_v2.go`
  - Pre-commit: `go test ./internal/decision/...`

- [x] 11. 向后兼容视图

  **What to do**:
  - 创建migrations/023_backward_compat_views.sql
  - 创建gov.v_object_types视图（合并object_type_registry+object_property）
  - 创建gov.v_marking_assignments视图（合并marking_definition+marking_assignment）
  - 创建gov.v_lineage_graph视图（合并lineage_node+lineage_edge）
  - 创建gov.v_governance_summary视图（合并classification+marking）
  - 编写DOWN脚本

  **Must NOT do**:
  - 不修改现有表结构
  - 不影响现有查询

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 12, 13)
  - **Blocks**: Tasks 14-18
  - **Blocked By**: Tasks 2, 3, 4

  **References**:
  - `migrations/017_ontology_tables.sql` - ontology表
  - `migrations/018_marking_tables.sql` - marking表
  - `migrations/019_decision_lineage.sql` - lineage表

  **Acceptance Criteria**:
  - [ ] 视图创建成功：`goose up`无错误
  - [ ] 视图可查询：`SELECT COUNT(*) FROM gov.v_object_types`返回数据
  - [ ] DOWN脚本工作：`goose down`成功删除视图

  **QA Scenarios**:
  ```
  Scenario: 视图验证
    Tool: Bash (psql)
    Preconditions: 迁移017-022已执行
    Steps:
      1. goose -dir migrations postgres "$DATABASE_URL" up
      2. psql -c "SELECT COUNT(*) FROM gov.v_object_types;"
      3. psql -c "SELECT COUNT(*) FROM gov.v_marking_assignments;"
      4. psql -c "SELECT COUNT(*) FROM gov.v_lineage_graph;"
    Expected Result: 视图返回正确数据
    Evidence: .sisyphus/evidence/task-11-compat-views.txt
  ```

  **Commit**: YES
  - Message: `feat(compat): add backward compatibility views`
  - Files: `migrations/023_backward_compat_views.sql`
  - Pre-commit: `goose status`

- [x] 12. 验证脚本

  **What to do**:
  - 创建scripts/verification/verify_phase1.sh（阶段1验证）
  - 创建scripts/verification/verify_phase2.sh（阶段2验证）
  - 创建scripts/verification/verify_phase3.sh（阶段3验证）
  - 创建scripts/verification/verify_phase4.sh（阶段4验证）
  - 创建scripts/verification/verify_phase5.sh（阶段5验证）
  - 添加Makefile targets: `make verify-phase1`到`make verify-phase5`

  **Must NOT do**:
  - 不修改现有测试框架
  - 不破坏现有CI流程

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 11, 13)
  - **Blocks**: Tasks 14-18
  - **Blocked By**: None

  **References**:
  - `scripts/migration/compare_pipeline_baseline.py` - 现有对比脚本
  - `scripts/migration/compare_api_baseline.py` - API对比脚本
  - `Makefile` - 现有targets

  **Acceptance Criteria**:
  - [ ] 脚本可执行：`chmod +x scripts/verification/*.sh`
  - [ ] Makefile targets工作：`make verify-phase1`执行成功
  - [ ] 验证逻辑正确：检查所有关键指标

  **QA Scenarios**:
  ```
  Scenario: 验证脚本执行
    Tool: Bash (make)
    Preconditions: 无
    Steps:
      1. make verify-phase1
      2. 检查输出显示所有检查通过
      3. make verify-phase2
    Expected Result: 验证脚本执行成功
    Evidence: .sisyphus/evidence/task-12-verification.txt
  ```

  **Commit**: YES
  - Message: `feat(verification): add phase verification scripts`
  - Files: `scripts/verification/*.sh`, `Makefile`
  - Pre-commit: `make verify-phase1`

- [x] 13. 回滚脚本

  **What to do**:
  - 创建scripts/backup/backup_pg.sh（备份脚本）
  - 创建scripts/backup/restore_pg.sh（恢复脚本）
  - 创建scripts/rollback/rollback_phase.sh（回滚脚本）
  - 创建docs/rollback-runbook.md（回滚手册）
  - 添加Makefile targets: `make backup`, `make restore`, `make rollback`

  **Must NOT do**:
  - 不自动执行回滚（需手动确认）
  - 不删除现有备份

  **Recommended Agent Profile**:
  - **Category**: `quick`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES
  - **Parallel Group**: Wave 3 (with Tasks 10, 11, 12)
  - **Blocks**: Tasks 14-18
  - **Blocked By**: None

  **References**:
  - `docker-compose.yml` - Docker配置
  - `Makefile` - 现有targets
  - `migrations/` - 迁移文件

  **Acceptance Criteria**:
  - [ ] 脚本可执行：`chmod +x scripts/backup/*.sh scripts/rollback/*.sh`
  - [ ] Makefile targets工作：`make backup`执行成功
  - [ ] 文档完整：`docs/rollback-runbook.md`包含所有场景

  **QA Scenarios**:
  ```
  Scenario: 备份脚本执行
    Tool: Bash (make)
    Preconditions: PostgreSQL运行中
    Steps:
      1. make backup
      2. 检查backups/目录生成.dump文件
      3. make restore BACKUP=backups/baxi_pre_phase_1.dump
    Expected Result: 备份恢复成功
    Evidence: .sisyphus/evidence/task-13-rollback.txt
  ```

  **Commit**: YES
  - Message: `feat(rollback): add backup and rollback scripts`
  - Files: `scripts/backup/*.sh`, `scripts/rollback/*.sh`, `docs/rollback-runbook.md`, `Makefile`
  - Pre-commit: `make backup`

- [x] 14. 阶段1验证（并行运行）

  **What to do**:
  - 验证Go API和Python API并行运行
  - 验证新表创建成功，数据迁移完成
  - 验证新接口实现正确，适配器工作正常
  - 验证特性标志系统工作正常
  - 运行完整测试套件（Go + Python）
  - 运行阶段1验证脚本

  **Must NOT do**:
  - 不切换前端流量
  - 不修改Python API

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Wave 3)
  - **Blocks**: Task 15
  - **Blocked By**: Tasks 5, 6, 7, 8, 9, 10, 11, 12, 13

  **References**:
  - `scripts/verification/verify_phase1.sh` - 阶段1验证脚本
  - `Makefile` - verify-phase1 target
  - `internal/feature/flags.go` - 特性标志

  **Acceptance Criteria**:
  - [ ] Go API正常运行：`curl http://localhost:8080/api/v1/health`返回200
  - [ ] Python API正常运行：`curl http://localhost:8765/api/v1/health`返回200
  - [ ] 新表有数据：`psql -c "SELECT COUNT(*) FROM gov.object_type_registry"` > 0
  - [ ] 测试全部通过：`make test`和`pytest`
  - [ ] 阶段1验证通过：`make verify-phase1`

  **QA Scenarios**:
  ```
  Scenario: 并行运行验证
    Tool: Bash (curl, make)
    Preconditions: Go API和Python API都在运行
    Steps:
      1. curl http://localhost:8080/api/v1/health
      2. curl http://localhost:8765/api/v1/health
      3. make verify-phase1
    Expected Result: 两个API都正常，验证通过
    Evidence: .sisyphus/evidence/task-14-phase1.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): phase 1 verification complete`
  - Files: `.sisyphus/evidence/task-14-phase1.txt`
  - Pre-commit: `make verify-phase1`

- [x] 15. 阶段2验证（Go为主读）

  **What to do**:
  - 切换前端到Go API（VITE_API_BASE_URL=http://localhost:8080）
  - 启用新接口（USE_ONTOLOGY_AWARE_REPO=true, USE_MARKING_SERVICE=true）
  - 验证前端读取正常
  - 运行影子模式测试（对比Go和Python响应）
  - 运行阶段2验证脚本

  **Must NOT do**:
  - 不修改Python API
  - 不关闭Python API

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 14)
  - **Blocks**: Task 16
  - **Blocked By**: Task 14

  **References**:
  - `frontend/.env` - 前端环境变量
  - `.env` - 后端环境变量
  - `scripts/verification/verify_phase2.sh` - 阶段2验证脚本

  **Acceptance Criteria**:
  - [ ] 前端连接Go API：`curl http://localhost:5173`返回正常页面
  - [ ] Go API响应正确：对比Go和Python响应一致
  - [ ] 影子模式无错误：日志无response mismatch警告
  - [ ] 阶段2验证通过：`make verify-phase2`

  **QA Scenarios**:
  ```
  Scenario: 前端读取验证
    Tool: Bash (curl, make)
    Preconditions: 前端已切换到Go API
    Steps:
      1. curl http://localhost:5173/console
      2. curl http://localhost:8080/api/v1/alerts
      3. make verify-phase2
    Expected Result: 前端正常，API响应正确
    Evidence: .sisyphus/evidence/task-15-phase2.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): phase 2 verification complete`
  - Files: `.sisyphus/evidence/task-15-phase2.txt`
  - Pre-commit: `make verify-phase2`

- [x] 16. 阶段3验证（双写）

  **What to do**:
  - 启用双写（USE_DUAL_WRITE=true）
  - 验证写操作同时写入Go和Python
  - 验证数据一致性
  - 运行阶段3验证脚本

  **Must NOT do**:
  - 不关闭Python API
  - 不切换写流量

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 15)
  - **Blocks**: Task 17
  - **Blocked By**: Task 15

  **References**:
  - `.env` - 环境变量
  - `scripts/verification/verify_phase3.sh` - 阶段3验证脚本
  - `internal/repository/dual_write_adapter.go` - 双写适配器

  **Acceptance Criteria**:
  - [ ] 双写启用：日志显示dual write enabled
  - [ ] 数据一致：对比Go和Python数据行数相同
  - [ ] 阶段3验证通过：`make verify-phase3`

  **QA Scenarios**:
  ```
  Scenario: 双写验证
    Tool: Bash (make)
    Preconditions: 双写已启用
    Steps:
      1. 触发写操作（如创建alert）
      2. 检查Go数据库有数据
      3. 检查Python数据库有数据
      4. make verify-phase3
    Expected Result: 数据一致，验证通过
    Evidence: .sisyphus/evidence/task-16-phase3.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): phase 3 verification complete`
  - Files: `.sisyphus/evidence/task-16-phase3.txt`
  - Pre-commit: `make verify-phase3`

- [x] 17. 阶段4验证（Go为主写）

  **What to do**:
  - 切换写流量到Go（USE_GO_PRIMARY_WRITE=true）
  - 验证写操作主要走Go
  - 验证Python降级为只读
  - 运行阶段4验证脚本

  **Must NOT do**:
  - 不关闭Python API
  - 不删除Python数据

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 16)
  - **Blocks**: Task 18
  - **Blocked By**: Task 16

  **References**:
  - `.env` - 环境变量
  - `scripts/verification/verify_phase4.sh` - 阶段4验证脚本
  - `api/main.py` - Python API（只读模式）

  **Acceptance Criteria**:
  - [ ] Go写入成功：写操作返回200
  - [ ] Python只读：写操作返回403 read_only
  - [ ] 阶段4验证通过：`make verify-phase4`

  **QA Scenarios**:
  ```
  Scenario: Go为主写验证
    Tool: Bash (curl, make)
    Preconditions: Go为主写已启用
    Steps:
      1. curl -X POST http://localhost:8080/api/v1/pipeline/run
      2. curl -X POST http://localhost:8765/api/v1/pipeline/run
      3. make verify-phase4
    Expected Result: Go写入成功，Python返回只读
    Evidence: .sisyphus/evidence/task-17-phase4.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): phase 4 verification complete`
  - Files: `.sisyphus/evidence/task-17-phase4.txt`
  - Pre-commit: `make verify-phase4`

- [x] 18. 阶段5验证（Python退役）

  **What to do**:
  - 停止Python API服务
  - 移除Python依赖
  - 更新前端只连接Go API
  - 清理代码（标记Python为deprecated）
  - 运行阶段5验证脚本

  **Must NOT do**:
  - 不删除Python代码（保留历史）
  - 不删除SQLite数据（保留备份）

  **Recommended Agent Profile**:
  - **Category**: `unspecified-high`
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Sequential (after Task 17)
  - **Blocks**: F1-F4
  - **Blocked By**: Task 17

  **References**:
  - `docker-compose.yml` - Docker配置
  - `scripts/verification/verify_phase5.sh` - 阶段5验证脚本
  - `legacy/python-sqlite` - 遗留分支

  **Acceptance Criteria**:
  - [ ] Python API停止：`curl http://localhost:8765`返回连接拒绝
  - [ ] Go API正常：`curl http://localhost:8080/api/v1/health`返回200
  - [ ] 前端正常：`curl http://localhost:5173/console`返回正常页面
  - [ ] 阶段5验证通过：`make verify-phase5`

  **QA Scenarios**:
  ```
  Scenario: Python退役验证
    Tool: Bash (curl, make)
    Preconditions: Python API已停止
    Steps:
      1. curl http://localhost:8765/api/v1/health
      2. curl http://localhost:8080/api/v1/health
      3. make verify-phase5
    Expected Result: Python不可用，Go正常，验证通过
    Evidence: .sisyphus/evidence/task-18-phase5.txt
  ```

  **Commit**: YES
  - Message: `feat(migration): phase 5 verification complete - Python退役`
  - Files: `.sisyphus/evidence/task-18-phase5.txt`
  - Pre-commit: `make verify-phase5`

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [x] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [x] F2. **Code Quality Review** — `unspecified-high`
  Run `tsc --noEmit` + linter + `go test ./...` + `pytest`. Review all changed files for: `as any`/`@ts-ignore`, empty catches, console.log in prod, commented-out code, unused imports. Check AI slop: excessive comments, over-abstraction, generic names (data/result/item/temp).
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [x] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: empty state, invalid input, rapid actions. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [x] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Task 1-5**: `feat(auth): add JWT identity propagation` - internal/api/middleware/auth.go
- **Task 6-8**: `feat(ontology): add OntologyAwareRepo interface` - internal/repository/ontology_aware_repo.go
- **Task 9**: `feat(migration): seed ontology data from YAML` - migrations/020_seed_ontology_data.sql
- **Task 10**: `feat(decision): add ContextBuilderV2` - internal/decision/context_builder_v2.go
- **Task 11-13**: `feat(compat): add backward compatibility views` - migrations/023_backward_compat_views.sql
- **Task 14-18**: `feat(migration): progressive migration phase N` - various files

---

## Success Criteria

### Verification Commands
```bash
make test                    # Expected: all Go tests pass
pytest                       # Expected: all Python tests pass
make verify                  # Expected: all verification checks pass
goose -dir migrations postgres "$DATABASE_URL" status  # Expected: all migrations applied
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Frontend successfully connected to Go API
- [ ] Python API退役完成
- [ ] 回滚方案验证通过
