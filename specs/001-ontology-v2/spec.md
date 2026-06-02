# Feature Specification: Baxi Ontology v2 — Executable Semantic Layer

**Feature Branch**: `001-ontology-v2`  
**Created**: 2026-06-02  
**Status**: Draft  
**Input**: 将 Ontology 从对象描述层升级为可执行语义层，统一驱动查询、关系、指标、上下文和动作绑定。先以 seller_late_delivery_alert 跑通最小闭环。

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 — 运营分析师查看卖家异常上下文 (Priority: P1)

**作为** 运营分析师，**当** 收到卖家延迟发货异常告警时，**我希望** 系统能自动构建一个结构化的业务上下文，包含卖家属性、关键指标、最近订单和可用操作，**从而** 我可以快速理解问题根因并采取相应行动。

**Why this priority**: 这是 Ontology v2 的核心价值场景。当前系统虽然能查询对象和构建上下文，但缺少"按场景精确配方"的能力，导致 LLM 上下文过大或遗漏关键证据。

**Independent Test**: 可以通过一个卖家延迟发货告警 case，验证系统是否按 context recipe 返回精确的 seller 属性、3 个关键指标、最近 10 笔订单列表、以及 3 个可用操作（notify_owner / create_followup_task / export_report）。

**Acceptance Scenarios**:

1. **Given** 一个 `seller_late_delivery_alert` 类型的告警 case，**When** 系统构建决策上下文，**Then** 上下文中包含卖家的 seller_state、seller_city、gmv、order_count、avg_review_score、late_delivery_rate 属性
2. **Given** 同一个 case，**When** 上下文构建完成，**Then** 包含 3 个指标（late_delivery_rate_7d、order_count_7d、gmv_7d）及其当前值、基线值、变化幅度
3. **Given** 同一个 case，**When** 上下文构建完成，**Then** 包含最近 10 笔订单的 order_id、order_status、payment_value、review_score、delivery_status
4. **Given** 同一个 case，**When** 上下文构建完成，**Then** 可用操作列表精确包含 notify_owner、create_followup_task、export_report
5. **Given** 同一个 case，**When** 上下文构建完成，**Then** 包含治理摘要（脱敏字段数、应用角色、字段分类级别）

---

### User Story 2 — 通过 Ontology Schema 动态查询卖家 (Priority: P1)

**作为** 系统开发者，**我希望** 对象查询不再依赖硬编码的表映射，而是由 Ontology schema 统一驱动，**从而** 新增或修改对象属性时只需更新 schema 配置，无需修改代码。

**Why this priority**: 当前 `objectTableMap` 和 `metricColumns` 是硬编码的 Go 代码，与 YAML schema 配置分离。这是 Ontology v2 的基础能力，直接影响系统的可维护性和扩展性。

**Independent Test**: 可以通过 seller 对象的查询和指标获取，验证 SQL 是否由 Ontology schema 动态编译生成，而非来自硬编码映射。

**Acceptance Scenarios**:

1. **Given** Ontology v2 schema 中定义了 seller 对象的 source（schema=dwd, table=item_level, primary_key=seller_id）和 properties（含 gmv 的 agg=sum、order_count 的 agg=count_distinct），**When** 查询 seller 对象，**Then** 生成的 SQL 包含正确的 SELECT 列、聚合函数和 WHERE 条件
2. **Given** seller schema 中 seller_state 和 seller_city 标记为 filterable=true，**When** 按 state/city 过滤搜索卖家，**Then** 查询成功返回结果
3. **Given** 未知对象类型（如 "unknown_type"），**When** 尝试查询，**Then** 系统返回明确的错误信息
4. **Given** 非 filterable 字段（如未标记 filterable 的字段），**When** 尝试用该字段过滤，**Then** 系统拒绝并返回权限错误

---

### User Story 3 — 通过关系链路查看卖家的最近订单 (Priority: P2)

**作为** 运营分析师，**当** 查看卖家详情时，**我希望** 能看到该卖家的最近订单列表（一对多关系），**从而** 了解该卖家的具体交易行为。

**Why this priority**: 当前 ObjectLink 的 Via 模型只能表达一对一关系，无法处理 seller → recent_orders 这类一对多关系。这是 Ontology v2 的关键扩展能力。

**Independent Test**: 可以通过指定 seller_id 和 link_name="recent_orders"，验证返回的是订单列表而非单个对象。

**Acceptance Scenarios**:

1. **Given** seller 对象定义了 recent_orders 关系（reverse_lookup 策略，目标 order，按 order_purchase_timestamp desc 排序，limit=20），**When** 查询 seller_123 的 recent_orders，**Then** 返回最多 20 条订单记录
2. **Given** 返回的订单列表，**When** 检查字段，**Then** 每条记录包含 order_id、order_status、payment_value、review_score、delivery_status
3. **Given** 无效的 link_name（如 "nonexistent_link"），**When** 查询，**Then** 系统返回错误提示该关系不存在
4. **Given** link 配置 max_depth=2，**When** 查询深层关系，**Then** 系统在深度 2 处停止遍历，防止无限递归

