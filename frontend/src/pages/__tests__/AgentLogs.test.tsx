import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, fireEvent } from "@testing-library/react"
import { renderWithQueryClient } from "@/test-setup"
import AgentLogs from "../AgentLogs"
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

const mockLogs = {
  items: [
    { execution_id: "e1", tool_name: "create_decision_case", status: "success", duration_ms: 150, session_id: "s1", created_at: "2026-05-28T10:00:00Z" },
    { execution_id: "e2", tool_name: "execute_action", status: "failed", duration_ms: 500, session_id: "s2", created_at: "2026-05-28T10:01:00Z" },
  ],
}

describe("AgentLogs", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders title and filters", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<AgentLogs />)
    expect(screen.getByText("Agent 执行日志")).toBeInTheDocument()
    expect(screen.getByText("全部工具")).toBeInTheDocument()
    expect(screen.getByText("全部状态")).toBeInTheDocument()
  })

  it("shows loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<AgentLogs />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("shows error panel", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<AgentLogs />)
    expect(await screen.findByText("请求异常")).toBeInTheDocument()
  })

  it("shows empty state when no logs", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [] })
    renderWithQueryClient(<AgentLogs />)
    expect(await screen.findByText("暂无执行日志")).toBeInTheDocument()
  })

  it("renders log table with data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockLogs)
    renderWithQueryClient(<AgentLogs />)
    expect(await screen.findByText("create_decision_case")).toBeInTheDocument()
    expect(screen.getByText("execute_action")).toBeInTheDocument()
    expect(screen.getByText("success")).toBeInTheDocument()
    expect(screen.getByText("failed")).toBeInTheDocument()
    expect(screen.getByText("150ms")).toBeInTheDocument()
    expect(screen.getByText("500ms")).toBeInTheDocument()
  })

  it("filters by tool selection", async () => {
    const getMock = vi.mocked(apiClient.get).mockResolvedValue(mockLogs)
    renderWithQueryClient(<AgentLogs />)
    await screen.findByText("create_decision_case")

    const select = screen.getByDisplayValue("全部工具") as HTMLSelectElement
    fireEvent.change(select, { target: { value: "create_decision_case" } })
    expect(getMock).toHaveBeenCalled()
  })
})
