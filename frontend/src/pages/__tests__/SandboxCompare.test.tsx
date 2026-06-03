import { describe, it, expect, vi, beforeEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithQueryClient } from "@/test-setup"
import SandboxCompare from "../SandboxCompare"
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

const mockSandboxes = {
  items: [
    { sandbox_id: "sbx_1", case_id: "case-1", data: { price: 100 }, status: "draft", compared_with: [], created_at: "2026-05-28T10:00:00Z" },
    { sandbox_id: "sbx_2", case_id: "case-1", data: { price: 90 }, status: "comparing", compared_with: ["sbx_1"], created_at: "2026-05-28T11:00:00Z" },
  ],
}

const mockComparisonResult = {
  sandbox_1_id: "sbx_1",
  sandbox_2_id: "sbx_2",
  differences: [
    { field: "price", value_1: 100, value_2: 90 },
  ],
}

describe("SandboxCompare", () => {
  beforeEach(() => { vi.clearAllMocks() })

  it("renders title and description", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<SandboxCompare />)
    expect(screen.getByText("Sandbox")).toBeInTheDocument()
    expect(screen.getByText("Compare proposals in isolated sandbox environments")).toBeInTheDocument()
  })

  it("shows loading skeleton", () => {
    vi.mocked(apiClient.get).mockImplementation(() => new Promise(() => {}))
    renderWithQueryClient(<SandboxCompare />)
    const skeletons = document.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("shows error panel", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"))
    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("Failed to load")).toBeInTheDocument()
  })

  it("shows empty state when no sandboxes", async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ items: [] })
    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("No sandboxes")).toBeInTheDocument()
  })

  it("renders sandbox list with data", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockSandboxes)
    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("Sandbox")).toBeInTheDocument()
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()
    const caseElements = screen.getAllByText("case-1"); expect(caseElements.length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText("draft")).toBeInTheDocument()
    expect(screen.getByText("comparing")).toBeInTheDocument()
  })

  it("shows comparison results when two sandboxes are selected", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/sandboxes") return mockSandboxes
      if (path === "/sandboxes/compare") return mockComparisonResult
      return {}
    })

    const user = userEvent.setup()
    renderWithQueryClient(<SandboxCompare />)

    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    const checkboxes = document.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')
    expect(checkboxes.length).toBe(2)
    await user.click(checkboxes[0])
    await user.click(checkboxes[1])

    expect(await screen.findByText("Comparison")).toBeInTheDocument()
    expect(screen.getByText("price")).toBeInTheDocument()
    expect(screen.getByText("100")).toBeInTheDocument()
    expect(screen.getByText("90")).toBeInTheDocument()
  })

  it("shows no differences message when comparison has no diffs", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/sandboxes") return mockSandboxes
      if (path === "/sandboxes/compare") return {
        sandbox_1_id: "sbx_1",
        sandbox_2_id: "sbx_2",
        differences: [],
      }
      return {}
    })

    const user = userEvent.setup()
    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    const checkboxes = document.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')
    await user.click(checkboxes[0])
    await user.click(checkboxes[1])

    expect(await screen.findByText("No differences found between selected sandboxes")).toBeInTheDocument()
  })

  it("shows comparison error panel", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path === "/sandboxes") return mockSandboxes
      if (path === "/sandboxes/compare") throw new Error("Compare failed")
      return {}
    })

    const user = userEvent.setup()
    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    const checkboxes = document.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')
    await user.click(checkboxes[0])
    await user.click(checkboxes[1])

    expect(await screen.findByText("Comparison failed")).toBeInTheDocument()
  })

  it("shows selection count when fewer than 2 selected", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockSandboxes)
    renderWithQueryClient(<SandboxCompare />)

    expect(await screen.findByText(/0\/2 selected/)).toBeInTheDocument()
  })

  it("opens create sandbox dialog", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockSandboxes)
    const user = userEvent.setup()

    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    await user.click(screen.getByText("Create Sandbox"))
    expect(screen.getByText("Create a new sandbox for a decision case")).toBeInTheDocument()
  })

  it("calls apiClient.post when creating sandbox", async () => {
    vi.mocked(apiClient.get).mockResolvedValue(mockSandboxes)
    vi.mocked(apiClient.post).mockResolvedValue({})
    const user = userEvent.setup()

    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    await user.click(screen.getByText("Create Sandbox"))
    const input = screen.getByPlaceholderText("Case ID")
    await user.type(input, "case-123")
    await user.click(screen.getByText("Create"))

    expect(apiClient.post).toHaveBeenCalledWith("/sandboxes", { case_id: "case-123", data: {} })
  })

  it("opens add proposal dialog", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path.startsWith("/sandboxes")) return mockSandboxes
      if (path.startsWith("/decisions/cases/")) return { proposals: [], count: 0 }
      return {}
    })
    const user = userEvent.setup()

    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    const addButtons = screen.getAllByText("Add Proposal")
    await user.click(addButtons[0])

    expect(await screen.findByText("Select a proposal to add to the sandbox")).toBeInTheDocument()
  })

  it("calls apiClient.post when adding proposal to sandbox", async () => {
    const getMock = vi.mocked(apiClient.get)
    getMock.mockImplementation(async (path: string) => {
      if (path.startsWith("/sandboxes")) return mockSandboxes
      if (path.startsWith("/decisions/cases/")) return { proposals: [], count: 0 }
      return {}
    })
    vi.mocked(apiClient.post).mockResolvedValue({})
    const user = userEvent.setup()

    renderWithQueryClient(<SandboxCompare />)
    expect(await screen.findByText("sbx_1")).toBeInTheDocument()

    const addButtons = screen.getAllByText("Add Proposal")
    await user.click(addButtons[0])

    const input = screen.getByPlaceholderText("Proposal ID")
    await user.type(input, "prop-123")
    await user.click(screen.getByText("Add"))

    expect(apiClient.post).toHaveBeenCalledWith("/sandboxes/sbx_1/proposals", { proposal_id: "prop-123" })
  })
})
