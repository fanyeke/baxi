import { useParams } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { DecisionCaseResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

export default function CaseDetail() {
  const { id } = useParams<{ id: string }>()

  const { data, isLoading, error } = useQuery({
    queryKey: ["decision-case", id],
    queryFn: () => apiClient.get<DecisionCaseResponse>(`/decisions/cases/${id}`),
    enabled: !!id,
  })

  if (isLoading) return <LoadingSkeleton type="cards" count={3} />
  if (error) return <ErrorPanel title="加载失败" message={error.message || "Failed to load case details"} />
  if (!data) return <EmptyState title="未找到案件" />

  const policyResults = data.policy_results

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">案件详情</h1>

      <section className="border rounded-lg p-4 space-y-4">
        <h2 className="text-lg font-semibold border-b pb-2">基本信息</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="text-xs text-muted-foreground block">案件 ID</label>
            <span className="font-mono text-sm">{data.decision_case_id}</span>
          </div>
          <div>
            <label className="text-xs text-muted-foreground block">状态</label>
            <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
              data.status === "completed" ? "bg-green-100 text-green-700" :
              data.status === "pending" ? "bg-yellow-100 text-yellow-700" :
              "bg-gray-100 text-gray-700"
            }`}>
              {data.status}
            </span>
          </div>
          <div>
            <label className="text-xs text-muted-foreground block">对象类型</label>
            <span className="text-sm">{data.object_type}</span>
          </div>
          <div>
            <label className="text-xs text-muted-foreground block">对象 ID</label>
            <span className="font-mono text-sm">{data.object_id}</span>
          </div>
          {data.source_type && (
            <div>
              <label className="text-xs text-muted-foreground block">来源类型</label>
              <span className="text-sm">{data.source_type}</span>
            </div>
          )}
          {data.source_id && (
            <div>
              <label className="text-xs text-muted-foreground block">来源 ID</label>
              <span className="font-mono text-sm">{data.source_id}</span>
            </div>
          )}
          <div>
            <label className="text-xs text-muted-foreground block">严重程度</label>
            <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
              data.severity === "critical" ? "bg-red-100 text-red-700" :
              data.severity === "high" ? "bg-orange-100 text-orange-700" :
              data.severity === "medium" ? "bg-yellow-100 text-yellow-700" :
              "bg-blue-100 text-blue-700"
            }`}>
              {data.severity}
            </span>
          </div>
          <div>
            <label className="text-xs text-muted-foreground block">Context Hash</label>
            <span className="font-mono text-xs text-muted-foreground">{data.context_hash}</span>
          </div>
          <div>
            <label className="text-xs text-muted-foreground block">创建时间</label>
            <span className="text-sm">{new Date(data.created_at).toLocaleString()}</span>
          </div>
          {data.updated_at && (
            <div>
              <label className="text-xs text-muted-foreground block">更新时间</label>
              <span className="text-sm">{new Date(data.updated_at).toLocaleString()}</span>
            </div>
          )}
        </div>
      </section>

      {policyResults && (
        <section className="border rounded-lg p-4 space-y-4">
          <h2 className="text-lg font-semibold border-b pb-2">策略执行结果</h2>

          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">人工审批:</span>
            <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
              policyResults.human_approval_required
                ? "bg-orange-100 text-orange-700"
                : "bg-green-100 text-green-700"
            }`}>
              {policyResults.human_approval_required ? "需要" : "不需要"}
            </span>
          </div>

          {policyResults.allowed_actions.length > 0 && (
            <div>
              <h3 className="text-sm font-medium mb-2">允许的操作</h3>
              <div className="flex flex-wrap gap-2">
                {policyResults.allowed_actions.map((action) => (
                  <span key={action} className="px-2 py-1 bg-green-50 text-green-700 rounded text-xs border border-green-200">
                    {action}
                  </span>
                ))}
              </div>
            </div>
          )}

          {Object.entries(policyResults.blocked_actions).length > 0 && (
            <div>
              <h3 className="text-sm font-medium mb-2">阻止的操作</h3>
              <div className="space-y-2">
                {Object.entries(policyResults.blocked_actions).map(([action, reason]) => (
                  <div key={action} className="flex items-start gap-2 p-2 bg-red-50 rounded border border-red-200">
                    <span className="px-2 py-0.5 bg-red-100 text-red-700 rounded text-xs font-medium">
                      {action}
                    </span>
                    <span className="text-xs text-red-600 flex-1">{reason}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {Object.entries(policyResults.risk_levels).length > 0 && (
            <div>
              <h3 className="text-sm font-medium mb-2">风险等级</h3>
              <div className="grid grid-cols-2 gap-2">
                {Object.entries(policyResults.risk_levels).map(([key, level]) => (
                  <div key={key} className="flex items-center justify-between p-2 bg-muted rounded">
                    <span className="text-xs text-muted-foreground">{key}</span>
                    <span className={`px-2 py-0.5 rounded text-xs font-medium ${
                      level === "critical" ? "bg-red-100 text-red-700" :
                      level === "high" ? "bg-orange-100 text-orange-700" :
                      level === "medium" ? "bg-yellow-100 text-yellow-700" :
                      "bg-blue-100 text-blue-700"
                    }`}>
                      {level}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {policyResults.requires_approval_actions.length > 0 && (
            <div>
              <h3 className="text-sm font-medium mb-2">需审批的操作</h3>
              <div className="flex flex-wrap gap-2">
                {policyResults.requires_approval_actions.map((action) => (
                  <span key={action} className="px-2 py-1 bg-orange-50 text-orange-700 rounded text-xs border border-orange-200">
                    {action}
                  </span>
                ))}
              </div>
            </div>
          )}
        </section>
      )}

      {policyResults?.evidence_sources && policyResults.evidence_sources.length > 0 && (
        <section className="border rounded-lg p-4 space-y-4">
          <h2 className="text-lg font-semibold border-b pb-2">证据来源</h2>
          <ul className="space-y-1">
            {policyResults.evidence_sources.map((source) => (
              <li key={source} className="flex items-center gap-2 text-sm">
                <span className="text-muted-foreground">•</span>
                {source}
              </li>
            ))}
          </ul>
        </section>
      )}

      {data.status === "action_executed" && (
        <section className="border rounded-lg p-4 space-y-4">
          <h2 className="text-lg font-semibold border-b pb-2">执行状态</h2>
          <p className="text-sm text-muted-foreground">该案件的操作已执行，可查看 Outbox 分发详情。</p>
          <a
            href="/outbox"
            className="inline-flex items-center gap-2 px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90 transition-colors"
          >
            查看 Outbox 分发
          </a>
        </section>
      )}
    </div>
  )
}
