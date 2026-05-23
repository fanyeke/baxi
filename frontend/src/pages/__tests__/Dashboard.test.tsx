import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import Dashboard from "../Dashboard"
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

describe("Dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders stat cards with data", async () => {
    vi.mocked(apiClient.get).mockImplementation(async (path: string) => {
      if (path === "/health") return { status: "ok", version: "1.0.0", db_connected: true }
      if (path === "/status") return { database: {}, last_pipeline_run: { status: "success" }, version: "1.0.0" }
      if (path === "/alerts?limit=1") return { items: [], total: 42 }
      if (path === "/tasks?limit=1") return { items: [], total: 15 }
      if (path === "/outbox?status=pending&limit=1") return { items: [], total: 3 }
      return {}
    })

    renderWithQueryClient(<Dashboard />)

    expect(screen.getByText("系统总览")).toBeInTheDocument()
    expect(await screen.findByText("数据库状态")).toBeInTheDocument()
    expect(await screen.findByText("OK")).toBeInTheDocument()
    expect(screen.getByText("42")).toBeInTheDocument()
    expect(screen.getByText("15")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("1.0.0")).toBeInTheDocument()
    expect(screen.getByText("success")).toBeInTheDocument()
  })

  it("shows loading skeleton when queries are loading", async () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))

    renderWithQueryClient(<Dashboard />)

    expect(screen.getByText("系统总览")).toBeInTheDocument()
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("shows error panel when queries fail", async () => {
    let callCount = 0
    vi.mocked(apiClient.get).mockImplementation(async () => {
      callCount++
      if (callCount === 1) throw new Error("Network error")
      return {}
    })

    renderWithQueryClient(<Dashboard />)

    expect(await screen.findByText("连接失败")).toBeInTheDocument()
  })
})