---

### User Story 4 — 安全的对象级动作执行 (Priority: P2)

**作为** 运营分析师，**当** 我需要对异常卖家执行通知操作时，**我希望** 系统能校验该操作是否已绑定到该对象类型，并且高风险操作需要审批，**从而** 避免误操作或未经授权的执行。

**Why this priority**: 当前 action binding 是全局的，没有对象级别的显式绑定。Ontology v2 需要在对象 schema 中定义允许的动作，并在执行前进行完整校验。

**Independent Test**: 可以通过 seller 上的 notify_owner 操作，验证对象级绑定、payload 校验、审批流程和 dry-run 行为。

**Acceptance Scenarios**:

1. **Given** seller 对象的 schema 中定义了 actions: [notify_owner, create_followup_task, export_report]，**When** 尝试对 seller 执行 notify_owner，**Then** 操作被允许
2. **Given** seller 对象的 schema 中未定义 escalate_to_human，**When** 尝试对 seller 执行 escalate_to_human，**Then** 操作被拒绝
3. **Given** notify_owner 在 action_registry 中标记为 requires_approval=true，**When** 直接执行（未经审批），**Then** 系统拒绝并提示需要审批
4. **Given** notify_owner 的 payload 缺少 required 字段（如 owner_role），**When** 尝试执行，**Then** 系统返回 payload 校验错误
5. **Given** 所有校验通过，**When** 执行 notify_owner，**Then** 系统默认以 dry-run 模式运行，不写入真实外部系统

---

### Edge Cases

- **Schema 不完整**: 当对象缺少 source.schema/source.table/source.primary_key 时，加载应失败并给出明确的验证错误
- **重复主键**: 当对象的 properties 中有多个 is_pk=true 时，验证应报错
- **循环关系**: 当对象 A 链接到 B，B 又链接回 A 时，link traversal 应在 max_depth 处停止，避免无限循环
- **Metric 缺失**: 当 property 引用了 metric_ref 但该 metric 在 metric_definitions.yml 中不存在时，验证应报错
- **Action registry 禁用**: 当 action_registry.yml 中禁用了某个 action，即使对象 schema 中绑定了该 action，执行也应被拒绝
- **LLM 不可读字段脱敏**: 当 property 标记为 llm_readable=false 时，该字段不能出现在 LLM 上下文中，且 redaction_summary 应记录该脱敏
- **Token 预算超限**: 当 context recipe 配置的总对象数超过 max_objects 或预估 token 超过 max_tokens_hint 时，系统应截断或报错
- **Recipe 缺失**: 当告警 rule_id 没有匹配到任何 recipe 时，系统回退到对象类型的默认 recipe（包含所有 llm_readable 属性和 metrics），不报错
- **Metric 定义重复**: properties 中 agg=sum（实时 OLAP）和 metric_definitions 中 gmv_7d（预聚合）语义不同，允许共存，QueryCompiler 和 MetricResolver 各自使用对应的定义
- **v1 对象查询**: 当查询未迁移到 v2 的对象类型时，系统自动使用 v1 的 objectTableMap 硬编码逻辑，对调用方透明

---

## Clarifications

### Session 2026-06-02

