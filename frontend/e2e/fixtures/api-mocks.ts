import { Page, Route } from "@playwright/test"

// ─── Shared constants ───────────────────────────────────────────────
export const BEARER_TOKEN = "test-token-long-enough-for-your-app-32"

export const HEALTH_RESPONSE = {
  status: "ok",
  version: "0.6.0",
  db_connected: true,
}

export const STATUS_RESPONSE = {
  schema_version: "1.0.0",
  migration_version: "5",
  alert_count: 12,
  pipeline_run: { status: "completed", last_run: "2026-05-30T00:00:00Z" },
  table_counts: { ops_metric_alert: 12, audit_pipeline_run: 5, mart_metric_daily: 1500 },
  recent_errors: [],
}

// ─── Page-level auth setup ──────────────────────────────────────────
export async function setupAuth(page: Page) {
  await page.goto("/")
  await page.evaluate((token: string) => {
    sessionStorage.setItem("API_BEARER_TOKEN", token)
  }, BEARER_TOKEN)
}

// ─── Layout API mocks (health + status, called on every page) ───────
export async function mockLayoutApis(page: Page) {
  await page.route("**/api/v1/health", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(HEALTH_RESPONSE),
    })
  })
  await page.route("**/api/v1/status", async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(STATUS_RESPONSE),
    })
  })
}

// ─── Alerts mock data ───────────────────────────────────────────────
export const MOCK_ALERTS = {
  items: [
    {
      event_id: "evt-001",
      rule_id: "gmv_drop",
      event_date: "2026-05-30",
      severity: "high",
      metric_name: "gmv",
      object_type: "metric",
      object_id: "gmv_daily",
      current_value: 12000,
      baseline_value: 50000,
      change_rate: -0.76,
      owner_role: "ops",
      status: "new",
      impact_score: 85,
    },
    {
      event_id: "evt-002",
      rule_id: "late_delivery_spike",
      event_date: "2026-05-29",
      severity: "medium",
      metric_name: "late_delivery_rate",
      object_type: "metric",
      object_id: "late_delivery",
      current_value: 0.15,
      baseline_value: 0.05,
      change_rate: 2.0,
      owner_role: "logistics",
      status: "acknowledged",
      impact_score: 55,
    },
    {
      event_id: "evt-003",
      rule_id: "low_stock",
      event_date: "2026-05-28",
      severity: "low",
      metric_name: "stock_level",
      object_type: "product",
      object_id: "sku-123",
      current_value: 5,
      baseline_value: 50,
      change_rate: -0.9,
      owner_role: "inventory",
      status: "resolved",
      impact_score: 20,
    },
  ],
  total: 3,
}

// ─── Tasks mock data ────────────────────────────────────────────────
export const MOCK_TASKS = {
  items: [
    {
      task_id: "task-001",
      task_title: "Investigate GMV drop",
      task_description: "GMV dropped 76% in the last 24 hours",
      status: "todo",
      priority: "high",
      owner_role: "ops",
      owner_user_id: "user-001",
      due_at: "2026-06-01",
      created_at: "2026-05-30T10:00:00Z",
      recommendation_id: "rec-001",
      event_id: "evt-001",
      target_object_type: "metric",
      target_object_id: "gmv_daily",
    },
    {
      task_id: "task-002",
      task_title: "Review delivery SLA",
      task_description: "Late delivery rate exceeded threshold",
      status: "in_progress",
      priority: "medium",
      owner_role: "logistics",
      owner_user_id: "user-002",
      due_at: "2026-06-03",
      created_at: "2026-05-29T14:00:00Z",
      recommendation_id: "rec-002",
      event_id: "evt-002",
      target_object_type: "metric",
      target_object_id: "late_delivery",
    },
    {
      task_id: "task-003",
      task_title: "Restock SKU-123",
      task_description: null,
      status: "done",
      priority: "low",
      owner_role: "inventory",
      owner_user_id: "user-003",
      due_at: null,
      created_at: "2026-05-28T08:00:00Z",
      recommendation_id: null,
      event_id: "evt-003",
      target_object_type: "product",
      target_object_id: "sku-123",
    },
  ],
  total: 3,
}

