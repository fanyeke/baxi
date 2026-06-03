import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import userEvent from "@testing-library/user-event"
import Pipeline from "../Pipeline"
import { apiClient } from "@/api/client"

vi.mock("@/api/client", () => ({
  apiClient: {
    post: vi.fn(),
  },
}))

function renderWithQueryClient(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

const mockPipelineResponse = {
  command: "python3 scripts/phase03_overall_business_analysis.py --mode daily",
  estimated_duration: "~15 minutes",
  pipeline_type: "daily",
  required_env_vars: ["API_KEY"],
  warnings: [],
}

describe("Pipeline", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("does NOT call apiClient.post on initial render (uses mutation, not query)", () => {
    renderWithQueryClient(<Pipeline />)

    // useQuery with enabled:true would auto-trigger POST on render
    // useMutation should NOT auto-trigger
    expect(apiClient.post).not.toHaveBeenCalled()
  })

  it("calls apiClient.post only when user clicks the trigger button", async () => {
    vi.mocked(apiClient.post).mockResolvedValue(mockPipelineResponse)
    renderWithQueryClient(<Pipeline />)

    // Before click: no POST
    expect(apiClient.post).not.toHaveBeenCalled()

    // Click trigger button
    const user = userEvent.setup()
    const triggerButton = screen.getByText("查看预览")
    await user.click(triggerButton)

    // After click: POST should be called
    expect(apiClient.post).toHaveBeenCalledWith("/pipeline/run", {
      config: "daily",
    })
  })

  it("shows result on successful mutation", async () => {
    vi.mocked(apiClient.post).mockResolvedValue(mockPipelineResponse)
    renderWithQueryClient(<Pipeline />)

    const user = userEvent.setup()
    await user.click(screen.getByText("查看预览"))

    // Should show the command preview
    expect(
      await screen.findByText("python3 scripts/phase03_overall_business_analysis.py --mode daily")
    ).toBeInTheDocument()
    expect(screen.getByText("~15 minutes")).toBeInTheDocument()
    expect(screen.getByText("API_KEY")).toBeInTheDocument()
  })

  it("switches pipeline type and runs mutation with correct type", async () => {
    vi.mocked(apiClient.post).mockResolvedValue({
      ...mockPipelineResponse,
      pipeline_type: "full",
    })
    renderWithQueryClient(<Pipeline />)

    const user = userEvent.setup()

    // Select "full" type (second radio)
    const radios = screen.getAllByRole("radio")
    await user.click(radios[1])

    // Click trigger
    await user.click(screen.getByText("查看预览"))

    expect(await screen.findByText("查看预览")).toBeInTheDocument()
    expect(apiClient.post).toHaveBeenCalledWith("/pipeline/run", {
      config: "full",
    })
  })
})
