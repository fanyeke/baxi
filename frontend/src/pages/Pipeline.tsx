import { useState } from "react"
import { useMutation } from "@tanstack/react-query"
import { apiClient } from "../api/client"
import type { PipelineRunResponse } from "../api/types"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

const PIPELINE_TYPES = [
  { type: "daily", label: "Daily Pipeline", desc: "8-step daily simulation" },
  { type: "full", label: "Full Pipeline", desc: "5-step full mode (634 days)" },
  { type: "db_full", label: "DB Full Pipeline", desc: "5-step DB mode with --dimensional" },
]

export default function Pipeline() {
  const [pipelineType, setPipelineType] = useState("daily")

  const mutation = useMutation({
    mutationFn: () => apiClient.post<PipelineRunResponse>("/pipeline/run", { pipeline_type: pipelineType }),
  })

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold">运行管道</h1>
      <p className="text-sm text-muted-foreground">选择管道类型查看命令预览。</p>

      <div className="space-y-2">
        {PIPELINE_TYPES.map(pt => (
          <label key={pt.type} className="flex items-center gap-3 p-3 border rounded-lg cursor-pointer hover:bg-muted/30">
            <input
              type="radio"
              name="pipeline"
              value={pt.type}
              checked={pipelineType === pt.type}
              onChange={() => setPipelineType(pt.type)}
            />
            <div>
              <p className="font-medium">{pt.label}</p>
              <p className="text-xs text-muted-foreground">{pt.desc}</p>
            </div>
          </label>
        ))}
      </div>

      <button
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        className="px-4 py-2 bg-primary text-primary-foreground rounded text-sm disabled:opacity-50"
      >
        查看预览
      </button>

      {mutation.isPending && <LoadingSkeleton type="text" count={3} />}

      {mutation.error && (
        <ErrorPanel title="预览加载失败" message="请确认 API 服务正常运行。" />
      )}

      {mutation.data && (
        <div className="space-y-3 p-4 border rounded-lg bg-muted/20">
          <div>
            <p className="text-sm font-medium">命令</p>
            <pre className="mt-1 p-2 bg-black/5 rounded text-sm font-mono overflow-x-auto">{mutation.data.command}</pre>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-xs text-muted-foreground">预计耗时</p>
              <p className="text-sm">{mutation.data.estimated_duration}</p>
            </div>
            <div>
              <p className="text-xs text-muted-foreground">管道类型</p>
              <p className="text-sm">{mutation.data.pipeline_type}</p>
            </div>
          </div>
          {mutation.data.required_env_vars.length > 0 && (
            <div>
              <p className="text-xs text-muted-foreground">所需环境变量</p>
              <div className="flex flex-wrap gap-1 mt-1">
                {mutation.data.required_env_vars.map(v => (
                  <span key={v} className="px-2 py-0.5 bg-muted rounded text-xs font-mono">{v}</span>
                ))}
              </div>
            </div>
          )}
          {mutation.data.warnings.length > 0 && (
            <div className="p-2 border border-yellow-300 bg-yellow-50 rounded">
              <p className="text-xs font-medium text-yellow-800">警告</p>
              {mutation.data.warnings.map((w, i) => (
                <p key={i} className="text-xs text-yellow-700 mt-1">{w}</p>
              ))}
            </div>
          )}
          <p className="text-xs text-muted-foreground mt-2">
            管道执行需在服务器终端手动运行，此处仅提供命令预览。
          </p>
        </div>
      )}
    </div>
  )
}
