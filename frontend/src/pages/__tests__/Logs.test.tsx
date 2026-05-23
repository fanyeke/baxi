import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import Logs from "../Logs"
import { apiClient } from "@/api/client"

vi.mock("@/api/client", () => ({
  apiClient: {
    get: vi.fn(),
  },
}))

function renderWithQueryClient(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

const mockErrorsData = {
  items: [
    {
      ts: "2026-05-20T10:00:00Z",
      error_code: "ERR_500",
      message: "Internal server error",
      diagnosis: "Check server logs",
      request_id: "req-1234567890abcdef",
    },
  ],
  total: 1,
}

describe("Logs", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders log page with tabs", () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockErrorsData)
    renderWithQueryClient(<Logs />)

    expect(screen.getByText("日志诊断")).toBeInTheDocument()
    expect(screen.getByText("错误日志")).toBeInTheDocument()
    expect(screen.getByText("审计日志")).toBeInTheDocument()
    expect(screen.getByText("最近请求")).toBeInTheDocument()
  })

  it("renders errors tab with data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockErrorsData)
    renderWithQueryClient(<Logs />)

    expect(await screen.findByText("ERR_500")).toBeInTheDocument()
  })

  it("shows error panel when errors tab query fails", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Failed to fetch error logs"))
    renderWithQueryClient(<Logs />)

    expect(await screen.findByText("加载失败")).toBeInTheDocument()
    expect(screen.getByText("Failed to fetch error logs")).toBeInTheDocument()
  })

  it("shows empty state when errors tab has no data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [], total: 0 })
    renderWithQueryClient(<Logs />)

    expect(await screen.findByText("暂无错误日志")).toBeInTheDocument()
  })
})
