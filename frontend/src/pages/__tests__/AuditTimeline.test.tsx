import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithQueryClient } from "@/test-setup"
import AuditTimeline from "../AuditTimeline"
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

const mockEvents = {
  items: [
    {
      outbox_id: "ob_abc123def456",
      source: "pipeline",
      target_channel: "feishu",
      adapter_name: "FeishuAdapter",
      mode: "batch",
      status: "completed",
      timestamp: "2026-05-28T10:00:00Z",
      error: null,
      external_ref: "ref-001",
    },
    {
      outbox_id: "ob_def789ghi012",
      source: "alert",
      target_channel: "email",
      adapter_name: "SMTPAdapter",
      mode: null,
      status: "pending",
      timestamp: "2026-05-28T11:00:00Z",
      error: "connection refused",
      external_ref: null,
    },
  ],
}

describe("AuditTimeline", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<AuditTimeline />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("renders error panel on failure", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<AuditTimeline />)
    expect(await screen.findByText("请求异常")).toBeInTheDocument()
  })

  it("renders empty state when no events", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [] })
    renderWithQueryClient(<AuditTimeline />)
    expect(await screen.findByText("暂无审计日志")).toBeInTheDocument()
  })

  it("renders timeline with grouped events", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockEvents)
    renderWithQueryClient(<AuditTimeline />)
    expect(await screen.findByText("审计时间线")).toBeInTheDocument()
    expect(screen.getByText("Case: dc_123")).toBeInTheDocument()
    expect(screen.getByText("feishu")).toBeInTheDocument()
    expect(screen.getByText("FeishuAdapter")).toBeInTheDocument()
    expect(screen.getByText("email")).toBeInTheDocument()
    expect(screen.getByText("SMTPAdapter")).toBeInTheDocument()
    expect(screen.getByText("batch")).toBeInTheDocument()
    expect(screen.getByText("connection refused")).toBeInTheDocument()
  })
})
