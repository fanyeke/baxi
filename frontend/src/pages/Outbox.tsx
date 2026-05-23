import { useState } from "react"
import { useQuery, useMutation } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { OutboxListResponse, DispatchResponse } from "../api/types"

export default function Outbox() {
  const [status, setStatus] = useState("")
  const [channel, setChannel] = useState("")
  const [results, setResults] = useState<DispatchResponse | null>(null)
  const [showConfirm, setShowConfirm] = useState(false)

  const params: Record<string, string> = { limit: "100" }
  if (status) params.status = status
  if (channel) params.channel = channel

  const { data, isLoading, refetch } = useQuery({
    queryKey: ["outbox", status, channel],
    queryFn: () => apiClient.get<OutboxListResponse>("/outbox", params),
  })

  const dryRunMutation = useMutation({
    mutationFn: () => apiClient.post<DispatchResponse>("/outbox/dispatch", { dry_run: true, channel: channel || undefined, limit: 100 }),
    onSuccess: (data) => setResults(data),
  })

  const applyMutation = useMutation({
    mutationFn: () => apiClient.post<DispatchResponse>("/outbox/dispatch", { apply: true, channel: channel || undefined, limit: 100 }),
    onSuccess: (data) => {
      setResults(data)
      setShowConfirm(false)
      refetch()
    },
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Outbox 分发</h1>

      <div className="flex gap-2">
        <select className="px-3 py-1 border rounded text-sm" value={status} onChange={e => setStatus(e.target.value)}>
          <option value="">全部状态</option>
          <option value="pending">待处理</option>
          <option value="dispatched">已分发</option>
          <option value="failed">失败</option>
        </select>
        <select className="px-3 py-1 border rounded text-sm" value={channel} onChange={e => setChannel(e.target.value)}>
          <option value="">全部通道</option>
          <option value="feishu_cli">飞书</option>
          <option value="local_cli">本地 CLI</option>
          <option value="manual">人工</option>
        </select>
      </div>

      <div className="flex gap-2">
        <button
          onClick={() => dryRunMutation.mutate()}
          disabled={dryRunMutation.isPending}
          className="px-4 py-2 bg-primary text-primary-foreground rounded text-sm hover:opacity-90 disabled:opacity-50"
        >
          {dryRunMutation.isPending ? "执行中..." : "Dry-Run 分发"}
        </button>
        <button
          onClick={() => setShowConfirm(true)}
          className="px-4 py-2 border border-destructive text-destructive rounded text-sm hover:bg-destructive/10"
        >
          真实分发
        </button>
      </div>

      {showConfirm && (
        <div className="p-4 border border-destructive rounded-lg bg-destructive/5">
          <p className="font-semibold text-destructive text-sm">确认执行？</p>
          <p className="text-xs text-muted-foreground mt-1">这将执行真实的分发操作，不可撤销。</p>
          <div className="flex gap-2 mt-2">
            <button onClick={() => applyMutation.mutate()} className="px-3 py-1 bg-destructive text-destructive-foreground rounded text-xs">确认执行</button>
            <button onClick={() => setShowConfirm(false)} className="px-3 py-1 border rounded text-xs">取消</button>
          </div>
        </div>
      )}

      {results && (
        <div className="p-3 border rounded-lg bg-muted/30">
          <p className="text-sm font-medium">分发结果: {results.dry_run ? "Dry-Run" : "Apply"} — {results.processed} 条</p>
          <div className="mt-2 space-y-1">
            {results.results.map(r => (
              <div key={r.outbox_id} className="text-xs flex gap-2">
                <span className="font-mono">{r.outbox_id.slice(0,8)}</span>
                <span className={r.status === "preview" ? "text-blue-600" : r.status === "dispatched" ? "text-green-600" : "text-red-600"}>{r.status}</span>
                <span className="text-muted-foreground">{r.adapter_name || ""}</span>
                {r.error && <span className="text-red-500">{r.error}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      {isLoading && <p className="text-muted-foreground">加载中...</p>}
      {data && data.items.length === 0 && <p className="text-muted-foreground">暂无 outbox 事件</p>}
      {data && data.items.length > 0 && (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="p-2 text-left">ID</th>
                <th className="p-2 text-left">事件类型</th>
                <th className="p-2 text-left">通道</th>
                <th className="p-2 text-left">状态</th>
                <th className="p-2 text-left">创建时间</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map(item => (
                <tr key={item.outbox_id} className="border-t hover:bg-muted/50">
                  <td className="p-2 font-mono text-xs">{item.outbox_id.slice(0,8)}</td>
                  <td className="p-2">{item.event_type}</td>
                  <td className="p-2">{item.target_channel}</td>
                  <td className="p-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      item.status === "dispatched" ? "bg-green-100 text-green-700" :
                      item.status === "failed" ? "bg-red-100 text-red-700" :
                      "bg-yellow-100 text-yellow-700"
                    }`}>{item.status}</span>
                  </td>
                  <td className="p-2">{item.created_at}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
