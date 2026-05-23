import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"
import Outbox from "../Outbox"
import { apiClient } from "@/api/client"

vi.mock("@/api/client", () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
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

const mockOutboxData = {
  items: [
    {
      outbox_id: "ob-001-aef1234567890",
      event_type: "alert_dispatch",
      source_type: "alert",
      source_id: "evt-001",
      target_channel: "feishu_cli",
      status: "pending",
      created_at: "2026-05-20T10:00:00Z",
      dispatch_attempts: 0,
      last_dispatch_at: null,
    },
    {
      outbox_id: "ob-002-fedcba0987654321",
      event_type: "task_notify",
      source_type: "task",
      source_id: "task-100",
      target_channel: "local_cli",
      status: "dispatched",
      created_at: "2026-05-20T09:00:00Z",
      dispatch_attempts: 1,
      last_dispatch_at: "2026-05-20T09:30:00Z",
    },
  ],
  total: 2,
}

const mockDispatchResponse = {
  request_id: "req-dispatch-001",
  dry_run: true,
  processed: 2,
  results: [
    { outbox_id: "ob-001-aef1234567890", status: "preview", adapter_name: "feishu_cli", message: null, external_ref: null, error: null },
  ],
}

describe("Outbox", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders outbox table with items", async () => {
    const { apiClient } = await import("../../api/client")
    vi.mocked(apiClient.get).mockResolvedValue(mockOutboxData)
    vi.mocked(apiClient.post).mockResolvedValue(mockDispatchResponse)

    renderWithQueryClient(<Outbox />)

    expect(screen.getByText("Outbox 分发")).toBeInTheDocument()

    // Wait for table to render
    expect(await screen.findByText("ob-001-a")).toBeInTheDocument()
    expect(screen.getByText("alert_dispatch")).toBeInTheDocument()
    expect(screen.getByText("feishu_cli")).toBeInTheDocument()
    expect(screen.getByText("pending")).toBeInTheDocument()
  })

  it("dry-run button triggers POST mutation", async () => {
    const { apiClient } = await import("../../api/client")
    vi.mocked(apiClient.get).mockResolvedValue(mockOutboxData)
    vi.mocked(apiClient.post).mockResolvedValue(mockDispatchResponse)

    renderWithQueryClient(<Outbox />)

    // Wait for table to render first
    expect(await screen.findByText("ob-001-a")).toBeInTheDocument()

    // Click dry-run button
    const dryRunBtn = screen.getByText("Dry-Run 分发")
    const user = userEvent.setup()
    await user.click(dryRunBtn)

    // Verify POST call was made with dry_run: true
    const postCalls = vi.mocked(apiClient.post).mock.calls
    expect(postCalls.length).toBeGreaterThan(0)
    expect(postCalls[0][0]).toBe("/outbox/dispatch")
    expect(postCalls[0][1]).toEqual({ dry_run: true, channel: undefined, limit: 100 })

    // Wait for results to display
    expect(await screen.findByText(/Dry-Run.*2 条/)).toBeInTheDocument()
  })

  it("confirm dialog appears for apply", async () => {
    const { apiClient } = await import("../../api/client")
    vi.mocked(apiClient.get).mockResolvedValue(mockOutboxData)
    vi.mocked(apiClient.post).mockResolvedValue(mockDispatchResponse)

    renderWithQueryClient(<Outbox />)

    // Wait for table to render
    expect(await screen.findByText("ob-001-a")).toBeInTheDocument()

    // Click "真实分发" button
    const applyBtn = screen.getByText("真实分发")
    const user = userEvent.setup()
    await user.click(applyBtn)

    // Confirm dialog should appear
    expect(screen.getByText("确认执行？")).toBeInTheDocument()
    expect(screen.getByText(/这将执行真实的分发操作，不可撤销/)).toBeInTheDocument()
    expect(screen.getByText("确认执行")).toBeInTheDocument()
    expect(screen.getByText("取消")).toBeInTheDocument()
  })
})
