# Baxi 项目技术报告大纲

> **主线**：从 Palantir AIP 的本体论（Ontology）理念出发，解释 Baxi 的设计哲学、架构取舍与实现路径。

---

## 摘要（1 页）

**一句话总结**：Baxi 是一个借鉴 Palantir AIP 本体论理念的数据治理与决策平台，将业务实体（Seller、Product、Alert）作为一等公民，通过管道构建、LLM 决策、动作执行形成闭环。

**关键指标**：
- 9 种本体对象类型，112K+ Seller 实体
- 7 步 ETL 管道，31 个 MCP 工具
- 端到端闭环：告警 → 案例 → 提案 → 审批 → 执行 → 结案

---

## 第 1 章 引言：为什么借鉴 Palantir AIP（3 页）

### 1.1 传统数据平台的困境
- 关系数据库模型：表是存储单元，业务语义隐藏在 SQL join 中
- BI 工具的局限：分析师需要理解表结构才能提问
- 动作执行的断层：发现洞察后无法直接触发业务动作

### 1.2 Palantir AIP 的本体论洞见
- **Objects 作为一等公民**：Seller、Order、Alert 不是表，是业务实体
- **Links 显性化关系**： seller → order → product 的关联是本体的一部分
- **Actions 绑定对象**：notify_owner(seller)、escolate(alert) 是对象的方法
- **Ontology 即代码**：业务模型驱动数据架构，而非反向适配

### 1.3 Baxi 的设计目标
- 构建一个可演示的、本体论驱动的数据治理闭环
- 让 AI Agent（Pi）能基于业务实体自主决策和动作
- 取舍：不求企业级规模，但求概念完整、链路贯通

---

## 第 2 章 Ontology 设计：业务实体的一等公民化（4 页）

### 2.1 对象类型体系
```
seller          —— 业务核心实体（112K+ 实例）
product         —— 商品实体
metric_alert    —— 告警实体（动态生成）
global          —— 全局指标实体
campaign        —— 营销实体
order           —— 订单实体
order_item      —— 订单明细
review          —— 审核记录
outbox_event    —— 待分发事件
```

### 2.2 ObjectTypeV2 Schema 定义
- `primary_key`：对象唯一标识
- `properties`：属性列表（类型、可空性、分类级别）
- `links`：关联对象定义
- `actions`：对象支持的操作列表

### 2.3 为什么放弃纯关系模型
- **传统做法**：`SELECT * FROM dwd.dim_seller_v2 WHERE seller_id = 'xxx'`
- **本体做法**：`ontology.GetObject("seller", "xxx")` → 返回带属性、关联、动作的对象
- **收益**：业务语义内聚，Agent 无需理解表结构即可操作

### 2.4 取舍：静态 Schema vs 动态演化
- 选择：静态 YAML 定义（`config/aip_object_schema.yml`）
- 理由：演示阶段需要确定性，动态 schema 增加复杂度
- 代价：新增对象类型需要改配置、重启服务

---

## 第 3 章 Pipeline：从原始数据到本体对象的转化（4 页）

### 3.1 Ontology 驱动的 ETL 设计
- 管道不是通用 ETL，是**本体构建器**
- 每一步都明确产出什么对象类型

```
raw.listings      → ingest_raw       → 原始数据录入
raw.orders        → build_dwd        → seller/order/order_item 对象构建
mart.metrics      → build_metrics    → global 对象指标计算
ops.alert         → detect_alerts    → metric_alert 对象生成
ops.recommendation → generate_recommendations → 建议对象
ops.task          → generate_tasks   → 任务对象
ops.outbox_event  → create_outbox    → 事件对象（待分发）
```

### 3.2 语义层级：raw → dwd → mart
- **raw**：原始数据，无业务语义
- **dwd**：数据仓库明细，对象属性标准化
- **mart**：数据集市，对象指标聚合
- **ops**：运营层，对象的动作触发

### 3.3 取舍：批处理 vs 实时流
- 选择：CSV 批处理（`./data/raw/*.csv`）
- 理由：演示可控、易复现、无需流式基础设施
- 代价：数据新鲜度低，无法演示实时告警

### 3.4 Pipeline 与 Ontology 的衔接点
- `IngestRawStep` 读取 CSV → 写入 raw 表
- `BuildDWD*` 步骤将 raw 数据转化为对象属性
- `DetectAlertsStep` 基于对象指标生成告警对象