// ─── Outbox mock data ───────────────────────────────────────────────
export const MOCK_OUTBOX = {
  items: [
    {
      outbox_id: "ob-001-abcdef123456",
      event_type: "alert_notification",
      source_type: "alert",
      source_id: "evt-001",
      target_channel: "feishu_cli",
      status: "pending",
      created_at: "2026-05-30T10:00:00Z",
      dispatch_attempts: 0,
      last_dispatch_at: null,
    },
    {
      outbox_id: "ob-002-ghijkl789012",
      event_type: "task_assignment",
      source_type: "task",
      source_id: "task-002",
      target_channel: "local_cli",
      status: "dispatched",
      created_at: "2026-05-29T14:00:00Z",
      dispatch_attempts: 1,
      last_dispatch_at: "2026-05-29T14:01:00Z",
    },
    {
      outbox_id: "ob-003-mnopqr345678",
      event_type: "governance_violation",
      source_type: "governance",
      source_id: "chk-001",
      target_channel: "manual",
      status: "failed",
      created_at: "2026-05-28T09:00:00Z",
      dispatch_attempts: 3,
      last_dispatch_at: "2026-05-28T09:05:00Z",
    },
  ],
  total: 3,
}

export const MOCK_DISPATCH_RESULT = {
  request_id: "req-dispatch-001",
  dry_run: true,
  processed: 2,
  results: [
    {
      outbox_id: "ob-001-abcdef123456",
      status: "preview",
      adapter_name: "feishu_adapter",
      message: "Would send to feishu",
      external_ref: null,
      error: null,
    },
    {
      outbox_id: "ob-002-ghijkl789012",
      status: "preview",
      adapter_name: "local_cli_adapter",
      message: "Would send to local CLI",
      external_ref: null,
      error: null,
    },
  ],
}

export const MOCK_DISPATCH_APPLY_RESULT = {
  request_id: "req-dispatch-002",
  dry_run: false,
  processed: 2,
  results: [
    {
      outbox_id: "ob-001-abcdef123456",
      status: "dispatched",
      adapter_name: "feishu_adapter",
      message: "Sent successfully",
      external_ref: "ext-feishu-001",
      error: null,
    },
    {
      outbox_id: "ob-002-ghijkl789012",
      status: "dispatched",
      adapter_name: "local_cli_adapter",
      message: "Sent successfully",
      external_ref: "ext-cli-001",
      error: null,
    },
  ],
}

// ─── Logs mock data ─────────────────────────────────────────────────
export const MOCK_ERROR_LOGS = {
  items: [
    {
      ts: "2026-05-30T10:30:00Z",
      level: "error",
      message: "Connection refused to database",
      request_id: "req-err-001-abcdef1234567890",
      error_code: "DB_CONN_REFUSED",
      diagnosis: "PostgreSQL instance is not reachable",
      suggested_action: "Check database container status",
      actor: "pipeline",
    },
    {
      ts: "2026-05-30T09:15:00Z",
      level: "error",
      message: "Feishu API rate limit exceeded",
      request_id: "req-err-002-ghijkl7890123456",
      error_code: "RATE_LIMIT",
      diagnosis: "Too many requests to feishu API",
      suggested_action: "Wait and retry",
      actor: "outbox",
    },
  ],
  total: 2,
}

export const MOCK_AUDIT_LOGS = {
  items: [
    {
      timestamp: "2026-05-30T10:00:00Z",
      outbox_id: "ob-001-abcdef123456",
      target_channel: "feishu_cli",
      adapter_name: "feishu_adapter",
      mode: "dry_run",
      status: "completed",
      external_ref: null,
      error: null,
      request_id: "req-audit-001",
      source: "outbox_dispatch",
    },
    {
      timestamp: "2026-05-29T14:00:00Z",
      outbox_id: "ob-002-ghijkl789012",
      target_channel: "local_cli",
      adapter_name: "local_cli_adapter",
      mode: "apply",
      status: "completed",
      external_ref: "ext-cli-001",
      error: null,
      request_id: "req-audit-002",
      source: "outbox_dispatch",
    },
  ],
  total: 2,
}

