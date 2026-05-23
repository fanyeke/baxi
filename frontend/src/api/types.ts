export interface HealthResponse {
  status: string
  version: string
  db_connected: boolean
}

export interface StatusResponse {
  database: Record<string, unknown>
  last_pipeline_run: Record<string, unknown> | null
  version: string
}

export interface AlertItem {
  event_id: string
  rule_id: string
  event_date: string
  severity: string
  metric_name: string
  object_type: string
  object_id: string
  current_value: number | null
  baseline_value: number | null
  change_rate: number | null
  owner_role: string
  status: string
  impact_score: number | null
}

export interface AlertListResponse {
  items: AlertItem[]
  total: number
}

export interface TaskItem {
  task_id: string
  task_title: string
  task_description: string | null
  status: string
  priority: string
  owner_role: string
  owner_user_id: string
  due_at: string | null
  created_at: string
  recommendation_id: string | null
  event_id: string | null
  target_object_type: string | null
  target_object_id: string | null
}

export interface TaskListResponse {
  items: TaskItem[]
  total: number
}

export interface OutboxItem {
  outbox_id: string
  event_type: string
  source_type: string
  source_id: string
  target_channel: string
  status: string
  created_at: string
  dispatch_attempts: number
  last_dispatch_at: string | null
}

export interface OutboxListResponse {
  items: OutboxItem[]
  total: number
}

export interface DispatchResultItem {
  outbox_id: string
  status: string
  adapter_name: string | null
  message: string | null
  external_ref: string | null
  error: string | null
}

export interface DispatchResponse {
  request_id: string
  dry_run: boolean
  processed: number
  results: DispatchResultItem[]
}

export interface ErrorLogEntry {
  ts: string
  level: string
  message: string
  request_id: string
  error_code: string
  diagnosis: string
  suggested_action: string
  actor: string
}

export interface ErrorLogListResponse {
  items: ErrorLogEntry[]
  total: number
}

export interface AuditLogEntry {
  timestamp: string
  outbox_id: string
  target_channel: string
  adapter_name: string
  mode: string | null
  status: string | null
  external_ref: string | null
  error: string | null
  request_id: string | null
  source: string
}

export interface AuditLogListResponse {
  items: AuditLogEntry[]
  total: number
}

export interface RecentLogEntry {
  ts: string
  level: string
  message: string
  request_id: string
  method: string
  path: string
  actor: string
}

export interface RecentLogListResponse {
  items: RecentLogEntry[]
  total: number
}

export interface FeishuTableResult {
  name: string
  status: string
  rows: number
  file: string
  created: number
  updated: number
  pulled: number
  imported: number
  skipped: number
}

export interface FeishuExportResponse {
  status: string
  message: string
  tables: FeishuTableResult[]
}

export interface FeishuSyncResponse {
  status: string
  message: string
  tables: FeishuTableResult[]
}

export interface FeishuStatusImportResponse {
  status: string
  message: string
  tables: FeishuTableResult[]
}

export interface PipelineRunResponse {
  command: string
  pipeline_type: string
  estimated_duration: string
  required_env_vars: string[]
  warnings: string[]
  description: string
}
