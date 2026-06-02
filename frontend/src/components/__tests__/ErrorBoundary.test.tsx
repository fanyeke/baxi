import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { ErrorBoundary } from "../ErrorBoundary"

vi.spyOn(console, "error").mockImplementation(() => {})

function ThrowError({ message }: { message?: string }): React.ReactNode {
  throw new Error(message)
  return null
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders children when no error occurs", () => {
    render(
      <ErrorBoundary>
        <div>Child Content</div>
      </ErrorBoundary>,
    )
    expect(screen.getByText("Child Content")).toBeInTheDocument()
  })

  it("catches errors and shows fallback UI with error message", () => {
    render(
      <ErrorBoundary>
        <ThrowError message="Something went wrong" />
      </ErrorBoundary>,
    )
    expect(screen.getByText("应用发生了错误")).toBeInTheDocument()
    expect(screen.getByText("Something went wrong")).toBeInTheDocument()
    expect(screen.getByText("重新加载")).toBeInTheDocument()
  })

  it("renders default error text when error has no message", () => {
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>,
    )
    expect(screen.getByText("未知错误，请尝试刷新页面")).toBeInTheDocument()
  })

  it("calls window.location.reload when reload button is clicked", () => {
    const reloadMock = vi.fn()
    Object.defineProperty(window, "location", {
      value: { reload: reloadMock },
      writable: true,
    })

    render(
      <ErrorBoundary>
        <ThrowError message="err" />
      </ErrorBoundary>,
    )
    screen.getByText("重新加载").click()
    expect(reloadMock).toHaveBeenCalledTimes(1)
  })

  it("componentDidCatch logs to console.error", () => {
    const consoleSpy = vi.spyOn(console, "error")
    render(
      <ErrorBoundary>
        <ThrowError message="log me" />
      </ErrorBoundary>,
    )
    expect(consoleSpy).toHaveBeenCalledWith(
      "[ErrorBoundary]",
      expect.any(Error),
      expect.any(Object),
    )
  })
})