- **Q**: v1/v2 共存策略 → **A**: 按对象独立选择。v2 schema 定义了的对象走 v2（QueryCompiler/LinkResolver/ContextRecipe），未定义的走 v1 fallback（objectTableMap 硬编码）。允许渐进式迁移，seller 先验证，其他对象后迁。
- **Q**: Context Recipe 的覆盖策略 → **A**: 按对象类型回退到默认 recipe。recipe 未定义时，使用对象类型的默认 recipe（包含所有 llm_readable 属性和 metrics），避免严格模式导致新告警类型无法工作。
- **Q**: Metric 数据源设计 → **A**: 分层设计。properties 中 agg/expression 用于实时 OLAP 查询（如 seller gmv=SUM(price)），metric_definitions 用于预聚合指标（如 gmv_7d 来自 mart.metric_dimension_daily）。两处语义不同，不视为重复。
- **Q**: QueryCompiler 安全策略 → **A**: 白名单校验 + 硬上限。filter 字段必须 filterable=true，sort 字段必须 searchable=true，limit 硬上限 10000。schema/table/column 必须来自 ontology config，禁止任意 SQL 注入。
- **Q**: v1 废弃策略 → **A**: 渐进废弃。v2 稳定运行 1-2 个 release 后标记 v1 deprecated，再 1 个 release 后删除。过渡期同时维护两套代码。

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统必须支持扩展的 Ontology v2 schema，包含对象级 source（schema/table/primary_key）、properties（含 type/source/agg/expression/metric_ref/sensitivity/llm_readable/searchable/filterable/is_pk）、metrics（引用外部 metric_definitions）、relationships（含 cardinality/strategy/source_key/target/limit/sort/fields）、actions（对象级动作绑定列表）和 governance（default_role/redact_pii）
- **FR-002**: 系统必须能够从 Ontology v2 schema 动态编译对象查询（QueryCompiler），替代硬编码的 objectTableMap，生成安全的 SQL（SELECT 列、聚合、WHERE、GROUP BY、LIMIT）。filter 字段必须 filterable=true，sort 字段必须 searchable=true，limit 硬上限 10000
- **FR-003**: 系统必须支持四类关系解析策略：direct_key（源字段直接指向目标主键）、reverse_lookup（目标表含 source key）、bridge_table（通过中间表关联）、query_ref（预定义查询模板）
- **FR-004**: 系统必须支持独立的 Metric Contract 定义（metric_definitions.yml），包含指标的数据源、过滤条件、值列、基线列、严重级别规则和 LLM 解释
- **FR-005**: 系统必须支持 Context Recipe 驱动的上下文构建，根据告警 rule_id 匹配 recipe，按 recipe 精确选择 root properties、metrics、links 和 actions，并应用 governance 和脱敏策略。当 recipe 未定义时，回退到对象类型的默认 recipe（包含所有 llm_readable 属性和 metrics）
- **FR-006**: 系统必须支持对象级 Action Binding，在执行动作前按顺序校验：object_type 存在 → action 绑定到该对象 → action 在全局 registry 中启用 → 当前 role 有权限 → payload 符合 schema → 若 requires_approval=true 则已审批 → 默认 dry-run
- **FR-007**: 系统必须保持 Ontology v1 的向后兼容性，旧 YAML 继续可加载，新 YAML 优先走 v2 parser，ObjectRegistry 同时支持 v1/v2。v2 schema 定义了的对象走 v2（QueryCompiler/LinkResolver/ContextRecipe），未定义的走 v1 fallback（objectTableMap 硬编码）
- **FR-008**: MCP 工具必须增强 describe_ontology、get_linked_objects 输出，新增 build_context（按 case_id + recipe_id 构建上下文）和 propose_action（创建 ActionProposal 不直接执行）
- **FR-009**: 系统必须对所有核心路径提供测试覆盖，包括 schema validation、QueryCompiler SQL 生成、LinkResolver 关系解析、ContextRecipe 上下文构建、ActionBinding 安全校验
- **FR-010**: 系统必须为 seller、order、product、metric_alert 至少 4 个对象提供完整的 Ontology v2 schema 定义，并以 seller_late_delivery_alert 场景跑通最小闭环

### Key Entities

- **ObjectTypeV2**: 业务对象的完整语义定义，是查询、关系、指标、上下文和动作的统一源头
- **MetricDefinition**: 指标合同，定义指标的计算逻辑、数据源、严重级别和解释说明
- **ContextRecipe**: 上下文配方，定义特定告警场景下需要包含哪些对象属性、指标、关联证据和可用操作
- **CompiledQuery**: 从 Ontology schema 编译出的安全查询计划，包含 SQL、参数、列映射
- **ObjectLinkV2**: 对象关系定义，支持多种 cardinality 和解析策略
- **ActionProposal**: 动作提案，包含目标对象、操作类型、payload、风险级别和审批状态

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: seller、order、product、metric_alert 至少 4 个对象由 Ontology v2 schema 驱动，新增或修改对象属性时无需修改代码
- **SC-002**: seller 对象查询不再依赖硬编码的 objectTableMap，查询性能与硬编码方案相比差异不超过 10%
- **SC-003**: seller → recent_orders 关系能正确返回最多 20 条订单记录（一对多），而非当前只能返回单个对象
- **SC-004**: seller_late_delivery_alert 场景的 LLM 上下文包含完整的证据链（seller 属性 + 3 个指标 + 最近订单 + 可用操作 + 治理摘要），上下文构建时间不超过 2 秒
- **SC-005**: 未绑定到对象类型的 action 执行请求 100% 被拒绝，requires_approval=true 的 action 未经审批直接执行 100% 被拒绝
- **SC-006**: 所有核心路径（schema validation、QueryCompiler、LinkResolver、ContextRecipe、ActionBinding）的测试覆盖率不低于 80%
- **SC-007**: MCP 侧可以完整描述 Ontology 结构（含 v2 扩展字段），并按 case_id 构建带 recipe 的上下文
- **SC-008**: 旧版 aip_object_schema.yml（v1 格式）继续可加载，系统运行不受影响。v1 代码在 v2 稳定运行 1-2 个 release 后标记 deprecated，再 1 个 release 后安全删除
