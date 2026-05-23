import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import Tasks from "../Tasks"
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

const mockTasksData = {
  items: [
    {
      task_id: "task-001",
      task_title: "Fix bug #123",
      owner_role: "backend-dev",
      priority: "high",
      due_at: "2026-05-25",
      status: "in_progress",
    },
  ],
  total: 1,
}

describe("Tasks", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders task table with items", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockTasksData)

    renderWithQueryClient(<Tasks />)

    expect(screen.getByText("任务中心")).toBeInTheDocument()
    expect(await screen.findByText("Fix bug #123")).toBeInTheDocument()
  })

  it("shows empty state when no tasks", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [], total: 0 })

    renderWithQueryClient(<Tasks />)

    expect(screen.getByText("任务中心")).toBeInTheDocument()
    expect(await screen.findByText("暂无任务")).toBeInTheDocument()
  })

  it("shows error panel when query fails", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))

    renderWithQueryClient(<Tasks />)

    expect(await screen.findByText("加载失败")).toBeInTheDocument()
    expect(screen.getByText("Network error")).toBeInTheDocument()
  })
})
