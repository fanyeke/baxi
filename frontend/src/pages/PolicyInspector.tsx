import { useParams } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { GovernanceStatusResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

export default function PolicyInspector() {
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, error } = useQuery({
    queryKey: ["governance-status", id],
    queryFn: () => apiClient.get<GovernanceStatusResponse>("/governance/status"),
  })

  if (isLoading) return <LoadingSkeleton type="cards" count={3} />
  if (error) return <ErrorPanel title="加载失败" message={error.message || "Failed to load governance status"} />
  if (!data) return <EmptyState title="暂无治理状态数据" />

  const configEntries = Object.entries(data.configs)

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">策略检查器</h1>
        <span className="text-sm text-muted-foreground font-mono">Policy ID: {id}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-card border rounded-lg p-4">
          <h2 className="text-sm font-medium text-muted-foreground mb-2">整体健康度</h2>
          <div className="flex items-center gap-2">
            <span
              className={`inline-flex items-center px-2 py-1 rounded text-xs font-medium ${
                data.overall_health === "healthy"
                  ? "bg-green-50 text-green-700"
                  : data.overall_health === "degraded"
                    ? "bg-yellow-50 text-yellow-700"
                    : "bg-red-50 text-red-700"
              }`}
            >
              {data.overall_health}
            </span>
          </div>
        </div>
        <div className="bg-card border rounded-lg p-4">
          <h2 className="text-sm font-medium text-muted-foreground mb-2">治理版本</h2>
          <span className="font-mono text-sm">{data.version}</span>
        </div>
      </div>

      <div className="bg-card border rounded-lg p-4">
        <h2 className="text-sm font-medium text-muted-foreground mb-3">配置版本</h2>
        {configEntries.length === 0 ? (
          <EmptyState title="暂无配置数据" />
        ) : (
          <div className="flex flex-wrap gap-2">
            {configEntries.map(([name, status]) => (
              <span
                key={name}
                className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${
                  status === "loaded"
                    ? "bg-green-50 text-green-700 border border-green-200"
                    : status === "error"
                      ? "bg-red-50 text-red-700 border border-red-200"
                      : "bg-muted text-muted-foreground border"
                }`}
              >
                {name}: {status}
              </span>
            ))}
          </div>
        )}
      </div>

      <div className="bg-card border rounded-lg p-4">
        <h2 className="text-sm font-medium text-muted-foreground mb-3">操作白名单</h2>
        <div className="border rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-muted">
              <tr>
                <th className="p-2 text-left">配置项</th>
                <th className="p-2 text-left">状态</th>
              </tr>
            </thead>
            <tbody>
              {configEntries.length === 0 ? (
                <tr>
                  <td colSpan={2} className="p-4 text-center text-muted-foreground">
                    暂无操作白名单数据
                  </td>
                </tr>
              ) : (
                configEntries.map(([name, status], i) => (
                  <tr key={i} className="border-t hover:bg-muted/50">
                    <td className="p-2 font-mono text-xs">{name}</td>
                    <td className="p-2">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                          status === "loaded"
                            ? "bg-green-50 text-green-700"
                            : status === "error"
                              ? "bg-red-50 text-red-700"
                              : "bg-muted text-muted-foreground"
                        }`}
                      >
                        {status}
                      </span>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
