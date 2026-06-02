import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithQueryClient } from "@/test-setup"
import PolicyInspector from "../PolicyInspector"
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
  useParams: () => ({ id: "policy-42" }),
}))

const mockStatus = {
  overall_health: "healthy",
  version: "v2.1.0",
  configs: {
    "access_control.yml": "loaded",
    "data_classification.yml": "loaded",
    "audit_policy.yml": "error",
  },
}

describe("PolicyInspector", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<PolicyInspector />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("renders error panel on failure", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<PolicyInspector />)
    expect(await screen.findByText("请求异常")).toBeInTheDocument()
  })

  it("renders empty state when no data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(null as any)
    renderWithQueryClient(<PolicyInspector />)
    expect(await screen.findByText("暂无治理状态数据")).toBeInTheDocument()
  })

  it("renders governance status with data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockStatus)
    renderWithQueryClient(<PolicyInspector />)
    expect(await screen.findByText("策略检查器")).toBeInTheDocument()
    expect(screen.getByText("Policy ID: policy-42")).toBeInTheDocument()
    expect(screen.getByText("healthy")).toBeInTheDocument()
    expect(screen.getByText("v2.1.0")).toBeInTheDocument()
    expect(screen.getByText("access_control.yml: loaded")).toBeInTheDocument()
    expect(screen.getByText("data_classification.yml: loaded")).toBeInTheDocument()
    expect(screen.getByText("audit_policy.yml: error")).toBeInTheDocument()
  })
})
