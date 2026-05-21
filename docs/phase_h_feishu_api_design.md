# Phase H: 飞书 API 接入方案设计

## 概述

本文档定义 Phase H 的飞书开放平台 API 接入方案，包括认证方式、权限范围、端点列表、速率限制和安全策略。

## 认证方案

### 认证模式：tenant_access_token（自建应用）

使用企业自建应用的 `tenant_access_token`，适用于服务端自动调用场景。

#### 认证流程

```
1. 使用 app_id + app_secret 请求 tenant_access_token
2. Token 有效期 7200 秒（2 小时）
3. 本地缓存，过期前 60 秒自动刷新
```

#### 获取 Token

```
POST https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal
Content-Type: application/json

{
  "app_id": "<APP_ID>",
  "app_secret": "<APP_SECRET>"
}
```

**Response**:
```json
{
  "code": 0,
  "msg": "ok",
  "tenant_access_token": "t-g40xxxxxx",
  "expire": 7200
}
```

### 鉴权方式

所有 API 请求在 HTTP Header 中携带 token：

```
Authorization: Bearer <tenant_access_token>
```

## 所需权限

### 最小权限集（第一版）

| 权限标识 | 说明 | 用途 |
|---------|------|------|
| `bitable:app` | 访问多维表格应用 | 读取/写入多维表格数据 |
| `bitable:app:readonly` | 只读访问多维表格 | 状态回流查询 |
| `drive:drive` | 云空间文件管理 | 创建/编辑飞书文档 |

### 后续扩展权限（第二版）

| 权限标识 | 说明 | 用途 |
|---------|------|------|
| `im:message` | 发送消息 | 群消息通知 |
| `im:chat` | 群组管理 | 查询群组信息 |

## API 端点

### Bitable 多维表格

| 操作 | 方法 | 端点 | 说明 |
|------|------|------|------|
| 创建记录 | POST | `/open-apis/bitable/v1/apps/:app_token/tables/:table_id/records` | 单条/批量创建 |
| 查询记录 | GET | `/open-apis/bitable/v1/apps/:app_token/tables/:table_id/records` | 支持 filter、page_size、page_token |
| 更新记录 | PUT | `/open-apis/bitable/v1/apps/:app_token/tables/:table_id/records/:record_id` | 全量更新 |
| 批量更新 | PUT | `/open-apis/bitable/v1/apps/:app_token/tables/:table_id/records/batch_update` | 批量更新多条 |

**批量创建/更新限制**: 单次最多 500 条记录

### 云文档

| 操作 | 方法 | 端点 | 说明 |
|------|------|------|------|
| 创建文档 | POST | `/open-apis/docx/v1/documents` | 创建飞书文档 |
| 写入内容 | PATCH | `/open-apis/docx/v1/documents/:document_id/blocks/:block_id/children` | 写入文档正文 |

### 群消息

| 操作 | 方法 | 端点 | 说明 |
|------|------|------|------|
| 发送消息 | POST | `/open-apis/im/v1/messages?receive_id_type=chat_id` | 发送富文本/文本消息 |

## 速率限制

### 通用限制

- **QPS**: 50 次/秒（单应用）
- **批量请求**: 每次最多 500 条记录
- **推荐并发**: 建议不超过 10 并发

### 具体端点限制

| 端点 | 限制 |
|------|------|
| 创建记录 | 100 次/分钟 |
| 查询记录 | 500 次/分钟 |
| 创建文档 | 100 次/分钟 |
| 发送消息 | 50 次/分钟 |

### 退避策略

- HTTP 429 (Too Many Requests): 指数退避重试，最多 3 次
  - 第 1 次: 等待 1s
  - 第 2 次: 等待 2s
  - 第 3 次: 等待 4s
- 超过 3 次仍失败: 记录错误并跳过，继续处理下一批

## 数据格式

### 记录创建格式

```json
{
  "fields": {
    "real_run_date": "2026-05-21",
    "simulated_date": "2016-10-04",
    "gmv": 1234.56,
    "order_count": 8,
    "low_review_rate": 0.15
  }
}
```

### 字段类型映射

| 本地类型 | 飞书字段类型 | 转换方式 |
|---------|-------------|---------|
| date (YYYY-MM-DD) | `11` (Date) | 直接传入字符串，飞书自动解析 |
| datetime (ISO 8601) | `13` (DateTime) | 毫秒级时间戳 |
| number (int/float) | `2` (Number) | 直接传入数字 |
| string (text) | `1` (Text) | 直接传入字符串 |
| boolean | `7` (Checkbox) | `true`/`false` |
| single_select | `5` (SingleSelect) | 传入选项 **name**（非 id） |
| text (long) | `1` (Text) | 直接传入，超长内容可截断 |

## 错误码处理

| 错误码 | 说明 | 处理策略 |
|--------|------|---------|
| 0 | 成功 | 继续 |
| 99991400 | app_token 无效 | 检查配置 |
| 99991403 | 权限不足 | 检查应用权限 |
| 99991663 | record_id 不存在 | 跳过 |
| 99991664 | 字段不存在 | 检查字段映射 |
| 170002 | 频率超限 | 指数退避重试 |

## 安全策略

1. **凭证存储**: app_id/app_secret 存储在 `.env`，不进入代码库
2. **Token 缓存**: 本地内存缓存，过期自动刷新
3. **日志脱敏**: 敏感信息（token）不进入日志
4. **Dry-Run 默认**: 脚本默认 `--dry-run`，必须显式 `--apply` 才真实写入

## 脚本架构

```
scripts/
├── feishu_client.py          # FeishuClient SDK (认证 + CRUD + 重试)
├── sync_feishu_bitable.py    # Bitable 同步脚本 (upsert + 审计)
├── publish_feishu_report.py  # 飞书文档发布
├── send_feishu_message.py    # 群消息发送
├── pull_feishu_status.py     # 状态回流
└── verify_feishu_tables.py   # 表验证脚本

config/
├── feishu_app.yml            # 应用配置 (app_id, app_secret)
├── feishu_table_ids.yml      # 表 ID 映射
└── feishu_field_mapping.yml  # 字段映射 (已有)
```

## 数据流

```
本地数据 (CSV/JSON)
    ↓ generate_feishu_sandbox.py
data/feishu/*.csv (沙盒格式)
    ↓ sync_feishu_bitable.py
飞书多维表格 (Bitable)
    ↓
飞书文档 (文档发布)
    ↓
飞书群消息 (通知)
    ↓
状态回流 (飞书 → 本地)
```

## 附录：飞书字段类型代码

| 类型 | 代码 | 说明 |
|------|------|------|
| Text | 1 | 文本 |
| Number | 2 | 数字 |
| DateTime | 13 | 日期时间 |
| Checkbox | 7 | 复选框 |
| SingleSelect | 5 | 单选 |
| MultiSelect | 4 | 多选 |

**参考文档**: https://open.feishu.cn/document/server-docs/docs/bitable-v1
