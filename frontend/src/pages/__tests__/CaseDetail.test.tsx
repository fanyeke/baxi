import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithQueryClient } from "@/test-setup"
import CaseDetail from "../CaseDetail"
import { apiClient } from "@/api/client"

vi.mock("@/api/client", () => ({
  apiClient: { get: vi.fn() },
  ApiClientError: class extends Error {
    status: number
    apiError: Record<string, unknown>
    constructor(status: number, apiError: Record<string, unknown>) {
      super(String(apiError.message))
      this.status = status
      this.apiError = apiError
    }
  },
}))

vi.mock("react-router-dom", () => ({
  useParams: () => ({ id: "dc_123" }),
}))

const mockCaseData = {
  decision_case_id: "dc_123",
  status: "completed",
  object_type: "seller",
  object_id: "seller-42",
  severity: "high",
  context_hash: "abc123",
  created_at: "2026-05-28T10:00:00Z",
  updated_at: "2026-05-28T12:00:00Z",
  source_type: "alert",
  source_id: "alert-1",
  policy_results: {
    human_approval_required: false,
    allowed_actions: ["notify_owner"],
    blocked_actions: { "delete_account": "requires higher clearance" },
    risk_levels: { "notify_owner": "low", "delete_account": "critical" },
    requires_approval_actions: ["escalate"],
    evidence_sources: ["GMV drop > 30% detected"],
  },
}

describe("CaseDetail", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<CaseDetail />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("renders error panel on failure", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<CaseDetail />)
    expect(await screen.findByText("请求异常")).toBeInTheDocument()
  })

  it("renders empty state when no data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(null as any)
    renderWithQueryClient(<CaseDetail />)
    expect(await screen.findByText("未找到案件")).toBeInTheDocument()
  })

  it("renders case details with data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockCaseData)
    renderWithQueryClient(<CaseDetail />)
    expect(await screen.findByText("案件详情")).toBeInTheDocument()
    expect(screen.getByText("dc_123")).toBeInTheDocument()
    expect(screen.getByText("completed")).toBeInTheDocument()
    expect(screen.getByText("seller")).toBeInTheDocument()
    expect(screen.getByText("seller-42")).toBeInTheDocument()
    expect(screen.getByText("high")).toBeInTheDocument()
    expect(screen.getByText("策略执行结果")).toBeInTheDocument()
    expect(screen.getByText("不需要")).toBeInTheDocument()
    expect(screen.getAllByText("notify_owner").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("delete_account").length).toBeGreaterThanOrEqual(1)
  })

  it("shows execution status when action_executed", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({
      ...mockCaseData,
      status: "action_executed",
    })
    renderWithQueryClient(<CaseDetail />)
    expect(await screen.findByText("执行状态")).toBeInTheDocument()
    expect(screen.getByText("查看 Outbox 分发")).toBeInTheDocument()
  })
})
