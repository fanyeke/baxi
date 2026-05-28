import { useQuery } from "@tanstack/react-query"
import { useParams } from "react-router-dom"
import { apiClient } from "../api/client"
import type { AuditLogListResponse, AuditLogEntry } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

function StatusIcon({ status, error }: { status: string | null; error: string | null }) {
  if (error) {
    return <span className="text-red-500 text-lg">✕</span>
  }
  if (status === "completed" || status === "success") {
    return <span className="text-green-500 text-lg">✓</span>
  }
  if (status === "pending" || status === "processing") {
    return <span className="text-gray-400 text-lg">●</span>
  }
  return <span className="text-gray-400 text-lg">●</span>
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit", second: "2-digit" })
}

function formatDate(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric" })
}

function groupByDate(events: AuditLogEntry[]): Map<string, AuditLogEntry[]> {
  const groups = new Map<string, AuditLogEntry[]>()

  for (const event of events) {
    const dateKey = formatDate(event.timestamp)
    if (!groups.has(dateKey)) {
      groups.set(dateKey, [])
    }
    groups.get(dateKey)!.push(event)
  }

  return groups
}

export default function AuditTimeline() {
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, error } = useQuery({
    queryKey: ["audit-timeline", id],
    queryFn: () => apiClient.get<AuditLogListResponse>("/logs/audit", { limit: "100" }),
  })

  if (isLoading) return <LoadingSkeleton type="text" count={5} />
  if (error) return <ErrorPanel title="加载失败" message={error.message || "Failed to load audit timeline"} />
  if (!data || data.items.length === 0) return <EmptyState title="暂无审计日志" />

  const sortedEvents = [...data.items].sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())

  const groupedEvents = groupByDate(sortedEvents)
  const dateGroups = Array.from(groupedEvents.entries())

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">审计时间线</h1>
        {id && <span className="text-sm text-muted-foreground font-mono">Case: {id}</span>}
      </div>

      <div className="space-y-8">
        {dateGroups.map(([date, events]) => (
          <div key={date} className="space-y-4">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-muted-foreground bg-muted px-2 py-1 rounded">{date}</span>
              <div className="flex-1 h-px bg-border" />
            </div>

            <div className="relative pl-4">
              <div className="absolute left-6 top-0 bottom-0 w-px bg-border" />

              <div className="space-y-4">
                {events.map((event, index) => (
                  <div key={event.outbox_id + index} className="relative flex gap-4">
                    <div className="relative z-10 flex items-center justify-center w-6 h-6 -ml-3 bg-background rounded-full">
                      <StatusIcon status={event.status} error={event.error} />
                    </div>

                    <div className="flex-1 pb-4">
                      <div className="flex items-start gap-3">
                        <div className="flex-1 space-y-1">
                          <div className="flex items-center gap-2">
                            <span className="text-sm font-medium">{formatTime(event.timestamp)}</span>
                            <span className="text-xs px-1.5 py-0.5 bg-muted rounded text-muted-foreground">
                              {event.source}
                            </span>
                            {event.mode && (
                              <span className="text-xs px-1.5 py-0.5 bg-blue-50 text-blue-700 rounded">
                                {event.mode}
                              </span>
                            )}
                          </div>

                          <div className="text-sm text-foreground">
                            <span className="font-medium">{event.target_channel}</span>
                            <span className="text-muted-foreground"> via </span>
                            <span>{event.adapter_name}</span>
                          </div>

                          <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                            <span className="font-mono bg-muted px-1.5 py-0.5 rounded">
                              ID: {event.outbox_id.slice(0, 8)}...
                            </span>
                            {event.status && (
                              <span className="px-1.5 py-0.5 rounded border">
                                {event.status}
                              </span>
                            )}
                            {event.external_ref && (
                              <span className="font-mono">Ref: {event.external_ref}</span>
                            )}
                          </div>

                          {event.error && (
                            <div className="text-xs text-red-600 bg-red-50 p-2 rounded mt-1">
                              {event.error}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
