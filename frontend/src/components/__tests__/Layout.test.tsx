import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import Layout from "../Layout"

function renderWithRouter() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/"]}>
        <Layout />
      </MemoryRouter>
    </QueryClientProvider>
  )
}

describe("Layout", () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it("renders app title", () => {
    renderWithRouter()
    expect(screen.getByText("Olist 决策中台")).toBeInTheDocument()
  })

  it("renders all navigation items", () => {
    renderWithRouter()
    expect(screen.getByText("总览")).toBeInTheDocument()
    expect(screen.getByText("告警中心")).toBeInTheDocument()
    expect(screen.getByText("任务中心")).toBeInTheDocument()
    expect(screen.getByText("Outbox 分发")).toBeInTheDocument()
    expect(screen.getByText("日志诊断")).toBeInTheDocument()
    expect(screen.getByText("飞书同步")).toBeInTheDocument()
    expect(screen.getByText("运行管道")).toBeInTheDocument()
    expect(screen.getByText("治理中心")).toBeInTheDocument()
    expect(screen.getByText("Agent 日志")).toBeInTheDocument()
  })

  it("renders API token input", () => {
    renderWithRouter()
    const tokenInput = screen.getByPlaceholderText("Bearer Token")
    expect(tokenInput).toBeInTheDocument()
  })

  it("stores API token in sessionStorage on change", () => {
    renderWithRouter()
    const tokenInput = screen.getByPlaceholderText("Bearer Token")
    fireEvent.change(tokenInput, { target: { value: "new-token" } })
    expect(sessionStorage.getItem("API_BEARER_TOKEN")).toBe("new-token")
  })

  it("uses default token when no token is stored", () => {
    renderWithRouter()
    expect(sessionStorage.getItem("API_BEARER_TOKEN")).toBe("test-token-for-dev-b2kA3QOBBD48wLFQAgAtLw")
  })

  it("loads existing token from sessionStorage", () => {
    sessionStorage.setItem("API_BEARER_TOKEN", "existing-token")
    renderWithRouter()
    const tokenInput = screen.getByPlaceholderText("Bearer Token") as HTMLInputElement
    expect(tokenInput.value).toBe("existing-token")
  })

  it("renders nav links with correct paths", () => {
    renderWithRouter()
    const alertLink = screen.getByText("告警中心").closest("a")
    expect(alertLink).toHaveAttribute("href", "/alerts")
  })
})
