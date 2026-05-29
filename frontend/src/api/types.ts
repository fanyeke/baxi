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

export interface AgentExecutionLog {
  execution_id: string
  session_id: string | null
  tool_name: string
  status: string
  error_message: string | null
  duration_ms: number | null
  llm_model: string | null
  llm_tokens: number | null
  created_at: string
}

export interface AgentLogListResponse {
  items: AgentExecutionLog[]
  total: number
}

export interface PolicyResults {
  human_approval_required: boolean
  allowed_actions: string[]
  blocked_actions: string[]
  risk_levels: Record<string, string>
  requires_approval_actions: string[]
  evidence_sources: string[]
}

export interface DecisionCaseResponse {
  decision_case_id: string
  status: string
  object_type: string
  object_id: string
  source_type: string
  source_id: string
  severity: string
  context_hash: string
  created_at: string
  updated_at: string
  policy_results: PolicyResults
}

export interface GovernanceStatusResponse {
  overall_health: string
  version: string
  configs: Record<string, string>
}

// --- Decision Review types ---

export interface ActionProposal {
  proposal_id: string
  case_id: string
  decision_id: string
  action_type: string
  title: string
  description: string
  risk_level: string
  requires_human_review: boolean
  apply_status: string
  payload: Record<string, unknown>
  created_at: string
}

export interface ProposalListResponse {
  case_id: string
  proposals: ActionProposal[]
  count: number
}

export interface ReviewRecord {
  record_id: string
  proposal_id: string
  verdict: string
  feedback: string
  reviewer_id: string
  created_at: string
}

export interface ReviewResponse {
  record_id: string
  proposal_id: string
  verdict: string
  reviewer_id: string
  feedback: string
  created_at: string
}

// --- Sandbox types ---

export interface Sandbox {
  sandbox_id: string
  case_id: string
  proposal_id?: string
  data: Record<string, unknown>
  status: string
  compared_with: string[]
  created_at: string
}

export interface ComparisonResult {
  sandbox_1_id: string
  sandbox_2_id: string
  differences: Array<{
    field: string
    value_1: unknown
    value_2: unknown
  }>
}

export interface CaseListResponse {
  cases: Array<{
    case_id: string
    status: string
    object_type: string
    object_id: string
    severity: string
    created_at: string
  }>
  total: number
}
