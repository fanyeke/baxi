import * as Tabs from "@radix-ui/react-tabs"
import {
  useCatalog,
  useClassification,
  useMarkings,
  useLineage,
  useCheckpoints,
  useHealth,
} from "../api/governance"
import type {
  CatalogAsset,
  Classification,
  MarkingInfo,
  LineageEdge,
  CheckpointRule,
  HealthCheck,
  MonitoringView,
} from "../api/governance"
import { EmptyState, LoadingSkeleton, ErrorPanel } from "../components"

const TAB_ITEMS = [
  { value: "catalog", label: "数据目录", icon: "📋" },
  { value: "class", label: "分类与标记", icon: "🔐" },
  { value: "lineage", label: "血缘关系", icon: "🔗" },
  { value: "checkpoints", label: "检查点", icon: "✅" },
  { value: "health", label: "健康检查", icon: "💚" },
] as const

function fmtError(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function LevelBadge({ level }: { level: string }) {
  const colors: Record<string, string> = {
    public_internal: "bg-green-100 text-green-700",
    internal: "bg-blue-100 text-blue-700",
    sensitive: "bg-yellow-100 text-yellow-700",
    pii: "bg-red-100 text-red-700",
    derived_sensitive: "bg-orange-100 text-orange-700",
  }
  const cls = colors[level.toLowerCase()] ?? "bg-gray-100 text-gray-700"
  return <span className={`px-2 py-0.5 rounded text-xs font-medium ${cls}`}>{level}</span>
}

function SeverityBadge({ severity }: { severity: string }) {
  const colors: Record<string, string> = {
    high: "bg-red-100 text-red-700",
    medium: "bg-yellow-100 text-yellow-700",
    low: "bg-green-100 text-green-700",
    info: "bg-blue-100 text-blue-700",
  }
  const cls = colors[severity.toLowerCase()] ?? "bg-gray-100 text-gray-700"
  return <span className={`px-2 py-0.5 rounded text-xs font-medium ${cls}`}>{severity}</span>
}

function YesNoBadge({ value }: { value: boolean }) {
  return value
    ? <span className="text-green-600 font-medium">是</span>
    : <span className="text-muted-foreground">否</span>
}

function DataTable({ headers, children }: { headers: string[]; children: React.ReactNode }) {
  return (
    <div className="border rounded-lg overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-muted">
          <tr>
            {headers.map((h) => (
              <th key={h} className="p-2 text-left whitespace-nowrap">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

function CatalogTab({ data, isLoading, error }: ReturnType<typeof useCatalog>) {
  if (isLoading) return <LoadingSkeleton type="table" count={6} />
  if (error) return <ErrorPanel title="加载失败" message={fmtError(error)} />
  if (!data || data.assets.length === 0) return <EmptyState title="暂无数据目录" />

  return (
    <DataTable headers={["资产 ID", "名称", "类型", "位置", "描述", "粒度", "状态"]}>
      {data.assets.map((a: CatalogAsset) => (
        <tr key={a.asset_id} className="border-t hover:bg-muted/50">
          <td className="p-2 font-mono text-xs">{a.asset_id}</td>
          <td className="p-2 font-medium">{a.name}</td>
          <td className="p-2">{a.asset_type}</td>
          <td className="p-2 font-mono text-xs">{a.location}</td>
          <td className="p-2 text-muted-foreground max-w-[200px] truncate">{a.description ?? "—"}</td>
          <td className="p-2">{a.grain ?? "—"}</td>
          <td className="p-2"><SeverityBadge severity={a.status} /></td>
        </tr>
      ))}
    </DataTable>
  )
}

function ClassTab({
  classQuery,
  markingQuery,
}: {
  classQuery: ReturnType<typeof useClassification>
  markingQuery: ReturnType<typeof useMarkings>
}) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">分类规则</h3>
        {classQuery.isLoading && <LoadingSkeleton type="table" count={3} />}
        {classQuery.error && <ErrorPanel title="加载失败" message={fmtError(classQuery.error)} />}
        {classQuery.data && classQuery.data.classifications.length === 0 && <EmptyState title="暂无分类" />}
        {classQuery.data && classQuery.data.classifications.length > 0 && (
          <DataTable headers={["资产引用", "级别", "依据", "字段规则"]}>
            {classQuery.data.classifications.map((c: Classification, i: number) => (
              <tr key={c.asset_ref + i} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{c.asset_ref}</td>
                <td className="p-2"><LevelBadge level={c.level} /></td>
                <td className="p-2 text-muted-foreground max-w-[300px] truncate">{c.rationale}</td>
                <td className="p-2 text-xs font-mono">{c.applies_to_fields ? JSON.stringify(c.applies_to_fields) : "—"}</td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>

      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">标记策略</h3>
        {markingQuery.isLoading && <LoadingSkeleton type="table" count={3} />}
        {markingQuery.error && <ErrorPanel title="加载失败" message={fmtError(markingQuery.error)} />}
        {markingQuery.data && Object.keys(markingQuery.data.markings).length === 0 && <EmptyState title="暂无标记" />}
        {markingQuery.data && Object.keys(markingQuery.data.markings).length > 0 && (
          <DataTable headers={["标记键", "访问类型", "强制控制", "合取", "继承", "适用范围", "策略"]}>
            {Object.entries(markingQuery.data.markings).map(([key, m]: [string, MarkingInfo], i: number) => (
              <tr key={key + i} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{key}</td>
                <td className="p-2 font-medium">{m.access_type}</td>
                <td className="p-2"><YesNoBadge value={m.mandatory_control} /></td>
                <td className="p-2"><YesNoBadge value={m.conjunctive} /></td>
                <td className="p-2 text-xs max-w-[150px] truncate">{m.inheritance.join(", ") || "—"}</td>
                <td className="p-2 text-xs max-w-[150px] truncate">{m.applies_to.join(", ") || "—"}</td>
                <td className="p-2 font-mono text-xs max-w-[200px] truncate">{m.policy}</td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>
    </div>
  )
}

function LineageTab({ data, isLoading, error }: ReturnType<typeof useLineage>) {
  if (isLoading) return <LoadingSkeleton type="table" count={4} />
  if (error) return <ErrorPanel title="加载失败" message={fmtError(error)} />
  if (!data) return <EmptyState title="暂无血缘关系" />

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">节点 ({data.nodes.length})</h3>
        {data.nodes.length === 0 ? (
          <EmptyState title="暂无节点" />
        ) : (
          <DataTable headers={["ID", "类型", "标签", "状态"]}>
            {data.nodes.map((n) => (
              <tr key={n.id} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{n.id}</td>
                <td className="p-2">{n.type}</td>
                <td className="p-2 font-medium">{n.label}</td>
                <td className="p-2"><SeverityBadge severity={n.status} /></td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>

      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">边 ({data.edges.length})</h3>
        {data.edges.length === 0 ? (
          <EmptyState title="暂无边" />
        ) : (
          <DataTable headers={["来源", "目标", "转换", "转换类型"]}>
            {data.edges.map((e: LineageEdge, i: number) => (
              <tr key={i} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{e.from}</td>
                <td className="p-2 font-mono text-xs">{e.to}</td>
                <td className="p-2 text-xs max-w-[200px] truncate">{e.transform}</td>
                <td className="p-2">{e.transform_type}</td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>
    </div>
  )
}

function CheckpointsTab({ data, isLoading, error }: ReturnType<typeof useCheckpoints>) {
  if (isLoading) return <LoadingSkeleton type="table" count={5} />
  if (error) return <ErrorPanel title="加载失败" message={fmtError(error)} />
  if (!data || Object.keys(data.checkpoints).length === 0) return <EmptyState title="暂无检查点" />

  return (
    <DataTable headers={["范围", "端点", "需要理由", "提示词", "检查类型"]}>
      {Object.entries(data.checkpoints).map(([key, r]: [string, CheckpointRule], i: number) => (
        <tr key={key + i} className="border-t hover:bg-muted/50">
          <td className="p-2 font-medium">{r.scope}</td>
          <td className="p-2 font-mono text-xs">{r.endpoint ?? "—"}</td>
          <td className="p-2"><YesNoBadge value={r.requires_justification} /></td>
          <td className="p-2 text-xs text-muted-foreground max-w-[250px] truncate">{r.prompt ?? "—"}</td>
          <td className="p-2 text-xs">{r.checkpoint_types?.join(", ") ?? "—"}</td>
        </tr>
      ))}
    </DataTable>
  )
}

function HealthTab({ data, isLoading, error }: ReturnType<typeof useHealth>) {
  if (isLoading) return <LoadingSkeleton type="table" count={5} />
  if (error) return <ErrorPanel title="加载失败" message={fmtError(error)} />
  if (!data) return <EmptyState title="暂无健康检查" />

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">监控视图 ({data.monitoring_views.length})</h3>
        {data.monitoring_views.length === 0 ? (
          <EmptyState title="暂无监控视图" />
        ) : (
          <DataTable headers={["ID", "范围", "检查类型", "规则", "严重度"]}>
            {data.monitoring_views.map((v: MonitoringView) => (
              <tr key={v.id} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{v.id}</td>
                <td className="p-2">{v.scope}</td>
                <td className="p-2">{v.check_type}</td>
                <td className="p-2 text-xs max-w-[200px] truncate">{v.rule}</td>
                <td className="p-2"><SeverityBadge severity={v.severity} /></td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>

      <div>
        <h3 className="text-sm font-semibold text-muted-foreground mb-2">健康检查 ({data.health_checks.length})</h3>
        {data.health_checks.length === 0 ? (
          <EmptyState title="暂无健康检查项" />
        ) : (
          <DataTable headers={["ID", "资源", "描述", "检查类型", "严重度", "验证"]}>
            {data.health_checks.map((h: HealthCheck) => (
              <tr key={h.id} className="border-t hover:bg-muted/50">
                <td className="p-2 font-mono text-xs">{h.id}</td>
                <td className="p-2">{h.resource ?? "—"}</td>
                <td className="p-2 text-muted-foreground max-w-[200px] truncate">{h.description}</td>
                <td className="p-2">{h.check_type}</td>
                <td className="p-2"><SeverityBadge severity={h.severity} /></td>
                <td className="p-2 font-mono text-xs max-w-[150px] truncate">{h.validation ?? "—"}</td>
              </tr>
            ))}
          </DataTable>
        )}
      </div>
    </div>
  )
}

function SummaryStats(props: {
  catalogCount: number
  classCount: number
  checkpointCount: number
  healthCount: number
}) {
  const items = [
    { label: "数据资产", value: props.catalogCount, icon: "📦" },
    { label: "分类规则", value: props.classCount, icon: "🏷️" },
    { label: "检查点", value: props.checkpointCount, icon: "✅" },
    { label: "健康检查", value: props.healthCount, icon: "💚" },
  ]

  return (
    <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-8">
      {items.map((s) => (
        <div key={s.label} className="border rounded-lg p-4 bg-card">
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <span>{s.icon}</span>
            <span>{s.label}</span>
          </div>
          <p className="text-2xl font-bold mt-1">{s.value}</p>
        </div>
      ))}
    </div>
  )
}

export default function Governance() {
  const catalog = useCatalog()
  const classQuery = useClassification()
  const markingQuery = useMarkings()
  const lineage = useLineage()
  const checkpoints = useCheckpoints()
  const health = useHealth()

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold">治理中心</h1>
        <p className="text-sm text-muted-foreground mt-1">
          数据资产治理、分类标记、血缘追踪与健康监控
        </p>
      </div>

      <Tabs.Root defaultValue="catalog" className="space-y-4">
        <Tabs.List className="flex gap-1 border-b" aria-label="治理功能">
          {TAB_ITEMS.map((tab) => (
            <Tabs.Trigger
              key={tab.value}
              value={tab.value}
              className="px-4 py-2 text-sm font-medium text-muted-foreground
                         data-[state=active]:text-foreground data-[state=active]:border-b-2
                         data-[state=active]:border-primary hover:text-foreground
                         transition-colors cursor-pointer"
            >
              <span className="mr-1.5">{tab.icon}</span>
              {tab.label}
            </Tabs.Trigger>
          ))}
        </Tabs.List>

        <Tabs.Content value="catalog">
          <CatalogTab {...catalog} />
        </Tabs.Content>

        <Tabs.Content value="class">
          <ClassTab classQuery={classQuery} markingQuery={markingQuery} />
        </Tabs.Content>

        <Tabs.Content value="lineage">
          <LineageTab {...lineage} />
        </Tabs.Content>

        <Tabs.Content value="checkpoints">
          <CheckpointsTab {...checkpoints} />
        </Tabs.Content>

        <Tabs.Content value="health">
          <HealthTab {...health} />
        </Tabs.Content>
      </Tabs.Root>

      <SummaryStats
        catalogCount={catalog.data?.assets.length ?? 0}
        classCount={classQuery.data?.classifications.length ?? 0}
        checkpointCount={checkpoints.data ? Object.keys(checkpoints.data.checkpoints).length : 0}
        healthCount={health.data
          ? health.data.health_checks.length + health.data.monitoring_views.length
          : 0}
      />
    </div>
  )
}