export const MOCK_RECENT_LOGS = {
  items: [
    {
      ts: "2026-05-30T10:30:00Z",
      level: "info",
      message: "Pipeline started",
      request_id: "req-recent-001",
      method: "POST",
      path: "/api/v1/pipeline/run",
      actor: "admin",
    },
    {
      ts: "2026-05-30T10:29:00Z",
      level: "info",
      message: "Alerts fetched",
      request_id: "req-recent-002",
      method: "GET",
      path: "/api/v1/alerts",
      actor: "system",
    },
  ],
  total: 2,
}

// ─── Feishu mock data ───────────────────────────────────────────────
export const MOCK_FEISHU_TABLE_RESULTS = [
  {
    name: "orders",
    status: "exported",
    rows: 1500,
    file: "data/feishu/orders.csv",
    created: 1500,
    updated: 0,
    pulled: 0,
    imported: 0,
    skipped: 0,
  },
  {
    name: "products",
    status: "exported",
    rows: 300,
    file: "data/feishu/products.csv",
    created: 300,
    updated: 0,
    pulled: 0,
    imported: 0,
    skipped: 0,
  },
]

export const MOCK_FEISHU_EXPORT = {
  status: "exported",
  message: "Exported 2 tables",
  tables: MOCK_FEISHU_TABLE_RESULTS,
}

export const MOCK_FEISHU_SYNC = {
  status: "synced",
  message: "Synced 2 tables to feishu",
  tables: [
    { ...MOCK_FEISHU_TABLE_RESULTS[0], status: "synced" },
    { ...MOCK_FEISHU_TABLE_RESULTS[1], status: "synced" },
  ],
}

export const MOCK_FEISHU_IMPORT = {
  status: "imported",
  message: "Imported 0 status changes",
  tables: [],
}

// ─── Pipeline mock data ─────────────────────────────────────────────
export const MOCK_PIPELINE_DAILY = {
  command: "python -m baxi.pipeline --type daily",
  pipeline_type: "daily",
  estimated_duration: "15 minutes",
  required_env_vars: ["DATABASE_URL", "OPENAI_API_KEY"],
  warnings: [],
  description: "8-step daily simulation",
}

export const MOCK_PIPELINE_FULL = {
  command: "python -m baxi.pipeline --type full --days 634",
  pipeline_type: "full",
  estimated_duration: "2 hours",
  required_env_vars: ["DATABASE_URL", "OPENAI_API_KEY", "FEISHU_TOKEN"],
  warnings: ["Full pipeline runs for 634 days of data"],
  description: "5-step full mode (634 days)",
}

export const MOCK_PIPELINE_DB_FULL = {
  command: "python -m baxi.pipeline --type db_full --dimensional",
  pipeline_type: "db_full",
  estimated_duration: "45 minutes",
  required_env_vars: ["DATABASE_URL"],
  warnings: [],
  description: "5-step DB mode with --dimensional",
}

// ─── Governance mock data ───────────────────────────────────────────
export const MOCK_CATALOG = {
  data_catalog: { version: "1.0" },
  assets: [
    {
      asset_id: "asset-001",
      asset_type: "table",
      name: "orders",
      location: "public.orders",
      description: "Customer orders",
      grain: "order_id",
      status: "active",
    },
    {
      asset_id: "asset-002",
      asset_type: "view",
      name: "daily_metrics",
      location: "mart.metric_daily",
      description: "Daily aggregated metrics",
      grain: "date",
      status: "active",
    },
  ],
}

export const MOCK_CLASSIFICATION = {
  classifications: [
    {
      asset_ref: "asset-001",
      level: "sensitive",
      rationale: "Contains PII data (customer names, addresses)",
      applies_to_fields: { customer_name: "pii", address: "pii" },
    },
    {
      asset_ref: "asset-002",
      level: "internal",
      rationale: "Aggregated metrics, no PII",
      applies_to_fields: undefined,
    },
  ],
}

export const MOCK_MARKINGS = {
  markings: {
    "mark-finance": {
      mandatory_control: true,
      access_type: "role_based",
      conjunctive: true,
      inheritance: ["mark-department"],
      applies_to: ["asset-001"],
      policy: "finance_team_only",
    },
  },
  pipeline_stage_markings: [],
}

