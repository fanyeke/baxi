import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import Governance from "../Governance"

vi.mock("../../api/governance", () => ({
  useCatalog: vi.fn(),
  useClassification: vi.fn(),
  useMarkings: vi.fn(),
  useLineage: vi.fn(),
  useCheckpoints: vi.fn(),
  useHealth: vi.fn(),
}))

import {
  useCatalog,
  useClassification,
  useMarkings,
  useLineage,
  useCheckpoints,
  useHealth,
} from "../../api/governance"

function renderWithQueryClient(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

const loadingState = { data: undefined, isLoading: true, error: null }
const errorState = { data: undefined, isLoading: false, error: new Error("API Error") }
const emptyState = { data: undefined, isLoading: false, error: null }

const mockCatalogData = {
  data_catalog: {},
  assets: [
    {
      asset_id: "asst-001",
      asset_type: "table",
      name: "订单表",
      location: "dw.orders",
      description: "电商订单主表",
      grain: "order",
      status: "active",
    },
  ],
}

const mockClassData = {
  classifications: [
    { asset_ref: "dw.orders", level: "confidential", rationale: "Contains PII" },
  ],
}

const mockMarkingData = {
  markings: {
    "mark-1": {
      mandatory_control: true,
      access_type: "role-based",
      conjunctive: false,
      inheritance: ["table", "column"],
      applies_to: ["dw.orders"],
      policy: "arn:aws:iam::policy/governance-mandatory",
    },
  },
  pipeline_stage_markings: [],
}

const mockLineageData = {
  nodes: [{ id: "n1", type: "table", label: "orders_raw", status: "active" }],
  edges: [{ from: "n1", to: "n2", transform: "SELECT * FROM", transform_type: "sql" }],
}

const mockCheckpointData = {
  checkpoints: {
    "chk-1": {
      scope: "pii-scan",
      endpoint: "/api/v1/scan",
      requires_justification: true,
      prompt: "Why access PII?",
      checkpoint_types: ["pre-query", "post-query"],
    },
  },
}

const mockHealthData = {
  monitoring_views: [
    { id: "mv-1", scope: "global", check_type: "drift", rule: "drift > 5%", severity: "high" },
  ],
  health_checks: [
    {
      id: "hc-1",
      resource: "catalog-sync",
      description: "每日目录同步检查",
      check_type: "freshness",
      severity: "medium",
      validation: "lag < 24h",
    },
  ],
}

describe("Governance", () => {
  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(useCatalog).mockReturnValue(loadingState as never)
    vi.mocked(useClassification).mockReturnValue(loadingState as never)
    vi.mocked(useMarkings).mockReturnValue(loadingState as never)
    vi.mocked(useLineage).mockReturnValue(loadingState as never)
    vi.mocked(useCheckpoints).mockReturnValue(loadingState as never)
    vi.mocked(useHealth).mockReturnValue(loadingState as never)
  })

  it("renders title and all 5 tab triggers", () => {
    renderWithQueryClient(<Governance />)

    expect(screen.getByText("治理中心")).toBeInTheDocument()
    expect(screen.getAllByText("数据目录").length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("分类与标记")).toBeInTheDocument()
    expect(screen.getByText("血缘关系")).toBeInTheDocument()
    expect(screen.getAllByText("检查点").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("健康检查").length).toBeGreaterThanOrEqual(1)
  })

  it("shows loading state on all tabs while data is loading", () => {
    renderWithQueryClient(<Governance />)

    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("handles error state gracefully", () => {
    vi.mocked(useCatalog).mockReturnValue(errorState as never)
    vi.mocked(useClassification).mockReturnValue(errorState as never)
    vi.mocked(useMarkings).mockReturnValue(errorState as never)
    vi.mocked(useLineage).mockReturnValue(errorState as never)
    vi.mocked(useCheckpoints).mockReturnValue(errorState as never)
    vi.mocked(useHealth).mockReturnValue(errorState as never)

    renderWithQueryClient(<Governance />)

    expect(screen.getByText("加载失败")).toBeInTheDocument()
  })

  it("displays catalog data after loading", () => {
    vi.mocked(useCatalog).mockReturnValue({
      data: mockCatalogData,
      isLoading: false,
      error: null,
    } as never)

    renderWithQueryClient(<Governance />)

    expect(screen.getByText("asst-001")).toBeInTheDocument()
    expect(screen.getByText("订单表")).toBeInTheDocument()
    expect(screen.getByText("dw.orders")).toBeInTheDocument()
    expect(screen.getByText("active")).toBeInTheDocument()
  })

  it("displays data across all tabs", () => {
    vi.mocked(useCatalog).mockReturnValue({
      data: mockCatalogData, isLoading: false, error: null,
    } as never)
    vi.mocked(useClassification).mockReturnValue({
      data: mockClassData, isLoading: false, error: null,
    } as never)
    vi.mocked(useMarkings).mockReturnValue({
      data: mockMarkingData, isLoading: false, error: null,
    } as never)
    vi.mocked(useLineage).mockReturnValue({
      data: mockLineageData, isLoading: false, error: null,
    } as never)
    vi.mocked(useCheckpoints).mockReturnValue({
      data: mockCheckpointData, isLoading: false, error: null,
    } as never)
    vi.mocked(useHealth).mockReturnValue({
      data: mockHealthData, isLoading: false, error: null,
    } as never)

    renderWithQueryClient(<Governance />)

    expect(screen.getByText("订单表")).toBeInTheDocument()
    const ones = screen.getAllByText("1")
    expect(ones.length).toBeGreaterThanOrEqual(1)
  })

  it("shows empty state when no data available", () => {
    vi.mocked(useCatalog).mockReturnValue({
      data: { assets: [], data_catalog: {} }, isLoading: false, error: null,
    } as never)

    renderWithQueryClient(<Governance />)

    expect(screen.getByText("暂无数据目录")).toBeInTheDocument()
  })
})
