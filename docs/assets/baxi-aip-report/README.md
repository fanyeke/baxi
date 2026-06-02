# Baxi AIP 报告视觉资产

生成方式：内置 GPT Image 2 路径。图片统一使用浅色背景、深蓝与青绿色主体、少量琥珀色强调。

## 正文图片

| 文件 | 用途 | Prompt 主题 |
|---|---|---|
| `aip-platform-overview.png` | 报告封面 | 数据、Ontology、AI 与受治理动作的概念总览 |
| `current-state-scorecard.png` | 执行摘要 | 四项完成度与“先收敛，再扩展”结论 |
| `system-architecture-infographic.png` | 系统架构 | 六层主路径、统一真相来源、执行边界与审计 Trace |
| `dataset-profile-infographic.png` | 数据集 | 11 个 CSV、1,565,259 行与 `raw -> dwd -> mart -> ops` |
| `data-to-semantic-layer.png` | 语义层概念 | 原始记录转为 Agent 可消费上下文 |
| `ontology-semantic-layer-infographic.png` | Ontology | 8 类业务对象与 `LLM Safe Context` |
| `governed-decision-loop.png` | 决策引擎 | 分类、脱敏、审计、checkpoint 与人工审核 |
| `mcp-pi-integration.png` | Pi 接入 | Agent 工具网关与补充 API 通道 |
| `priority-map.png` | 风险清单 | P0、P1、P2 工程优先级 |
| `convergence-roadmap.png` | 路线图 | Phase A 到 Phase E 的闭环收敛路线 |

## 备选图片

| 文件 | 用途 | Prompt 主题 |
|---|---|---|
| `alternatives/governance-rail-layered-architecture.png` | 替换架构图 | Governance Rail 贯穿四层架构 |
| `alternatives/llm-safe-context-envelope.png` | 补充安全上下文 | 模型只接收被允许的事实 |
| `alternatives/pi-mcp-primary-path.png` | 替换 Pi 接入图 | MCP 主路径与 HTTP API 补充通道 |
| `alternatives/governed-action-review-flow-v2.png` | 补充动作闭环 | Proposal、Sandbox、Review、Outbox、Audit |
| `alternatives/eval-lifecycle.png` | 补充评估体系 | Context Hash、Replay、Compare、Eval 与 Outcome |

## 代码版架构图

`system-architecture.mmd` 是正文架构图的 Mermaid 源码版本。GPT Image 2 信息图负责快速传达结论，Mermaid 版本负责精确表达和后续维护。