export const MOCK_LINEAGE = {
  nodes: [
    { id: "asset-001", type: "table", label: "orders", status: "active" },
    { id: "asset-002", type: "view", label: "daily_metrics", status: "active" },
    { id: "asset-003", type: "table", label: "raw_events", status: "deprecated" },
  ],
  edges: [
    { from: "asset-001", to: "asset-002", transform: "daily_agg", transform_type: "aggregation" },
    { from: "asset-003", to: "asset-001", transform: "etl_clean", transform_type: "transformation" },
  ],
}

export const MOCK_CHECKPOINTS = {
  checkpoints: {
    "chk-001": {
      scope: "order_creation",
      endpoint: "/api/v1/decisions",
      requires_justification: true,
      prompt: "Justify the order amount change",
      checkpoint_types: ["threshold", "approval"],
    },
    "chk-002": {
      scope: "price_update",
      endpoint: undefined,
      requires_justification: false,
      prompt: undefined,
      checkpoint_types: ["audit"],
    },
  },
}

export const MOCK_GOVERNANCE_HEALTH = {
  monitoring_views: [
    {
      id: "mv-001",
      scope: "data_freshness",
      check_type: "staleness",
      rule: "data must be < 24h old",
      severity: "high",
    },
  ],
  health_checks: [
    {
      id: "hc-001",
      resource: "database",
      description: "Database connection healthy",
      check_type: "connectivity",
      severity: "critical",
      validation: "SELECT 1",
    },
    {
      id: "hc-002",
      resource: "pipeline",
      description: "Pipeline runs on schedule",
      check_type: "schedule",
      severity: "medium",
      validation: "last_run < 24h",
    },
  ],
}

// ─── Agent Logs mock data ───────────────────────────────────────────
export const MOCK_AGENT_LOGS = {
  items: [
    {
      execution_id: "exec-001",
      session_id: "sess-001",
      tool_name: "create_decision_case",
      status: "success",
      error_message: null,
      duration_ms: 1200,
      llm_model: "gpt-4o",
      llm_tokens: 2500,
      created_at: "2026-05-30T10:00:00Z",
    },
    {
      execution_id: "exec-002",
      session_id: "sess-001",
      tool_name: "generate_decision",
      status: "failed",
      error_message: "Timeout calling LLM",
      duration_ms: 30000,
      llm_model: "gpt-4o",
      llm_tokens: 0,
      created_at: "2026-05-30T10:01:00Z",
    },
    {
      execution_id: "exec-003",
      session_id: null,
      tool_name: "execute_action",
      status: "success",
      error_message: null,
      duration_ms: 500,
      llm_model: null,
      llm_tokens: null,
      created_at: "2026-05-30T09:50:00Z",
    },
  ],
  total: 3,
}

// ─── Case Detail mock data ──────────────────────────────────────────
export const MOCK_CASE = {
  decision_case_id: "case-001",
  status: "completed",
  object_type: "metric",
  object_id: "gmv_daily",
  source_type: "alert",
  source_id: "evt-001",
  severity: "high",
  context_hash: "abc123def456",
  created_at: "2026-05-30T10:00:00Z",
  updated_at: "2026-05-30T10:30:00Z",
  policy_results: {
    human_approval_required: true,
    allowed_actions: ["notify", "create_task"],
    blocked_actions: { delete_order: "Requires VP approval for orders > $10k" },
    risk_levels: { gmv_daily: "high", order_volume: "medium" },
    requires_approval_actions: ["update_pricing"],
    evidence_sources: ["gmv_drop_event", "historical_comparison"],
  },
}

export const MOCK_CASE_ACTION_EXECUTED = {
  ...MOCK_CASE,
  status: "action_executed",
}

// ─── Decision Review mock data ──────────────────────────────────────
export const MOCK_CASE_LIST = {
  cases: [
    {
      case_id: "case-001",
      status: "proposed",
      object_type: "metric",
      object_id: "gmv_daily",
      severity: "high",
      created_at: "2026-05-30T10:00:00Z",
    },
    {
      case_id: "case-002",
      status: "proposed",
      object_type: "metric",
      object_id: "late_delivery",
      severity: "medium",
      created_at: "2026-05-29T14:00:00Z",
    },
  ],
  total: 2,
}

