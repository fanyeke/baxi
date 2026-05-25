# core/config.py — 核心配置模块

> 集中管理项目路径常量、环境变量读取和 Feishu 凭据加载。

## 设计原则

- **单点配置**：所有路径、环境变量、凭据都从这里导入
- **向后兼容**：`scripts/config.py` 是已弃用的 shim，新代码应直接 `from core.config import ...`
- **占位符检测**：`get_env_or_raise()` 会拒绝 `YOUR_*` 和 `REPLACE_ME` 占位符值

---

## 路径常量

```python
from core.config import PROJECT_ROOT, RAW_DATA_DIR, DB_PATH, OUTPUT_DIR
```

| 常量 | 值 | 说明 |
|------|-----|------|
| `PROJECT_ROOT` | `Path(__file__).parent.parent` | 项目根目录 |
| `RAW_DATA_DIR` | `PROJECT_ROOT / "data" / "raw"` | 原始 CSV |
| `DB_PATH` | `PROJECT_ROOT / "data" / "olist_ops.db"` | SQLite 数据库 |
| `OUTPUT_DIR` | `PROJECT_ROOT / "outputs"` | 分析产物输出 |
| `SQL_DIR` | `PROJECT_ROOT / "sql"` | Schema + 迁移 |
| `FEISHU_DIR` | `PROJECT_ROOT / "data" / "feishu"` | 飞书 CSV 导出 |
| `SYSTEM_DIR` | `PROJECT_ROOT / "data" / "system"` | 运行时状态 |

---

## 环境变量读取

### `get_env_or_raise(key: str, default=None) -> str`

```python
from core.config import get_env_or_raise

# 必须设置的变量（缺失会抛 RuntimeError）
api_token = get_env_or_raise("API_BEARER_TOKEN")

# 有默认值的变量（缺失返回默认值，不报错）
user = get_env_or_raise("DEFAULT_USER", default="qoder")
```

**占位符检测**：如果值为空、以 `YOUR_` 开头、或等于 `REPLACE_ME`，则抛出 `RuntimeError`。

### 完整环境变量列表

| 变量 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `API_BEARER_TOKEN` | ✅ | — | API 认证令牌 |
| `FEISHU_APP_ID` | ❌ | — | 飞书 App ID |
| `FEISHU_APP_SECRET` | ❌ | — | 飞书 App Secret |
| `FEISHU_BASE_APP_TOKEN` | ❌ | — | 飞书多维表格 App Token |
| `FEISHU_CHAT_ID` | ❌ | — | 飞书群聊 ID |
| `LLM_API_KEY` | ❌ | — | LLM API Key（OpenAI-compatible） |
| `DEFAULT_USER` | ❌ | `qoder` | 默认用户身份 |
| `CORS_ORIGINS` | ❌ | `http://localhost:5173` | CORS 来源（逗号分隔） |
| `TRUSTED_PROXY_IPS` | ❌ | `127.0.0.1,::1` | 可信代理 IP |
| `ENABLE_DOCS` | ❌ | `0` | 启用 Swagger/OpenAPI（`1`=启用） |
| `DEBUG` | ❌ | `0` | 调试模式（`1`=显示完整堆栈） |

---

## Feishu 凭据加载

### `load_feishu_credentials() -> dict`

```python
from core.config import load_feishu_credentials

creds = load_feishu_credentials()
# {'app_id': 'xxx', 'app_secret': 'yyy', 'base_app_token': 'zzz', 'chat_id': 'aaa'}
```

**优先级**：环境变量 > `config/feishu_table_ids.yml`

---

## SQL 安全校验

### `validate_sql_identifier(name: str) -> str`

```python
from core.config import validate_sql_identifier

safe = validate_sql_identifier("my_table")  # "my_table"
safe = validate_sql_identifier("drop table")  # RuntimeError
```

用于防止 SQL 注入，确保表名/列名只包含字母、数字和下划线。

---

## 向后兼容

```python
# ❌ 已弃用（会发出 DeprecationWarning）
from scripts import config

# ✅ 推荐
from core.config import PROJECT_ROOT, get_env_or_raise, DB_PATH
```

`scripts/config.py` 保留为 shim，通过动态 `sys.path` 调整重新导出 `core.config` 的所有符号，但会在导入时发出 `DeprecationWarning`。
