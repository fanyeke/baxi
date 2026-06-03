# Phase 05: Security Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-03
**Phase:** 05-security-hardening
**Areas discussed:** SEC-01 (skipped), SEC-02 (CORS), SEC-03 (skipped)

---

## Phase 05 Scope Decision

**User's choice:** "本地执行的测试程序，安全可以不用多做什么"
**Notes:** 用户明确项目是本地测试/演示程序，安全投入最小化。仅修复 CORS scheme 验证。

---

## SEC-02: CORS Scheme 验证

| Option | Description | Selected |
|--------|-------------|----------|
| URL 解析 + 精确比较（推荐） | url.Parse 提取 scheme+host+port 精确比较 | ✓ |
| 字符串前缀匹配 | 简单检查 Origin 前缀，不够精确 | |

**User's choice:** URL 解析 + 精确比较

**Follow-up — 来源列表格式:**

| Option | Description | Selected |
|--------|-------------|----------|
| 保持现有格式（推荐） | 逗号分隔格式不变，只改验证逻辑 | ✓ |
| 改为 JSON 数组 | 更清晰但需要迁移配置 | |

**User's choice:** 保持现有格式

---

## SEC-01 and SEC-03: Skipped

| Option | Description | Selected |
|--------|-------------|----------|
| SEC-01: 最小化 | 文档说明 API_BEARER_TOKEN 定期轮换，不改代码 | |
| SEC-01: 中等 | 多 token 支持但不引入 JWT | |
| SEC-02: 修复 scheme 验证 | cors.go URL 解析 + 精确比较 | ✓ |
| SEC-03: .env 文件化 | docker-compose.yml 引用环境变量 | |
| 全部跳过 | 安全不做任何修改 | |

**User's choice:** 仅 SEC-02，其余跳过
**Notes:** 本地演示程序，安全攻击面极小。JWT/token 轮换和 Docker Compose 凭据管理均为不必要开销。

## the agent's Discretion

- URL 解析失败时的行为（fail closed vs fail open）
- 端口默认值（80/443）
- 具体测试方法

## Deferred Ideas

- SEC-01（JWT/token 轮换）——若部署到非本地环境再实现
- SEC-03（Docker Compose 凭据）——若公开部署再处理