export const MOCK_PROPOSALS = {
  case_id: "case-001",
  proposals: [
    {
      proposal_id: "prop-001",
      case_id: "case-001",
      decision_id: "dec-001",
      action_type: "create_task",
      title: "Create investigation task",
      description: "Investigate the root cause of GMV drop",
      risk_level: "low",
      requires_human_review: false,
      apply_status: "proposed",
      payload: { task_title: "Investigate GMV", priority: "high" },
      created_at: "2026-05-30T10:05:00Z",
    },
    {
      proposal_id: "prop-002",
      case_id: "case-001",
      decision_id: "dec-002",
      action_type: "update_pricing",
      title: "Adjust pricing strategy",
      description: "Reduce prices to recover GMV",
      risk_level: "high",
      requires_human_review: true,
      apply_status: "proposed",
      payload: { discount_pct: 15, affected_skus: ["sku-001", "sku-002"] },
      created_at: "2026-05-30T10:06:00Z",
    },
  ],
  count: 2,
}

export const MOCK_REVIEW_RECORD = {
  record_id: "review-001",
  proposal_id: "prop-001",
  verdict: "approved",
  reviewer_id: "supervisor",
  feedback: "LGTM, low risk",
  created_at: "2026-05-30T11:00:00Z",
}

// ─── Governance Status mock data (Policy Inspector) ─────────────────
export const MOCK_GOVERNANCE_STATUS = {
  overall_health: "healthy",
  version: "2.1.0",
  configs: {
    catalog: "loaded",
    classification: "loaded",
    markings: "loaded",
    lineage: "loaded",
    checkpoints: "loaded",
  },
}

// ─── Sandbox mock data ──────────────────────────────────────────────
export const MOCK_SANDBOXES = {
  items: [
    {
      sandbox_id: "sb-001",
      case_id: "case-001",
      proposal_id: "prop-001",
      data: { gmv_target: 50000 },
      status: "draft",
      compared_with: [],
      created_at: "2026-05-30T10:00:00Z",
    },
    {
      sandbox_id: "sb-002",
      case_id: "case-001",
      proposal_id: "prop-002",
      data: { gmv_target: 60000, pricing: "reduced" },
      status: "draft",
      compared_with: [],
      created_at: "2026-05-30T10:01:00Z",
    },
  ],
}

export const MOCK_COMPARISON = {
  sandbox_1_id: "sb-001",
  sandbox_2_id: "sb-002",
  differences: [
    { field: "gmv_target", value_1: 50000, value_2: 60000 },
    { field: "pricing", value_1: null, value_2: "reduced" },
  ],
}

// ─── Audit Timeline mock data ───────────────────────────────────────
export const MOCK_AUDIT_TIMELINE = {
  items: [
    {
      timestamp: "2026-05-30T10:00:00Z",
      outbox_id: "ob-001-abcdef123456",
      target_channel: "feishu_cli",
      adapter_name: "feishu_adapter",
      mode: "dry_run",
      status: "completed",
      external_ref: null,
      error: null,
      request_id: "req-tl-001",
      source: "outbox_dispatch",
    },
    {
      timestamp: "2026-05-30T10:05:00Z",
      outbox_id: "ob-001-abcdef123456",
      target_channel: "feishu_cli",
      adapter_name: "feishu_adapter",
      mode: "apply",
      status: "completed",
      external_ref: "ext-feishu-001",
      error: null,
      request_id: "req-tl-002",
      source: "outbox_dispatch",
    },
    {
      timestamp: "2026-05-29T14:00:00Z",
      outbox_id: "ob-002-ghijkl789012",
      target_channel: "local_cli",
      adapter_name: "local_cli_adapter",
      mode: "apply",
      status: "failed",
      external_ref: null,
      error: "Connection timeout",
      request_id: "req-tl-003",
      source: "outbox_dispatch",
    },
  ],
  total: 3,
}

