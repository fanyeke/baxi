import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { TaskListResponse } from "../api/types"

export default function Tasks() {
  const [status, setStatus] = useState("")
  const [priority, setPriority] = useState("")
  const params: Record<string, string> = { limit: "100" }
  if (status) params.status = status
  if (priority) params.priority = priority

  const { data, isLoading } = useQuery({
    queryKey: ["tasks", status, priority],
    queryFn: () => apiClient.get<TaskListResponse>("/tasks", params),
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">任务中心</h1>
      <div className="flex gap-2">
        <select className="px-3 py-1 border rounded text-sm" value={status} onChange={e => setStatus(e.target.value)}>
          <option value="">全部状态</option>
          <option value="todo">待办</option>
          <option value="in_progress">进行中</option>
          <option value="done">已完成</option>
        </select>
        <select className="px-3 py-1 border rounded text-sm" value={priority} onChange={e => setPriority(e.target.value)}>
          <option value="">全部优先级</option>
          <option value="high">高</option>
          <option value="medium">中</option>
          <option value="low">低</option>
        </select>
      </div>

      {isLoading && <p className="text-muted-foreground">加载中...</p>}
      {data && data.items.length === 0 && <p className="text-muted-foreground">暂无任务</p>}
      {data && data.items.length > 0 && (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="p-2 text-left">标题</th>
                <th className="p-2 text-left">负责人</th>
                <th className="p-2 text-left">优先级</th>
                <th className="p-2 text-left">截止</th>
                <th className="p-2 text-left">状态</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map(item => (
                <tr key={item.task_id} className="border-t hover:bg-muted/50">
                  <td className="p-2 font-medium">{item.task_title}</td>
                  <td className="p-2">{item.owner_role}</td>
                  <td className="p-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      item.priority === "high" ? "bg-red-100 text-red-700" :
                      item.priority === "medium" ? "bg-yellow-100 text-yellow-700" :
                      "bg-green-100 text-green-700"
                    }`}>{item.priority}</span>
                  </td>
                  <td className="p-2">{item.due_at || "—"}</td>
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
