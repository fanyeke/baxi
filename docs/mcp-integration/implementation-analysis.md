# Baxi MCP 集成 — 实现重难点分析

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                    Pi Agent Framework                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  自定义扩展                                          │   │
│  │  - pi-mcp-adapter (MCP 客户端)                      │   │
│  │  - decision-skill (决策逻辑)                         │   │
│  │  - threshold-trigger (阈值触发)                      │   │
│  │  - scheduler (定时任务)                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│                      MCP 协议 (stdio)                        │
│                           │                                  │
└───────────────────────────┼──────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Baxi MCP Server (Go)                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  决策工具    │  │  治理工具    │  │  管道工具    │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│                           │                                  │
│                    Go 后端 (现有代码)                         │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔴 高难度挑战

### 1. Go MCP Server 实现

**挑战**：
- 需要将现有的 `DecisionService`、`GovernanceService` 等业务逻辑暴露为 MCP 工具
- 工具定义需要精确的 JSON Schema（输入/输出）
- 错误处理需要转换为 MCP 错误格式
- 需要实现 stdio 传输层（JSON-RPC 2.0 over stdin/stdout）

**关键决策**：

| 决策点 | 选项 | 推荐 | 理由 |
|--------|------|------|------|
| SDK 选择 | Official vs mark3labs | mark3labs | Go 1.22 兼容，更流行 |
| 工具粒度 | 1:1 vs 聚合 | 聚合 | 减少工具数量，降低 LLM token 消耗 |
| 长时间操作 | 同步 vs 异步 | 异步 | 管道执行可能需要几分钟 |

**解决方案**：

```go
// 1. 创建 MCP Server 包
internal/mcp/
├── server.go          # MCP 服务器初始化
├── tools_decision.go  # 决策相关工具
├── tools_governance.go # 治理相关工具
├── tools_pipeline.go  # 管道相关工具
└── tools_status.go    # 状态查询工具

// 2. 工具定义示例
func (s *BaxiMCPServer) registerDecisionTools() {
    s.server.AddTool(mcp.NewTool("create_decision_case",
        mcp.WithDescription("Create a decision case from an alert"),
        mcp.WithString("alert_id", mcp.Required(), mcp.Description("Alert ID")),
    ), s.createCaseHandler)
}

// 3. 处理器调用现有服务
func (s *BaxiMCPServer) createCaseHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    alertID, err := req.RequireString("alert_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // 调用现有 DecisionService
    case, err := s.decisionSvc.CreateCaseFromAlert(ctx, alertID, "mcp_agent")
    if err != nil {
        return mcp.NewToolResultError("Failed to create case: " + err.Error()), nil
    }
    
    return mcp.NewToolResultJSON(case), nil
}
```

**难点清单**：
- [ ] 实现 `cmd/baxi-mcp/main.go` 入口点
- [ ] 创建 `internal/mcp/` 包，封装 MCP 服务器
- [ ] 定义 10+ 个 MCP 工具，覆盖核心业务流程
- [ ] 实现 stdio 传输层（使用 mark3labs/mcp-go）
- [ ] 处理长时间运行的操作（管道执行、LLM 调用）
- [ ] 实现错误处理和日志记录

---

### 2. Pi 扩展开发（TypeScript 侧）

**挑战**：
- 需要开发 MCP 客户端扩展，连接 Go MCP Server
- 需要实现阈值触发逻辑（何时自动调用决策？）
- 需要实现定时任务调度（轮询告警？）
- 需要管理会话状态（决策上下文、历史记录）

**关键决策**：

| 决策点 | 选项 | 推荐 | 理由 |
|--------|------|------|------|
| MCP 客户端 | pi-mcp-adapter vs 自定义 | pi-mcp-adapter | 成熟稳定，750 stars |
| 阈值规则 | Pi 侧 vs Go 侧 | Go 侧 | 复杂规则引擎已在 Go 实现 |
| 定时任务 | Pi 扩展 vs cron | Pi 扩展 | 与 agent 生命周期集成 |

**解决方案**：

```typescript
// ~/.pi/agent/extensions/baxi-decision/index.ts
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
    // 1. 注册决策 skill
    pi.registerTool({
        name: "baxi_create_case",
        label: "Create Decision Case",
        description: "Create a decision case from a Baxi alert",
        parameters: Type.Object({
            alert_id: Type.String({ description: "Alert ID" }),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            // 调用 MCP 工具
            const result = await ctx.mcp.call("create_decision_case", {
                alert_id: params.alert_id,
            });
            return {
                content: [{ type: "text", text: JSON.stringify(result) }],
                details: result,
            };
        },
    });

    // 2. 阈值触发逻辑
    pi.on("session_start", async (_event, ctx) => {
        // 启动定时任务
        setInterval(async () => {
            const alerts = await ctx.mcp.call("list_alerts", {
                severity: "high",
                status: "new",
            });
            
            for (const alert of alerts.items) {
                // 检查阈值
                if (alert.delta_pct > 20) {
                    // 自动创建决策案例
                    await ctx.mcp.call("create_decision_case", {
                        alert_id: alert.event_id,
                    });
                    ctx.ui.notify(`Auto-created case for alert ${alert.event_id}`, "info");
                }
            }
        }, 60000); // 每分钟检查一次
    });
}
```

**难点清单**：
- [ ] 安装并配置 pi-mcp-adapter
- [ ] 创建 `baxi-decision` 扩展
- [ ] 实现阈值触发逻辑（delta_pct > 20%）
- [ ] 实现定时任务调度（每分钟检查告警）
- [ ] 管理会话状态（决策上下文）
- [ ] 实现用户通知（ctx.ui.notify）