// ─── Helper: register all mocks for a page ─────────────────────────
export async function mockAllApis(page: Page) {
  await mockLayoutApis(page)

  // Alerts
  await page.route("**/api/v1/alerts**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_ALERTS),
    })
  })

  // Tasks
  await page.route("**/api/v1/tasks**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_TASKS),
    })
  })

  // Outbox
  await page.route("**/api/v1/outbox/dispatch", async (route) => {
    if (route.request().method() === "POST") {
      const body = JSON.parse(route.request().postData() || "{}")
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(body.dry_run ? MOCK_DISPATCH_RESULT : MOCK_DISPATCH_APPLY_RESULT),
      })
    } else {
      await route.fulfill({ status: 405 })
    }
  })
  await page.route("**/api/v1/outbox**", async (route) => {
    if (route.request().url().includes("/dispatch")) return
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_OUTBOX),
    })
  })

  // Logs
  await page.route("**/api/v1/logs/errors**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_ERROR_LOGS),
    })
  })
  await page.route("**/api/v1/logs/audit**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_AUDIT_LOGS),
    })
  })
  await page.route("**/api/v1/logs/recent**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_RECENT_LOGS),
    })
  })
  await page.route("**/api/v1/logs/agent**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_AGENT_LOGS),
    })
  })

  // Feishu
  await page.route("**/api/v1/feishu/export", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_FEISHU_EXPORT),
      })
    } else {
      await route.fulfill({ status: 405 })
    }
  })
  await page.route("**/api/v1/feishu/sync", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_FEISHU_SYNC),
      })
    } else {
      await route.fulfill({ status: 405 })
    }
  })
  await page.route("**/api/v1/feishu/status/import", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_FEISHU_IMPORT),
      })
    } else {
      await route.fulfill({ status: 405 })
    }
  })

  // Pipeline
  await page.route("**/api/v1/pipeline/run", async (route) => {
    if (route.request().method() === "POST") {
      const body = JSON.parse(route.request().postData() || "{}")
      const mockData =
        body.pipeline_type === "full"
          ? MOCK_PIPELINE_FULL
          : body.pipeline_type === "db_full"
            ? MOCK_PIPELINE_DB_FULL
            : MOCK_PIPELINE_DAILY
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(mockData),
      })
    } else {
      await route.fulfill({ status: 405 })
    }
  })

  // Governance
  await page.route("**/api/v1/governance/catalog**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_CATALOG),
    })
  })
  await page.route("**/api/v1/governance/classification**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_CLASSIFICATION),
    })
  })
  await page.route("**/api/v1/governance/markings**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_MARKINGS),
    })
  })
  await page.route("**/api/v1/governance/lineage**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_LINEAGE),
    })
  })
  await page.route("**/api/v1/governance/checkpoints**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_CHECKPOINTS),
    })
  })
  await page.route("**/api/v1/governance/health**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_GOVERNANCE_HEALTH),
    })
  })
  await page.route("**/api/v1/governance/status**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_GOVERNANCE_STATUS),
    })
  })

  // Decisions / Cases
  await page.route("**/api/v1/decisions/cases**", async (route) => {
    const url = route.request().url()
    if (url.includes("/proposals")) {
      // Proposals for a specific case
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_PROPOSALS),
      })
    } else if (url.match(/\/cases\/[^/]+$/)) {
      // Single case detail
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CASE),
      })
    } else {
      // Case list
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CASE_LIST),
      })
    }
  })

  // Proposals review
  await page.route("**/api/v1/proposals/**", async (route) => {
    const url = route.request().url()
    if (url.includes("/review")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_REVIEW_RECORD),
      })
    } else if (url.includes("/approve") || url.includes("/reject") || url.includes("/cancel")) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          record_id: "review-new",
          proposal_id: "prop-001",
          verdict: url.includes("/approve") ? "approved" : url.includes("/reject") ? "rejected" : "cancelled",
          reviewer_id: "operator",
          feedback: "",
          created_at: new Date().toISOString(),
        }),
      })
    } else {
      await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
    }
  })

  // Sandboxes
  await page.route("**/api/v1/sandboxes/compare**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(MOCK_COMPARISON),
    })
  })
  await page.route("**/api/v1/sandboxes/**", async (route) => {
    const url = route.request().url()
    if (url.includes("/compare")) return
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          sandbox_id: "sb-new",
          case_id: "case-001",
          data: {},
          status: "draft",
          compared_with: [],
          created_at: new Date().toISOString(),
        }),
      })
    } else {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_SANDBOXES),
      })
    }
  })
}