---

## 第 4 章 Decision Engine：基于本体上下文的智能决策（5 页）

### 4.1 传统决策系统的问题
- 规则引擎：硬编码 if-else，无法处理未知场景
- 纯 LLM：幻觉风险，无法解释决策依据
- Baxi 的解法：**Ontology 上下文 + LLM 生成 + 规则校验**

### 4.2 LLMSafeContext 的三层构建
```
Trigger 层：告警对象 + 触发条件 + 当前值/基线值
Object  层：目标对象（seller）的完整属性 + 关联对象（product/order）
Governance 层：数据分类、访问策略、允许/禁止的动作
```

### 4.3 ContextBuilderV2 的 Ontology 感知
- 从 `ai.decision_case` 获取案例 → 解析 object_type / object_id
- 调用 `ontologyRepo.GetObjectByID()` → 获取对象实例
- 调用 `governance.MarkingService` → 附加分类信息
- 调用 `actionRegistry` → 确定允许的动作类型

### 4.4 决策流程
```
alert(metric_alert) → create_case → build_context
  → decide(LLM / rule fallback) → proposals
    → [notify_owner, create_task, apply_tag, escalate]
```

### 4.5 取舍：LLM 成本 vs 规则覆盖
- 设计：LLM 为主要决策器，规则引擎为 fallback
- 理由：LLM 处理复杂场景，规则保证确定性边界
- 安全阀：`BAXI_ALLOW_LIVE_EXECUTION` 环境变量控制真实执行

### 4.6 决策的可解释性
- `decision_lineage` 表记录完整决策链路
- `llm_decision` 表保存原始输入/输出/验证结果
- Agent 可以追溯：为什么推荐 notify_owner？因为 seller 的 revenue_drop 超过阈值。

---

## 第 5 章 Action System：本体绑定的操作执行（4 页）

### 5.1 Action Registry：对象操作的类型系统
```
notify_owner     —— 通知对象 owner（Feishu/邮件）
create_task      —— 创建跟踪任务
apply_tag        —— 给对象打标签
escalate         —— 升级告警级别
```

### 5.2 为什么只有 4 种动作
- Palantir AIP 允许任意动作，Baxi 限制为 4 种标准类型
- 理由：演示阶段需要安全边界，防止 Agent 执行危险操作
- 取舍：灵活性换取安全性

### 5.3 Proposal → Approval → Execution 状态机
```
draft → pending_review → approved → executing → completed
           ↓ reject
      rejected
```

### 5.4 沙箱机制：操作的安全验证
- `create_sandbox(case_id)` → 隔离测试环境
- `add_to_sandbox(proposal_id)` → 将提案放入沙箱
- `compare_sandboxes()` → 对比不同方案影响
- 取舍：沙箱是逻辑隔离（DB 记录），非物理隔离

### 5.5 执行适配器
- FeishuAdapter：飞书 webhook 通知
- GitHubAdapter：GitHub issue 创建
- CLIAdapter：本地命令执行
- ManualAdapter：人工确认执行

---

## 第 6 章 Governance：本体的元数据治理（3 页）

### 6.1 为什么治理要嵌入本体层
- 传统做法： governance 是外部系统（如 Apache Atlas）
- Baxi 做法： classification + lineage + access_policy 是对象属性的一部分
- 收益：Agent 查询对象时自动获取治理信息

### 6.2 数据分类（Classification）
- 字段级别：PII、财务、运营、公开
- 对象级别：seller 为 PII，metric_alert 为运营
- 动态标记：基于规则自动分类

### 6.3 血缘追踪（Lineage）
- `dwd.dim_seller_v2` ← `raw.listings` ← `data/raw/sellers.csv`
- 对象属性级血缘：知道每个字段的数据来源

### 6.4 访问策略（Access Policy）
- 角色 × 对象类型 × 动作 = 是否允许
- Agent 操作前自动校验：`check_access(role, object_type, action)`

---

## 第 7 章 MCP 接口：本体能力的对外暴露（3 页）

### 7.1 为什么用 MCP 而不是 REST API
- MCP（Model Context Protocol）是 AI Agent 的标准接口
- 31 个工具 = 31 种本体操作
- stdio 传输：本地进程通信，低延迟、无网络依赖

