import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { AgentLogListResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

export default function AgentLogs() {
  const [tool, setTool] = useState("")
  const [status, setStatus] = useState("")
  const params: Record<string, string> = { limit: "100" }
  if (tool) params.tool = tool
  if (status) params.status = status

  const { data, isLoading, error } = useQuery({
    queryKey: ["agentLogs", tool, status],
    queryFn: () => apiClient.get<AgentLogListResponse>("/agent-execution-logs", params),
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">Agent 执行日志</h1>
      <div className="flex gap-2">
        <select className="px-3 py-1 border rounded text-sm" value={tool} onChange={e => setTool(e.target.value)}>
          <option value="">全部工具</option>
          <option value="create_decision_case">创建决策案例</option>
          <option value="generate_decision">生成决策</option>
          <option value="execute_action">执行动作</option>
        </select>
        <select className="px-3 py-1 border rounded text-sm" value={status} onChange={e => setStatus(e.target.value)}>
          <option value="">全部状态</option>
          <option value="success">成功</option>
          <option value="failed">失败</option>
        </select>
      </div>

      {isLoading && <LoadingSkeleton type="table" count={5} />}
      {error && <ErrorPanel title="加载失败" message={String(error)} />}
      {data && data.items.length === 0 && <EmptyState title="暂无执行日志" />}
      {data && data.items.length > 0 && (
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="p-2 text-left">时间</th>
                <th className="p-2 text-left">工具</th>
                <th className="p-2 text-left">状态</th>
                <th className="p-2 text-left">耗时</th>
                <th className="p-2 text-left">会话ID</th>
              </tr>
            </thead>
            <tbody>
              {data.items.map(item => (
                <tr key={item.execution_id} className="border-t hover:bg-muted/50">
                  <td className="p-2">{item.created_at}</td>
                  <td className="p-2 font-mono text-xs">{item.tool_name}</td>
                  <td className="p-2">
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      item.status === "success" ? "bg-green-100 text-green-700" :
                      item.status === "failed" ? "bg-red-100 text-red-700" :
                      "bg-gray-100 text-gray-700"
                    }`}>{item.status}</span>
                  </td>
                  <td className="p-2">{item.duration_ms != null ? `${item.duration_ms}ms` : "-"}</td>
                  <td className="p-2 font-mono text-xs">{item.session_id ?? "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