---

### 3. 决策流程集成

**挑战**：
- 现有的 `DecisionService.Decide()` 流程需要适配 MCP 工具调用
- 需要处理异步操作（LLM 调用可能很慢）
- 需要保持 `RequiresHumanReview = true` 的安全约束
- 需要将决策结果格式化为 LLM 可理解的结构

**关键决策**：

| 决策点 | 选项 | 推荐 | 理由 |
|--------|------|------|------|
| 结果呈现 | 直接返回 vs 通知 vs 审批流 | 通知 + 审批 | 保持人在回路 |
| LLM 失败 | 重试 vs 回退 | 回退到规则引擎 | 现有设计 |
| 审计追踪 | 内存 vs 数据库 | 数据库 | 已有 audit 表 |

**解决方案**：

```go
// MCP 工具处理器
func (s *BaxiMCPServer) decideHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    caseID, err := req.RequireString("case_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    
    // 调用现有决策服务
    decisionCtx, decision, proposals, err := s.decisionSvc.Decide(ctx, caseID)
    if err != nil {
        return mcp.NewToolResultError("Decision failed: " + err.Error()), nil
    }
    
    // 格式化结果
    result := map[string]interface{}{
        "case_id":      caseID,
        "decision":     decision,
        "proposals":    proposals,
        "context":      decisionCtx,
        "requires_review": decision.RequiresHumanReview,
    }
    
    return mcp.NewToolResultJSON(result), nil
}
```

**难点清单**：
- [ ] 适配 `DecisionService.Decide()` 为 MCP 工具
- [ ] 处理 LLM 调用超时（设置 30s 超时）
- [ ] 实现错误回退（LLM 失败 → 规则引擎）
- [ ] 格式化决策结果为 JSON
- [ ] 保持 `RequiresHumanReview = true` 约束
- [ ] 记录 MCP 调用到审计日志

---

## 🟡 中等难度

### 4. 治理规则集成

**挑战**：
- 数据分类（L1/L2/L3）需要在 MCP 工具中正确传递
- 字段脱敏需要根据调用者角色动态调整
- 访问策略需要与 MCP 工具的权限模型对齐

**解决方案**：

```go
func (s *BaxiMCPServer) checkAccessHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    role, _ := req.RequireString("role")
    objectType, _ := req.RequireString("object_type")
    action, _ := req.RequireString("action")
    
    // 调用现有访问策略服务
    decision := s.accessPolicySvc.CheckAccess(ctx, role, objectType, action)
    
    return mcp.NewToolResultJSON(map[string]interface{}{
        "allowed": decision.AccessAllowed,
        "reason":  decision.Reason,
    }), nil
}
```

**难点清单**：
- [ ] 暴露 `check_access` 工具
- [ ] 暴露 `classify_field` 工具
- [ ] 暴露 `redact_context` 工具
- [ ] 处理角色映射（MCP 调用者 → Baxi 角色）

---

### 5. 测试策略

**难点清单**：
- [ ] Go 侧：使用 `testcontainers` 测试 MCP Server
- [ ] Pi 侧：使用 Vitest 测试扩展逻辑
- [ ] E2E：使用真实 LLM 调用测试完整流程
- [ ] 性能测试：MCP 调用延迟 < 100ms

---

## 🟢 低难度

### 6. 配置管理

**难点清单**：
- [ ] MCP Server 配置（环境变量）
- [ ] Pi 扩展配置（mcp.json）
- [ ] 阈值规则配置（YAML）

### 7. 日志和监控

**难点清单**：
- [ ] 统一 request_id 传递
- [ ] Go 侧记录 MCP 调用日志
- [ ] Pi 侧记录扩展日志

---

## 实现阶段

| 阶段 | 任务 | 难度 | 依赖 | 预计工时 |
|------|------|------|------|----------|
| **Phase 1** | Go MCP Server（stdio） | 🔴 | 无 | 3-5 天 |
| **Phase 2** | Pi 扩展（MCP 客户端） | 🔴 | Phase 1 | 2-3 天 |
| **Phase 3** | 决策工具集成 | 🔴 | Phase 1, 2 | 2-3 天 |
| **Phase 4** | 阈值触发逻辑 | 🟡 | Phase 3 | 1-2 天 |
| **Phase 5** | 定时任务调度 | 🟡 | Phase 4 | 1 天 |
| **Phase 6** | 治理规则集成 | 🟡 | Phase 3 | 1-2 天 |
| **Phase 7** | 测试和文档 | 🟢 | All | 2-3 天 |

**总计**：12-19 天

---

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| Go MCP SDK | mark3labs/mcp-go | v0.41.1 |
| Pi Agent Framework | @earendil-works/pi | v0.76.0 |
| Pi MCP Adapter | pi-mcp-adapter | v2.8.0 |
| LLM Provider | OpenAI / Anthropic | - |
| 数据库 | PostgreSQL 16 | - |
| 测试 | testcontainers + Vitest | - |

---

## 风险和缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| MCP 协议变更 | 高 | 使用成熟 SDK，关注规范更新 |
| Pi 扩展兼容性 | 中 | 锁定版本，测试升级 |
| LLM 调用延迟 | 中 | 设置超时，回退到规则引擎 |
| 安全风险 | 高 | 保持 RequiresHumanReview，审计日志 |