### 7.2 工具映射：本体操作的标准化
```
decision 域：create_case / decide / resolve_case / list_cases / get_case / list_proposals
ontology  域：describe_ontology / get_object / get_linked_objects / execute_action
action    域：execute_proposal / get_decision_context
review    域：approve_proposal / reject_proposal / get_proposal_by_id
```

### 7.3 Pi Agent 的交互模式
- Agent 无需理解 SQL 或表结构
- Agent 通过自然语言意图调用工具："为 seller-123 创建一个 revenue_drop 的告警案例"
- 工具返回结构化 JSON，Agent 基于结果做下一步决策

### 7.4 取舍：stdio 的局限性
- 只能本地运行，无法远程连接
- 无内置认证（依赖 `BAXI_MCP_USER_ID` 环境变量）
- 理由：演示阶段简化部署，生产环境可扩展为 SSE/WebSocket

---

## 第 8 章 验证与测试：本体驱动的端到端验证（3 页）

### 8.1 E2E 测试设计哲学
- 测试不是验证 API 返回值，是验证**本体状态转换**
- 一个完整闭环 = 对象状态的完整生命周期

### 8.2 标准测试路径
```
metric_alert(告警对象)
  → create_decision_case → ai.decision_case(案例对象)
    → decide → action_proposal(提案对象)
      → approve_proposal → review_record(审批记录)
        → execute_proposal(dry_run) → execution_result(执行结果)
          → resolve_case → case closed
```

### 8.3 自适应测试
- 无告警时：从 ontology 搜索 seller 对象，自动生成测试告警
- Schema 缺失时：`--auto-fix` 自动添加缺失列
- LLM 失败时：回退到 SQL 手动创建提案

### 8.4 测试结果（当前状态）
- 14 通过 / 2 失败 / 2 警告 / 1 跳过
- 失败项：global 对象类型未注册、ops.event_outbox 表缺失
- 本体搜索：112,652 seller 对象可用

---

## 第 9 章 取舍与局限：为什么这样设计（3 页）

### 9.1 架构层面
| 取舍 | 选择 | 放弃 | 理由 |
|------|------|------|------|
| 对象模型 | 静态 YAML 定义 | 动态 schema 演进 | 演示确定性 |
| 数据流 | 批处理 CSV | 实时流（Kafka/Flink） | 部署简化 |
| 决策引擎 | LLM + 规则 fallback | 纯规则 / 纯 LLM | 平衡灵活与确定 |
| 执行安全 | dry_run 默认 + 环境变量门禁 | 完全开放 / 完全禁止 | 可控演示 |
| 接口协议 | MCP stdio | REST API / gRPC | Agent 原生适配 |

### 9.2 已知问题
- **Schema 缺口**：global 对象表缺 baseline_value / snapshot_date 列
- **数据依赖**：pipeline 需要手动放置 CSV 文件
- **实时性**：批处理架构，无法演示实时告警场景
- **规模**：PostgreSQL 单机，非分布式架构

### 9.3 下一步演进
- Schema 自动演进：基于对象定义自动生成 migration
- 实时管道：CDC 从业务数据库捕获变更
- 多模态对象：支持图片、文档等非结构化数据
- 分布式执行：分片处理大规模对象集

---

## 第 10 章 结论（1 页）

### 核心观点
Baxi 证明了：本体论驱动的数据平台能让 AI Agent 理解业务语义、自主决策、安全执行。关键不是技术复杂度，而是**把业务对象作为系统设计的核心**。

### 关键数字
- 9 种对象类型，31 个 MCP 工具
- 7 步管道，4 种标准动作
- 端到端闭环 14/19 步通过

### 启示
Palantir AIP 的本体论不是大企业专属。一个小团队、一个周末、一个可运行的 demo，就能验证：当数据系统围绕业务对象设计时，AI Agent 才能真正成为业务参与者，而不只是查询工具。

---

## 附录

### A. 技术栈
- Go 1.23 + chi/v5 + pgx/v5
- React 19 + Vite 6 + Tailwind CSS v4
- PostgreSQL 15 + Goose migrations
- MCP stdio + Pi Agent

### B. 项目统计
- Go 后端：29 个 internal 包
- 前端：13 个页面，5 个共享组件
- 配置：28 个 YAML governance 文件
- 测试：单元测试 + E2E 测试 + 安全测试
- 提交：96 个 commit，6 个 phase

### C. 快速启动
```bash
make up          # docker compose up postgres
go run ./cmd/baxi-cli e2e --auto-fix
```
