import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithQueryClient } from "@/test-setup"
import DecisionReview from "../DecisionReview"
import { apiClient } from "@/api/client"

vi.mock("@/api/client", () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
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

const mockCases = {
  cases: [
    { case_id: "dc_123", status: "pending", object_type: "seller", object_id: "s-42", severity: "high", created_at: "2026-05-28T10:00:00Z" },
  ],
}

const mockProposals = {
  proposals: [
    { proposal_id: "ap_1", case_id: "dc_123", decision_id: "dec_1", action_type: "notify_owner", title: "Notify owner", risk_level: "medium", apply_status: "proposed", requires_human_review: true, payload: {}, description: "Send notification", created_at: "2026-05-28T10:00:00Z" },
    { proposal_id: "ap_2", case_id: "dc_123", decision_id: "dec_2", action_type: "escalate", title: "Escalate", risk_level: "high", apply_status: "approved", requires_human_review: false, payload: { reason: "test" }, description: "Escalate to manager", created_at: "2026-05-28T09:00:00Z" },
  ],
}

const mockReviewRecord = {
  record_id: "review_1",
  proposal_id: "ap_1",
  verdict: "approved",
  reviewer_id: "admin",
  feedback: "Looks good",
  created_at: "2026-05-28T10:00:00Z",
}

function setupDefaultGet() {
  const getMock = vi.mocked(apiClient.get)
  getMock.mockImplementation(async (path: string) => {
    if (path === "/decisions/cases") return mockCases
    if (path.startsWith("/decisions/cases/") && !path.includes("/review")) return mockProposals
    if (path.startsWith("/proposals/") && path.endsWith("/review")) return mockReviewRecord
    return {}
  })
}

describe("DecisionReview", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders title and filters", async () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<DecisionReview />)
    expect(screen.getByText("Decision Review")).toBeInTheDocument()
  })

  it("shows loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<DecisionReview />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("shows error panel when cases fail to load", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<DecisionReview />)
    expect(await screen.findByText("请求异常")).toBeInTheDocument()
  })

  it("renders proposals with data", async () => {
    setupDefaultGet()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("Decision Review")).toBeInTheDocument()
    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    expect(screen.getByText("escalate")).toBeInTheDocument()
    expect(screen.getByText("medium")).toBeInTheDocument()
    expect(screen.getByText("high")).toBeInTheDocument()
    expect(screen.getAllByText("proposed").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("approved").length).toBeGreaterThanOrEqual(1)
  })

  it("shows proposal detail panel when a row is clicked", async () => {
    setupDefaultGet()
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))

    expect(screen.getByText("Proposal Details")).toBeInTheDocument()
    const ap1Elements = screen.getAllByText("ap_1")
    expect(ap1Elements.length).toBeGreaterThanOrEqual(1)
    const dcElements = screen.getAllByText("dc_123")
    expect(dcElements.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("dec_1")).toBeInTheDocument()
  })

  it("shows approve/reject/cancel buttons in detail panel", async () => {
    setupDefaultGet()
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))

    const buttons = screen.getAllByRole("button")
    const buttonTexts = buttons.map(b => b.textContent)
    expect(buttonTexts).toContain("Approve")
    expect(buttonTexts).toContain("Reject")
    expect(buttonTexts).toContain("Cancel")
  })

  it("shows proposal with description and payload", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/decisions/cases") return mockCases
      if (path.startsWith("/decisions/cases/") && !path.includes("/review")) return {
        proposals: [{
          ...mockProposals.proposals[1],
          description: "Escalate to manager",
          payload: { reason: "test" },
        }],
      }
      if (path.startsWith("/proposals/")) return mockReviewRecord
      return {}
    })

    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("escalate")).toBeInTheDocument()
    await user.click(screen.getByText("escalate"))

    expect(screen.getByText("Description")).toBeInTheDocument()
    expect(screen.getByText("Escalate to manager")).toBeInTheDocument()
    expect(screen.getByText("Payload")).toBeInTheDocument()
  })

  it("opens approve dialog and calls approve mutation", async () => {
    setupDefaultGet()
    vi.mocked(apiClient.post).mockResolvedValue({})
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))
    await user.click(screen.getByText("Approve"))

    expect(await screen.findByText("Approve Proposal")).toBeInTheDocument()
    const dialogInput = screen.getByPlaceholderText("Feedback (optional)")
    await user.type(dialogInput, "Looks good")

    const dialog = screen.getByRole("dialog")
    const confirmBtn = within(dialog).getByText("Approve")
    await user.click(confirmBtn)

    expect(apiClient.post).toHaveBeenCalledWith(
      "/proposals/ap_1/approve",
      { reviewer_id: "operator", feedback: "Looks good" },
    )
  })

  it("opens reject dialog and calls reject mutation", async () => {
    setupDefaultGet()
    vi.mocked(apiClient.post).mockResolvedValue({})
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))
    await user.click(screen.getByText("Reject"))

    expect(await screen.findByText("Reject Proposal")).toBeInTheDocument()
    const dialog = screen.getByRole("dialog")
    const rejectBtn = within(dialog).getByText("Reject")
    await user.click(rejectBtn)

    expect(apiClient.post).toHaveBeenCalledWith(
      "/proposals/ap_1/reject",
      { reviewer_id: "operator", feedback: "" },
    )
  })

  it("opens cancel dialog and calls cancel mutation", async () => {
    setupDefaultGet()
    vi.mocked(apiClient.post).mockResolvedValue({})
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))
    await user.click(screen.getByText("Cancel"))

    expect(await screen.findByText("Cancel Proposal")).toBeInTheDocument()
    const dialog = screen.getByRole("dialog")
    const confirmBtn = within(dialog).getByText("Confirm")
    await user.click(confirmBtn)

    expect(apiClient.post).toHaveBeenCalledWith(
      "/proposals/ap_1/cancel",
      { reviewer_id: "operator", feedback: "" },
    )
  })

  it("displays review history after selecting proposal", async () => {
    setupDefaultGet()
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))

    expect(await screen.findByText("Review History")).toBeInTheDocument()
    const approvedElements = screen.getAllByText("approved")
    expect(approvedElements.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText("admin")).toBeInTheDocument()
    expect(screen.getByText("Looks good")).toBeInTheDocument()
  })

  it("shows review history section even when review fetch is loading", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/decisions/cases") return mockCases
      if (path.startsWith("/decisions/cases/") && !path.includes("/review")) return mockProposals
      if (path.startsWith("/proposals/")) return new Promise(() => {})
      return {}
    })

    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()
    await user.click(screen.getByText("notify_owner"))

    expect(await screen.findByText("Review History")).toBeInTheDocument()
  })

  it("filters proposals by case search", async () => {
    setupDefaultGet()
    const user = userEvent.setup()
    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("notify_owner")).toBeInTheDocument()

    const searchInput = screen.getByPlaceholderText("Search case ID...")
    await user.type(searchInput, "NONEXISTENT")

    expect(screen.queryByText("notify_owner")).not.toBeInTheDocument()
    expect(screen.getByText("暂无决策案例")).toBeInTheDocument()
  })

  it("shows empty state when no proposals match filter", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/decisions/cases") return mockCases
      if (path.startsWith("/decisions/cases/")) return { proposals: [], count: 0 }
      return {}
    })

    renderWithQueryClient(<DecisionReview />)

    expect(await screen.findByText("Decision Review")).toBeInTheDocument()
    expect(await screen.findByText("No proposals match the current filter")).toBeInTheDocument()
  })
})
