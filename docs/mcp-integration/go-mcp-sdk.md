# Go MCP Server 开发文档

## SDK 选择

| 维度 | Official SDK (`go-sdk`) | Mark3Labs (`mcp-go`) |
|------|------------------------|----------------------|
| **架构** | 单一 `mcp/` 包 | `mcp/` + `server/` + `client/` 多包 |
| **Go 版本** | 1.25 | 1.23 |
| **JSON-RPC** | 从 gopls fork（久经考验） | 自定义实现 |
| **工具 Schema** | 基于反射 + `jsonschema` 标签 | 流式构建器 API |
| **验证** | `AddTool[In,Out]` 自动验证 | `Require*` 辅助函数（可选） |
| **Stdio** | `StdioTransport{}` 传给 `server.Run()` | `server.ServeStdio(s)` |
| **Hooks** | 方法处理器（可选） | 完整 Hooks 系统 |
| **Stars** | ~4,600 | ~8,600 |
| **成熟度** | 官方，Google 维护 | 社区，更久经考验 |

**建议**：使用 `mark3labs/mcp-go`，因为：
- Go 1.22 兼容（项目当前版本）
- 更流行的社区 SDK
- 更丰富的构建器 API
- 完整的 Hooks 和会话管理

---

## 1. mark3labs/mcp-go 快速开始

### 安装

```bash
go get github.com/mark3labs/mcp-go@v0.41.1
```

### 最小服务器示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    s := server.NewMCPServer("Baxi MCP Server", "1.0.0", server.WithToolCapabilities(false))

    // 定义工具
    tool := mcp.NewTool("hello_world",
        mcp.WithDescription("Say hello to someone"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Name of the person to greet"),
        ),
    )
    s.AddTool(tool, helloHandler)

    // 启动 stdio 服务器
    if err := server.ServeStdio(s); err != nil {
        fmt.Printf("Server error: %v\n", err)
    }
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    name, err := request.RequireString("name")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}
```

---

## 2. 工具定义 API

### 流式构建器

```go
tool := mcp.NewTool("calculate",
    mcp.WithDescription("Perform basic arithmetic"),
    mcp.WithString("operation",
        mcp.Required(),
        mcp.Description("The operation"),
        mcp.Enum("add", "subtract", "multiply", "divide"),
    ),
    mcp.WithNumber("x", mcp.Required(), mcp.Description("First number")),
    mcp.WithNumber("y", mcp.Required(), mcp.Description("Second number")),
)
```

### 可用构建器函数

| 函数 | 用途 |
|------|------|
| `WithString(name, opts...)` | 字符串参数 |
| `WithNumber(name, opts...)` | 数字参数 |
| `WithBoolean(name, opts...)` | 布尔参数 |
| `WithArray(name, opts...)` | 数组参数 |
| `WithObject(name, opts...)` | 对象参数 |
| `Required()` | 标记为必需 |
| `Description(desc)` | 参数描述 |
| `Enum(values...)` | 枚举值 |
| `Default(value)` | 默认值 |
| `Min(value)` | 最小值 |
| `Max(value)` | 最大值 |
| `Pattern(regex)` | 正则模式 |

### 类型化工具处理器

```go
type MyArgs struct {
    Name  string `json:"name"`
    Count int    `json:"count"`
}

handler := mcp.NewTypedToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args MyArgs) (*mcp.CallToolResult, error) {
    return mcp.NewToolResultText(fmt.Sprintf("%+v", args)), nil
})
```

### 结构化输出

```go
type Out struct {
    Value int `json:"value"`
}

handler := mcp.NewStructuredToolHandler(func(ctx context.Context, req mcp.CallToolRequest, args In) (Out, error) {
    return Out{Value: 42}, nil
})
```

---

## 3. 结果构建器

```go
mcp.NewToolResultText("hello")                // 简单文本
mcp.NewToolResultError("something broke")      // 错误结果
mcp.NewToolResultJSON(data)                    // JSON 内容 + 结构化
mcp.NewToolResultStructuredOnly(data)           // 仅结构化 + JSON 回退
mcp.NewToolResultImage("caption", data, "png") // 文本 + 图片
mcp.NewToolResultAudio("desc", data, "wav")    // 文本 + 音频
mcp.FormatNumberResult(3.14)                   // 格式化浮点数
```

---

## 4. 参数提取辅助函数

```go
func handler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // 必需参数（带错误）
    name, err := request.RequireString("name")
    count, err := request.RequireInt("count")

    // 可选参数（带默认值）
    name := request.GetString("name", "default")
    count := request.GetInt("count", 42)

    // 可用方法
    // GetString, GetInt, GetFloat, GetBool
    // GetStringSlice, GetIntSlice, GetFloatSlice, GetBoolSlice
    // Require* 变体
}
```

---

## 5. Stdio 服务器

### 基本用法

```go
server.ServeStdio(s)
```

### 带选项

```go
server.ServeStdio(s,
    server.WithErrorLogger(log.New(os.Stderr, "", 0)),
    server.WithWorkerPoolSize(10),
    server.WithQueueSize(1000),
)
```

---

## 6. 请求 Hooks

```go
s := server.NewMCPServer("App", "1.0.0")

