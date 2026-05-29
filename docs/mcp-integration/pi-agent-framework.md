# Pi Agent Framework — 开发文档

> **项目**: [earendil-works/pi](https://github.com/earendil-works/pi) · 57K+ stars · **v0.76.0** (May 27, 2026)  
> **作者**: Mario Zechner (badlogic) · **许可证**: MIT  
> **官网**: [pi.dev](https://pi.dev) · **npm**: `@earendil-works/pi-*`

## 架构概览

```
┌─────────────────────────────────────┐
│  你的应用                            │
│  (OpenClaw, CLI, Slack bot, etc.)   │
├────────────────┬────────────────────┤
│ pi-coding-agent│ pi-tui             │
│ Sessions, tools│ Terminal UI,       │
│ extensions     │ markdown, editor   │
├────────────────┴────────────────────┤
│  pi-agent-core                      │
│  Agent loop, tool execution, events │
├─────────────────────────────────────┤
│  pi-ai                              │
│  Streaming, models, multi-provider  │
└─────────────────────────────────────┘
```

---

## 1. pi-agent-core SDK

**npm**: `@earendil-works/pi-agent-core` · 1.3M weekly downloads

### 1.1 安装

```bash
npm install @earendil-works/pi-agent-core
```

### 1.2 创建 Agent

```typescript
import { Agent } from "@earendil-works/pi-agent-core";
import { getModel } from "@earendil-works/pi-ai";

const agent = new Agent({
  initialState: {
    systemPrompt: "You are a helpful assistant.",
    model: getModel("anthropic", "claude-sonnet-4-20250514"),
    tools: [],
    thinkingLevel: "off",
  },
});

agent.subscribe((event) => {
  if (event.type === "message_update" && event.assistantMessageEvent.type === "text_delta") {
    process.stdout.write(event.assistantMessageEvent.delta);
  }
});

await agent.prompt("Hello!");
```

### 1.3 定义工具

使用 [TypeBox](https://github.com/sinclairzx81/typebox) 进行类型安全的参数定义：

```typescript
import { Type } from "typebox";
import type { AgentTool } from "@earendil-works/pi-agent-core";

const readFileTool: AgentTool = {
  name: "read_file",
  label: "Read File",
  description: "Read a file's contents",
  parameters: Type.Object({
    path: Type.String({ description: "File path" }),
  }),
  execute: async (toolCallId, params, signal, onUpdate) => {
    const content = await fs.readFile(params.path, "utf-8");
    return {
      content: [{ type: "text", text: content }],
      details: { path: params.path, size: content.length },
    };
  },
};

agent.state.tools = [readFileTool];
```

**错误处理**: 抛出错误（不要返回错误消息作为内容）：

```typescript
execute: async (toolCallId, params) => {
  if (!fs.existsSync(params.path)) {
    throw new Error(`File not found: ${params.path}`);
  }
  return { content: [{ type: "text", text: "..." }] };
}
```

### 1.4 事件流

Agent 发出以下事件（订阅顺序保证）：

| 事件 | 描述 |
|------|------|
| `agent_start` | Agent 开始处理 |
| `agent_end` | 运行的最终事件 |
| `turn_start` | 一次 LLM 调用 + 工具执行开始 |
| `turn_end` | Turn 完成，包含消息 + 工具结果 |
| `message_start` | 任何消息开始（user, assistant, toolResult） |
| `message_update` | **仅 Assistant** — 包含流式 delta |
| `message_end` | 消息完成 |
| `tool_execution_start` | 工具开始 |
| `tool_execution_update` | 工具流式进度 |
| `tool_execution_end` | 工具完成 |

**带工具调用的 Prompt 流程**：

```
prompt("Read config.json")
├─ agent_start
├─ turn_start
├─ message_start/end  { userMessage }
├─ message_start      { assistantMessage with toolCall }
├─ message_update...  (streaming)
├─ message_end
├─ tool_execution_start  { toolCallId, toolName, args }
├─ tool_execution_update { partialResult }
├─ tool_execution_end    { result }
├─ message_start/end  { toolResultMessage }
├─ turn_end
├─ turn_start         (下一个 turn — LLM 响应工具结果)
├─ message_update...
├─ message_end
└─ agent_end
```

### 1.5 Steering & Follow-ups

在工具运行时中断 Agent，或在完成后排队工作：

```typescript
// 中断：在当前工具完成后传递，剩余工具被跳过
agent.steer({
  role: "user",
  content: "Stop! Do this instead.",
  timestamp: Date.now(),
});

// Follow-up：在 Agent 自然停止后排队
agent.followUp({
  role: "user",
  content: "Also summarize the result.",
  timestamp: Date.now(),
});
```

### 1.6 工具执行模式

- **`parallel`**（默认）：顺序预检，并发执行，按完成顺序发出 `tool_execution_end`
- **`sequential`**：一次一个
- 每个工具可通过 `executionMode: "sequential"` 覆盖 — 强制整个批次顺序执行

### 1.7 Hooks

```typescript
const agent = new Agent({
  beforeToolCall: async ({ toolCall, args, context }) => {
    if (toolCall.name === "bash") {
      return { block: true, reason: "bash is disabled" };
    }
  },
  afterToolCall: async ({ toolCall, result, isError, context }) => {
    return { details: { ...result.details, audited: true } };
  },
});
```

### 1.8 状态管理

```typescript
agent.state.systemPrompt = "New prompt";
agent.state.model = getModel("openai", "gpt-4o");
agent.state.thinkingLevel = "medium";
agent.state.tools = [myTool];
agent.state.messages = newMessages; // 顶层数组被复制
agent.reset();
```

---

## 2. 扩展开发

**官方文档**: [pi.dev/docs/latest/extensions](https://pi.dev/docs/latest/extensions)

扩展是 TypeScript 模块，可以订阅生命周期事件、注册自定义工具、添加命令等。

### 2.1 快速开始

创建 `~/.pi/agent/extensions/my-extension.ts`：

```typescript
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
  // 响应事件
  pi.on("session_start", async (_event, ctx) => {
    ctx.ui.notify("Extension loaded!", "info");
  });

  // 拦截工具调用
  pi.on("tool_call", async (event, ctx) => {
    if (event.toolName === "bash" && event.input.command?.includes("rm -rf")) {
      const ok = await ctx.ui.confirm("Dangerous!", "Allow rm -rf?");
      if (!ok) return { block: true, reason: "Blocked by user" };
    }
  });

  // 注册自定义工具
  pi.registerTool({
    name: "greet",
    label: "Greet",
    description: "Greet someone by name",
    parameters: Type.Object({
      name: Type.String({ description: "Name to greet" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      return {
        content: [{ type: "text", text: `Hello, ${params.name}!` }],
        details: {},
      };
    },
  });

  // 注册命令
  pi.registerCommand("hello", {
    description: "Say hello",
    handler: async (args, ctx) => {
      ctx.ui.notify(`Hello ${args || "world"}!`, "info");
    },
  });
}
```

测试：`pi -e ./my-extension.ts`

### 2.2 扩展位置

| 位置 | 作用域 |
|------|--------|
| `~/.pi/agent/extensions/*.ts` | 全局 |
| `~/.pi/agent/extensions/*/index.ts` | 全局（子目录） |
| `.pi/extensions/*.ts` | 项目本地 |
| `.pi/extensions/*/index.ts` | 项目本地 |

### 2.3 可用导入

| 包 | 用途 |
|---|------|
| `@earendil-works/pi-coding-agent` | 扩展类型（`ExtensionAPI`, `ExtensionContext`, events） |
| `typebox` | 工具参数的 Schema 定义 |
| `@earendil-works/pi-ai` | AI 工具 |
| `@earendil-works/pi-tui` | 自定义渲染的 TUI 组件 |

### 2.4 完整事件生命周期

```
pi starts
  ├─► session_start { reason: "startup" }
  └─► resources_discover
      │
user sends prompt
  ├─► (extension commands checked first)
  ├─► input (can intercept)
  ├─► before_agent_start (inject message, modify system prompt)
  ├─► agent_start
  ├─► turn_start
  ├─► context (modify messages before LLM call)
  ├─► before_provider_request
  ├─► after_provider_response
  │   LLM may call tools:
  │     ├─► tool_execution_start
  │     ├─► tool_call (can block)
  │     ├─► tool_result (can modify)
  │     └─► tool_execution_end
  └─► turn_end / agent_end
```

### 2.5 关键 ExtensionAPI 方法

**事件** — `pi.on(event, handler)` — 拦截生命周期：
- `session_start`, `session_before_switch`, `session_before_fork`, `session_before_compact`, `session_shutdown`
- `before_agent_start`, `agent_start`, `agent_end`, `turn_start`, `turn_end`
- `context`, `message_start/update/end`
- `tool_call`（可**阻止**）, `tool_result`（可**修改**）
- `model_select`, `thinking_level_select`

**注册能力**：
- `pi.registerTool(definition)` — 为 LLM 注册工具
- `pi.registerCommand(name, options)` — 注册 `/mycommand`
- `pi.registerShortcut("ctrl+x", options)` — 键盘快捷键
- `pi.registerFlag(name, options)` — CLI 标志
- `pi.registerProvider(name, config)` — 自定义 LLM 提供商

**发送/交互**：
- `pi.sendMessage(message)` — 向对话注入消息
- `pi.sendUserMessage(content)` — 注入用户消息
- `pi.appendEntry(customType, data)` — 会话持久化（重启后保留）

### 2.6 状态持久化

```typescript
// 存储重启后保留的状态
pi.appendEntry("my-ext-state", { todos: ["item1", "item2"] });

// 在下次启动时检索
pi.on("session_start", async (event, ctx) => {
  const entries = ctx.sessionManager.getEntries();
  const myState = entries.find(e => e.customType === "my-ext-state");
});
```

### 2.7 自定义 UI

扩展可以通过 `ctx.ui` 完全访问 TUI：
- `ctx.ui.notify(msg, level)` — toast 通知
- `ctx.ui.confirm(title, msg)` — 是/否对话框
- `ctx.ui.select(title, options)` — 选择列表
- `ctx.ui.input(title, placeholder)` — 文本输入
- `ctx.ui.setStatus(id, text)` — 页脚状态栏
- `ctx.ui.setWidget(id, lines)` — 编辑器上方的 widget
- `ctx.ui.custom(component)` — 完整 TUI 组件

---

## 3. pi-mcp-adapter

**GitHub**: [nicobailon/pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter) · 750 stars · v2.8.0  
**npm**: `pi-mcp-adapter` · 20.7K weekly downloads

### 3.1 安装

```bash
pi install npm:pi-mcp-adapter
```

这会安装到 `~/.pi/agent/extensions/pi-mcp-adapter/` 并添加到设置中。然后重启 Pi。

### 3.2 配置

创建 `~/.pi/agent/mcp.json`：

```json
{
  "settings": {
    "toolPrefix": "server",
    "idleTimeout": 10,
    "directTools": false
  },
  "mcpServers": {
    "chrome-devtools": {
      "command": "npx",
      "args": ["-y", "chrome-devtools-mcp@latest"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    }
  }
}
```

### 3.3 使用

适配器注册一个 `mcp` 代理工具（~200 tokens）而不是直接加载所有 MCP 工具：

```
# 搜索工具
mcp({ search: "screenshot" })

# 列出服务器的所有工具
mcp({ server: "chrome-devtools" })

# 调用工具
mcp({ tool: "chrome_devtools_take_screenshot", args: '{"format": "png"}' })

# 描述特定工具
mcp({ describe: "chrome_devtools_take_screenshot" })

# 强制服务器连接
mcp({ connect: "chrome-devtools" })
```

### 3.4 生命周期模式

| 模式 | 行为 | 最佳用途 |
|------|------|----------|
| `lazy`（默认） | 首次工具调用时连接，空闲超时后断开 | 大多数服务器 |
| `eager` | 启动时连接，不自动重连 | 会话中频繁使用 |
| `keep-alive` | 启动时连接，通过健康检查自动重连 | 关键的始终在线服务器 |

### 3.5 直接工具注册

将特定工具提升为一等 Pi 工具：

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "directTools": ["search_repositories", "get_file_contents"]
    }
  }
}
```

每个直接工具在系统提示中花费约 150-300 tokens。代理对于有 75+ 工具的服务器更好。

### 3.6 交互式管理

在 Pi 中使用 `/mcp` 打开管理面板，包含服务器状态、工具切换、重连和 OAuth 设置。

---

## 4. Skill 系统

**官方文档**: [pi.dev/docs/latest/skills](https://pi.dev/docs/latest/skills)

技能是自包含的能力包，Agent 按需加载。Pi 实现了 [Agent Skills 标准](https://agentskills.io/specification)。

### 4.1 技能结构

```
my-skill/
├── SKILL.md              # 必需：frontmatter + 说明
├── scripts/
│   └── process.sh
└── references/
    └── api-reference.md
```

### 4.2 SKILL.md 格式

```markdown
---
name: my-skill
description: What this skill does and when to use it. Be specific.
---

# My Skill

## Setup
Run once before first use:
```bash
cd /path/to/skill && npm install
```

## Usage
```bash
./scripts/process.sh <input>
```
```

### 4.3 Frontmatter

| 字段 | 必需 | 描述 |
|------|------|------|
| `name` | 是 | 最多 64 字符。小写 a-z, 0-9, 连字符 |
| `description` | 是 | 最多 1024 字符 |
| `allowed-tools` | 否 | 空格分隔的预批准工具 |
| `disable-model-invocation` | 否 | 如果 true，则对系统提示隐藏（仅使用 `/skill:name`） |

### 4.4 技能位置

- **全局**: `~/.pi/agent/skills/`, `~/.agents/skills/`
- **项目**: `.pi/skills/`, `.agents/skills/`（向上遍历祖先到 git root）
- **包**: `package.json` 中的 `skills/` 或 `pi.skills` 条目
- **设置**: `settings.json` 中的 `"skills": ["path/to/skill"]`
- **CLI**: `--skill <path>`（可重复）

### 4.5 技能如何工作

1. 启动时，Pi 扫描技能位置 → 提取名称/描述
2. 系统提示以 XML 格式包含可用技能（渐进式披露）
3. 当任务匹配时，Agent 使用 `read` 加载完整的 SKILL.md
4. Agent 遵循说明，使用相对路径引用脚本/资产

### 4.6 技能命令

技能注册为 `/skill:name` 命令：

```bash
/skill:brave-search           # 加载并执行技能
/skill:pdf-tools extract      # 加载技能并传递参数
```

通过 `settings.json` 启用：`{ "enableSkillCommands": true }`

### 4.7 技能仓库

- [Anthropic Skills](https://github.com/anthropics/skills) — docx, pdf, pptx, xlsx, web dev
- [Pi Skills](https://github.com/badlogic/pi-skills) — web search, browser automation, Google APIs, transcription

---

## 5. 嵌入 Pi（SDK 模式）

**官方文档**: [pi.dev/docs/latest/sdk](https://pi.dev/docs/latest/sdk)  
**真实示例**: [OpenClaw](https://github.com/OpenClaw/OpenClaw)

### 5.1 安装

```bash
npm install @earendil-works/pi-coding-agent
```

### 5.2 快速开始

```typescript
import { AuthStorage, createAgentSession, ModelRegistry, SessionManager } from "@earendil-works/pi-coding-agent";

const authStorage = AuthStorage.create();
const modelRegistry = ModelRegistry.create(authStorage);

const { session } = await createAgentSession({
  sessionManager: SessionManager.inMemory(),
  authStorage,
  modelRegistry,
});

session.subscribe((event) => {
  if (event.type === "message_update" && event.assistantMessageEvent.type === "text_delta") {
    process.stdout.write(event.assistantMessageEvent.delta);
  }
});

await session.prompt("What files are in the current directory?");
```

### 5.3 核心 API

**`createAgentSession(options)`** — 主工厂：

```typescript
const { session } = await createAgentSession({
  model: opus,                       // 来自 getModel() 的 Model
  tools: ["read", "bash", "edit"],  // 内置工具选择
  customTools: [myTool],            // 自定义 AgentTool[]
  thinkingLevel: "off",
  sessionManager: SessionManager.inMemory(),
  authStorage,
  modelRegistry,
  resourceLoader: new DefaultResourceLoader({
    systemPromptOverride: () => "You are a helpful assistant.",
  }),
});
```

**`AgentSession`** — 会话对象：

```typescript
session.prompt("text");                    // 发送提示，等待完成
session.steer("New instruction");          // 在流式传输时中断
session.followUp("Also do X");            // 在完成后排队
session.subscribe(listener);              // 事件流
session.setModel(model);                  // 切换模型
session.setThinkingLevel(level);          // 更改思考级别
session.compact("Preserve file paths");   // 触发压缩
session.dispose();                        // 清理
```

### 5.4 会话管理

```typescript
// 内存（无持久化）
SessionManager.inMemory();

// 新的持久化会话
SessionManager.create(process.cwd());

// 继续最近的
SessionManager.continueRecent(process.cwd());

// 打开特定文件
SessionManager.open("/path/to/session.jsonl");

// 列出会话
const sessions = await SessionManager.list(process.cwd());
```

会话使用树结构，`id`/`parentId` 链接，支持就地分支。

### 5.5 程序化模式

Pi 在**四种模式**下运行：

| 模式 | 描述 |
|------|------|
| **Interactive** | 完整 TUI 体验 |
| **Print/JSON** | `pi -p "query"` 用于脚本；`--mode json` 用于事件流 |
| **RPC** | JSON 协议通过 stdin/stdout，用于非 Node 集成 |
| **SDK** | 通过 `createAgentSession()` 嵌入 Pi 到你的应用 |

---

## 关键参考链接

| 资源 | URL |
|------|-----|
| GitHub 仓库 | [github.com/earendil-works/pi](https://github.com/earendil-works/pi) |
| 官方文档 | [pi.dev/docs/latest](https://pi.dev/docs/latest) |
| 扩展文档 | [pi.dev/docs/latest/extensions](https://pi.dev/docs/latest/extensions) |
| SDK 文档 | [pi.dev/docs/latest/sdk](https://pi.dev/docs/latest/sdk) |
| 技能文档 | [pi.dev/docs/latest/skills](https://pi.dev/docs/latest/skills) |
| MCP 适配器 | [nicobailon/pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter) |
| MCP 适配器文档 | [nicobailon-pi-mcp-adapter.mintlify.app](https://nicobailon-pi-mcp-adapter.mintlify.app/introduction) |
| Nader Dabit 指南 | [How to Build a Custom Agent Framework with PI](https://nader.substack.com/p/how-to-build-a-custom-agent-framework) |
| YouTube 速成课程 | [Pi Agent – Crash Course](https://www.youtube.com/watch?v=N30XGyPrr6I) |
