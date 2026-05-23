import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { AlertListResponse } from "../api/types"

export default function Alerts() {
  const [severity, setSeverity] = useState("")
  const [status, setStatus] = useState("")
  const params: Record<string, string> = { limit: "100" }
  if (severity) params.severity = severity
  if (status) params.status = status

  const { data, isLoading, error } = useQuery({
    queryKey: ["alerts", severity, status],
    queryFn: () => apiClient.get<AlertListResponse>("/alerts", params),
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">告警中心</h1>
      <div className="flex gap-2">
        <select className="px-3 py-1 border rounded text-sm" value={severity} onChange={e => setSeverity(e.target.value)}>
          <option value="">全部等级</option>
          <option value="high">高</option>
          <option value="medium">中</option>
          <option value="low">低</option>
        </select>
        <select className="px-3 py-1 border rounded text-sm" value={status} onChange={e => setStatus(e.target.value)}>
          <option value="">全部状态</option>
          <option value="new">新</option>
          <option value="acknowledged">已确认</option>
          <option value="resolved">已解决</option>
        </select>
      </div>

      {isLoading && <p className="text-muted-foreground">加载中...</p>}
      {error && <p className="text-destructive">加载失败</p>}
      {data && data.items.length === 0 && <p className="text-muted-foreground">暂无告警</p>}
      {data && data.items.length > 0 && (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="p-2 text-left">日期</th>
                <th className="p-2 text-left">规则</th>
                <th className="p-2 text-left">等级</th>
                <th className="p-2 text-left">对象</th>
                <th className="p-2 text-left">负责人</th>
                <th className="p-2 text-left">状态</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map(item => (
                <tr key={item.event_id} className="border-t hover:bg-muted/50">
                  <td className="p-2">{item.event_date}</td>
                  <td className="p-2 font-mono text-xs">{item.rule_id}</td>
                  <td className="p-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      item.severity === "high" ? "bg-red-100 text-red-700" :
                      item.severity === "medium" ? "bg-yellow-100 text-yellow-700" :
                      "bg-green-100 text-green-700"
                    }`}>{item.severity}</span>
                  </td>
                  <td className="p-2">{item.object_type}/{item.object_id}</td>
                  <td className="p-2">{item.owner_role}</td>
                  <td className="p-2">{item.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
