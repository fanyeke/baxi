import { useState } from "react"
import { useMutation } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { FeishuExportResponse, FeishuSyncResponse, FeishuStatusImportResponse } from "../api/types"
import { ConfirmApplyDialog } from "../components/ConfirmApplyDialog"

export default function Feishu() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">飞书同步</h1>
      <ExportSection />
      <SyncSection />
      <ImportSection />
    </div>
  )
}

function StatusCard({ title, result }: { title: string; result: { status: string; message?: string } | null }) {
  if (!result) return null
  const isOk = result.status === "preview" || result.status === "not_configured" || result.status === "exported"
  return (
    <div className={`p-3 rounded-lg text-sm border ${isOk ? "bg-muted/30" : "bg-destructive/10 border-destructive/30"}`}>
      <span className="font-medium">{title}: </span>
      <span className={isOk ? "text-muted-foreground" : "text-destructive"}>{result.status}</span>
      {result.message && <p className="text-xs text-muted-foreground mt-1">{result.message}</p>}
    </div>
  )
}

function ExportSection() {
  const [showApply, setShowApply] = useState(false)
  const mutation = useMutation({
    mutationFn: (apply: boolean) => apiClient.post<FeishuExportResponse>("/feishu/export", { apply }),
  })

  return (
    <div className="space-y-2">
      <h2 className="text-lg font-semibold">导出</h2>
      <p className="text-xs text-muted-foreground">将本地数据导出为飞书格式 CSV</p>
      <div className="flex gap-2">
        <button onClick={() => mutation.mutate(false)} className="px-4 py-2 bg-primary text-primary-foreground rounded text-sm">Dry-Run 导出</button>
        <button onClick={() => setShowApply(true)} className="px-4 py-2 border rounded text-sm text-destructive">真实导出</button>
      </div>
      {showApply && (
        <ConfirmApplyDialog
          open={showApply}
          title="确认导出？"
          description="确认导出文件到 data/feishu/？"
          onConfirm={() => { mutation.mutate(true); setShowApply(false) }}
          onCancel={() => setShowApply(false)}
        />
      )}
      <StatusCard title="导出" result={mutation.data || (mutation.isError ? { status: "failed", message: String(mutation.error) } : null)} />
    </div>
  )
}

function SyncSection() {
  const [showApply, setShowApply] = useState(false)
  const mutation = useMutation({
    mutationFn: (apply: boolean) => apiClient.post<FeishuSyncResponse>("/feishu/sync", { apply }),
  })

  return (
    <div className="space-y-2">
      <h2 className="text-lg font-semibold">同步到飞书</h2>
      <p className="text-xs text-muted-foreground">将 CSV 数据同步到飞书多维表格</p>
      <div className="flex gap-2">
        <button onClick={() => mutation.mutate(false)} className="px-4 py-2 bg-primary text-primary-foreground rounded text-sm">Dry-Run 同步</button>
        <button onClick={() => setShowApply(true)} className="px-4 py-2 border rounded text-sm text-destructive">真实同步</button>
      </div>
      {showApply && (
        <ConfirmApplyDialog
          open={showApply}
          title="确认同步到飞书？"
          description="确认同步数据到飞书？这会写入飞书多维表格。"
          onConfirm={() => { mutation.mutate(true); setShowApply(false) }}
          onCancel={() => setShowApply(false)}
        />
      )}
      <StatusCard title="同步" result={mutation.data || (mutation.isError ? { status: "failed", message: String(mutation.error) } : null)} />
    </div>
  )
}

function ImportSection() {
  const mutation = useMutation({
    mutationFn: () => apiClient.post<FeishuStatusImportResponse>("/feishu/status/import", {}),
  })

  return (
    <div className="space-y-2">
      <h2 className="text-lg font-semibold">状态导入</h2>
      <p className="text-xs text-muted-foreground">从飞书拉取状态修改并同步到本地数据库</p>
      <button onClick={() => mutation.mutate()} className="px-4 py-2 bg-primary text-primary-foreground rounded text-sm">Dry-Run 导入</button>
      <StatusCard title="导入" result={mutation.data || (mutation.isError ? { status: "failed", message: String(mutation.error) } : null)} />
      <p className="text-xs text-muted-foreground mt-2">提示：状态修改建议通过飞书完成，此处仅供预览。</p>
    </div>
  )
}