s.Hooks.AddBeforeCallTool(func(ctx context.Context, req *mcp.CallToolRequest) {
    log.Printf("Calling tool: %s", req.Params.Name)
})

s.Hooks.AddAfterCallTool(func(ctx context.Context, req *mcp.CallToolRequest, res *mcp.CallToolResult) {
    log.Printf("Tool %s completed", req.Params.Name)
})

s.Hooks.AddOnError(func(ctx context.Context, err error) {
    log.Printf("Error: %v", err)
})
```

---

## 7. 会话管理

### 每会话工具

```go
s.AddSessionTool(sessionID, tool, handler)
```

### 工具过滤

```go
s := server.NewMCPServer("App", "1.0.0", server.WithToolFilter(func(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
    session := server.ClientSessionFromContext(ctx)
    return filterToolsForSession(session, tools)
}))
```

---

## 8. 错误处理模式

### 工具错误（对 LLM 可见）

```go
func handler(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    if input.Invalid {
        return mcp.NewToolResultError("invalid input: " + err.Error()), nil
    }
    return mcp.NewToolResultText("ok"), nil
}
```

### 协议错误（中断连接）

```go
func handler(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    if err := db.CheckConnection(); err != nil {
        return nil, fmt.Errorf("database unavailable: %w", err) // 协议错误
    }
    return mcp.NewToolResultText("ok"), nil
}
```

---

## 9. 完整示例：Baxi MCP Server

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
)

func main() {
    s := server.NewMCPServer("Baxi MCP Server", "1.0.0",
        server.WithToolCapabilities(false),
        server.WithInstructions("E-commerce governance and decision platform"),
    )

    // 决策工具
    s.AddTool(mcp.NewTool("create_decision_case",
        mcp.WithDescription("Create a decision case from an alert"),
        mcp.WithString("alert_id", mcp.Required(), mcp.Description("Alert ID")),
    ), createCaseHandler)

    s.AddTool(mcp.NewTool("decide",
        mcp.WithDescription("Generate decision for a case"),
        mcp.WithString("case_id", mcp.Required(), mcp.Description("Decision case ID")),
    ), decideHandler)

    // 治理工具
    s.AddTool(mcp.NewTool("check_access",
        mcp.WithDescription("Check access policy for a role"),
        mcp.WithString("role", mcp.Required(), mcp.Description("User role")),
        mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type")),
        mcp.WithString("action", mcp.Required(), mcp.Description("Action to check")),
    ), checkAccessHandler)

    // 管道工具
    s.AddTool(mcp.NewTool("run_pipeline",
        mcp.WithDescription("Run data pipeline"),
        mcp.WithString("pipeline_type", mcp.Description("Pipeline type"), mcp.Enum("daily", "full")),
    ), runPipelineHandler)

    if err := server.ServeStdio(s); err != nil {
        log.Printf("Server error: %v", err)
        os.Exit(1)
    }
}

func createCaseHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    alertID, err := request.RequireString("alert_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    // 调用现有 DecisionService
    result := map[string]interface{}{
        "case_id": "case-123",
        "status":  "created",
    }
    return mcp.NewToolResultJSON(result), nil
}

func decideHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    caseID, err := request.RequireString("case_id")
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    // 调用现有 DecisionService.Decide()
    result := map[string]interface{}{
        "decision_type": "investigate",
        "severity":      "high",
        "confidence":    0.85,
    }
    return mcp.NewToolResultJSON(result), nil
}

func checkAccessHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    role, _ := request.RequireString("role")
    objectType, _ := request.RequireString("object_type")
    action, _ := request.RequireString("action")
    // 调用现有 AccessPolicyService
    result := map[string]interface{}{
        "allowed": true,
        "role":    role,
    }
    return mcp.NewToolResultJSON(result), nil
}

func runPipelineHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    pipelineType := request.GetString("pipeline_type", "daily")
    // 调用现有 PipelineRunner
    result := map[string]interface{}{
        "run_id":   "run-456",
        "status":   "started",
        "type":     pipelineType,
    }
    return mcp.NewToolResultJSON(result), nil
}
```

---

## 关键参考链接

| 资源 | URL |
|------|-----|
| mark3labs/mcp-go 仓库 | https://github.com/mark3labs/mcp-go |
| pkg.go.dev 文档 | https://pkg.go.dev/github.com/mark3labs/mcp-go@v0.41.1 |
| server 包文档 | https://pkg.go.dev/github.com/mark3labs/mcp-go@v0.41.1/server |
| README 示例 | https://github.com/mark3labs/mcp-go/blob/main/README.md |
| 类型化工具 | https://github.com/mark3labs/mcp-go/blob/main/mcp/typed_tools.go |
| stdio 服务器 | https://github.com/mark3labs/mcp-go/blob/main/server/stdio.go |
| MCP 规范 (2025-03-26) | https://modelcontextprotocol.io/specification/2025-03-26 |
