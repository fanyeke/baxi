import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { HealthResponse, StatusResponse, AlertListResponse, TaskListResponse, OutboxListResponse } from "../api/types"

export default function Dashboard() {
  const health = useQuery({ queryKey: ["health"], queryFn: () => apiClient.get<HealthResponse>("/health") })
  const status = useQuery({ queryKey: ["status"], queryFn: () => apiClient.get<StatusResponse>("/status") })
  const alerts = useQuery({ queryKey: ["alerts"], queryFn: () => apiClient.get<AlertListResponse>("/alerts?limit=1") })
  const tasks = useQuery({ queryKey: ["tasks"], queryFn: () => apiClient.get<TaskListResponse>("/tasks?limit=1") })
  const outbox = useQuery({ queryKey: ["outbox-pending"], queryFn: () => apiClient.get<OutboxListResponse>("/outbox?status=pending&limit=1") })

  const isLoading = health.isLoading || status.isLoading
  const error = health.error || status.error

  if (isLoading) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">系统总览</h1>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[1,2,3,4,5,6].map(i => (
            <div key={i} className="h-24 bg-muted animate-pulse rounded-lg" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    const err = error as { message?: string }
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">系统总览</h1>
        <div className="p-4 border border-destructive/50 bg-destructive/10 rounded-lg">
          <p className="font-semibold text-destructive">连接失败</p>
          <p className="text-sm text-muted-foreground mt-1">{err.message || "无法连接到 API 服务"}</p>
          <p className="text-xs text-muted-foreground mt-1">请确认 API 服务运行在 localhost:8765，且 Token 已配置</p>
        </div>
      </div>
    )
  }

  const h = health.data
  const s = status.data

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">系统总览</h1>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <StatCard label="数据库状态" value={h?.db_connected ? "OK" : "Disconnected"} color={h?.db_connected ? "text-green-600" : "text-red-600"} />
        <StatCard label="告警总数" value={String(alerts.data?.total ?? "—")} />
        <StatCard label="任务总数" value={String(tasks.data?.total ?? "—")} />
        <StatCard label="Outbox Pending" value={String(outbox.data?.total ?? "—")} />
        <StatCard label="API 版本" value={h?.version ?? "—"} />
        <StatCard label="上次 Pipeline" value={String(s?.last_pipeline_run?.status ?? "N/A")} color={s?.last_pipeline_run?.status === "success" ? "text-green-600" : "text-muted-foreground"} />
      </div>
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div className="p-4 border rounded-lg bg-card">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className={`text-2xl font-bold mt-1 ${color ?? "text-foreground"}`}>{value}</p>
    </div>
  )
}
