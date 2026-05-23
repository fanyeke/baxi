import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { ErrorLogListResponse, AuditLogListResponse, RecentLogListResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"

type Tab = "errors" | "audit" | "recent"

export default function Logs() {
  const [tab, setTab] = useState<Tab>("errors")

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">日志诊断</h1>

      <div className="flex gap-1 border-b">
        {[["errors", "错误日志"], ["audit", "审计日志"], ["recent", "最近请求"]].map(([key, label]) => (
          <button
            key={key}
            onClick={() => setTab(key as Tab)}
            className={`px-4 py-2 text-sm border-b-2 -mb-px transition-colors ${
              tab === key ? "border-primary font-medium" : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {tab === "errors" && <ErrorsTab />}
      {tab === "audit" && <AuditTab />}
      {tab === "recent" && <RecentTab />}
    </div>
  )
}

function ErrorsTab() {
  const { data, isLoading } = useQuery({
    queryKey: ["log-errors"],
    queryFn: () => apiClient.get<ErrorLogListResponse>("/logs/errors", { limit: "100" }),
  })

  if (isLoading) return <LoadingSkeleton type="table" count={5} />
  if (!data || data.items.length === 0) return <EmptyState title="暂无错误日志" />

  return (
    <div className="border rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            <th className="p-2 text-left">时间</th>
            <th className="p-2 text-left">错误码</th>
            <th className="p-2 text-left">消息</th>
            <th className="p-2 text-left">诊断</th>
            <th className="p-2 text-left">Request ID</th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((entry, i) => (
            <tr key={i} className="border-t hover:bg-muted/50">
              <td className="p-2 text-xs">{entry.ts}</td>
              <td className="p-2"><span className="font-mono text-xs bg-red-50 text-red-700 px-1 rounded">{entry.error_code}</span></td>
              <td className="p-2">{entry.message}</td>
              <td className="p-2 text-xs text-muted-foreground">{entry.diagnosis}</td>
              <td className="p-2 font-mono text-xs">{entry.request_id.slice(0,16)}...</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function AuditTab() {
  const { data, isLoading } = useQuery({
    queryKey: ["log-audit"],
    queryFn: () => apiClient.get<AuditLogListResponse>("/logs/audit", { limit: "100" }),
  })

  if (isLoading) return <LoadingSkeleton type="table" count={3} />
  if (!data || data.items.length === 0) return <EmptyState title="暂无审计日志" />

  return (
    <div className="border rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            <th className="p-2 text-left">时间</th>
            <th className="p-2 text-left">操作</th>
            <th className="p-2 text-left">模式</th>
            <th className="p-2 text-left">状态</th>
            <th className="p-2 text-left">Outbox</th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((entry, i) => (
            <tr key={i} className="border-t hover:bg-muted/50">
              <td className="p-2 text-xs">{entry.timestamp}</td>
              <td className="p-2">{entry.target_channel}</td>
              <td className="p-2">{entry.mode || "—"}</td>
              <td className="p-2">{entry.status || "—"}</td>
              <td className="p-2 font-mono text-xs">{entry.outbox_id?.slice(0,8) || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function RecentTab() {
  const { data, isLoading } = useQuery({
    queryKey: ["log-recent"],
    queryFn: () => apiClient.get<RecentLogListResponse>("/logs/recent", { limit: "50" }),
  })

  if (isLoading) return <LoadingSkeleton type="text" count={5} />
  if (!data || data.items.length === 0) return <EmptyState title="暂无请求日志" />

  return (
    <div className="border rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            <th className="p-2 text-left">时间</th>
            <th className="p-2 text-left">方法</th>
            <th className="p-2 text-left">路径</th>
            <th className="p-2 text-left">操作者</th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((entry, i) => (
            <tr key={i} className="border-t hover:bg-muted/50">
              <td className="p-2 text-xs">{entry.ts}</td>
              <td className="p-2"><span className="font-mono text-xs bg-muted px-1 rounded">{entry.method}</span></td>
              <td className="p-2 font-mono text-xs">{entry.path}</td>
              <td className="p-2">{entry.actor}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
