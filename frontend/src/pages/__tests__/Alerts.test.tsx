import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"
import Alerts from "../Alerts"
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

const mockAlertsData = {
  items: [
    {
      event_id: "evt-001",
      rule_id: "rule-gmv-drop",
      event_date: "2026-05-20",
      severity: "high",
      metric_name: "GMV",
      object_type: "category",
      object_id: "cat-123",
      current_value: 1000,
      baseline_value: 5000,
      change_rate: -80,
      owner_role: "data-analyst",
      status: "new",
      impact_score: 9.5,
    },
    {
      event_id: "evt-002",
      rule_id: "rule-churn-spike",
      event_date: "2026-05-21",
      severity: "medium",
      metric_name: "churn_rate",
      object_type: "cohort",
      object_id: "coh-456",
      current_value: 12,
      baseline_value: 5,
      change_rate: 140,
      owner_role: "product-manager",
      status: "acknowledged",
      impact_score: 7.0,
    },
  ],
  total: 2,
}

describe("Alerts", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders alert table with items", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockAlertsData)

    renderWithQueryClient(<Alerts />)

    expect(screen.getByText("告警中心")).toBeInTheDocument()

    // Wait for table to render
    expect(await screen.findByText("rule-gmv-drop")).toBeInTheDocument()
    expect(screen.getByText("rule-churn-spike")).toBeInTheDocument()
    expect(screen.getByText("high")).toBeInTheDocument()
    expect(screen.getByText("medium")).toBeInTheDocument()
    expect(screen.getByText("category/cat-123")).toBeInTheDocument()
  })

  it("filter by severity dropdown changes query params", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockAlertsData)

    renderWithQueryClient(<Alerts />)

    // Verify initial call (no filters)
    expect(await screen.findByText("rule-gmv-drop")).toBeInTheDocument()

    // Count initial apiClient.get calls
    const initialCalls = vi.mocked(apiClient.get).mock.calls.length
    expect(initialCalls).toBeGreaterThanOrEqual(1)

    // Change severity filter
    const user = userEvent.setup()
    const severitySelect = screen.getAllByRole("combobox")[0]
    await user.selectOptions(severitySelect, "high")

    // After selecting "high", the query should be refetched with new params
    const callsAfter = vi.mocked(apiClient.get).mock.calls.length
    expect(callsAfter).toBeGreaterThan(initialCalls)
  })

  it("shows empty state when no alerts", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [], total: 0 })

    renderWithQueryClient(<Alerts />)

    expect(screen.getByText("告警中心")).toBeInTheDocument()
    expect(await screen.findByText("暂无告警")).toBeInTheDocument()
  })
})
