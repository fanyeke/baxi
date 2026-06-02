import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ErrorPanel } from "../ErrorPanel"

describe("ErrorPanel", () => {
  it("renders title and message", () => {
    render(<ErrorPanel title="Error" message="Something went wrong" />)
    expect(screen.getByText("Error")).toBeInTheDocument()
    expect(screen.getByText("Something went wrong")).toBeInTheDocument()
  })

  it("renders diagnosis when provided", () => {
    render(<ErrorPanel title="Error" message="msg" diagnosis="DB connection failed" />)
    expect(screen.getByText("DB connection failed")).toBeInTheDocument()
  })

  it("renders suggested action when provided", () => {
    render(<ErrorPanel title="Error" message="msg" suggested_action="Restart the service" />)
    expect(screen.getByText("Restart the service")).toBeInTheDocument()
  })

  it("renders request_id when provided", () => {
    render(<ErrorPanel title="Error" message="msg" request_id="req-12345-abcde" />)
    expect(screen.getByText(/req-12345/)).toBeInTheDocument()
  })

  it("renders retry button and calls onRetry when clicked", () => {
    const onRetry = vi.fn()
    render(<ErrorPanel title="Error" message="msg" onRetry={onRetry} />)
    const btn = screen.getByText("重试")
    expect(btn).toBeInTheDocument()
    fireEvent.click(btn)
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it("applies inline variant class when variant is inline", () => {
    const { container } = render(<ErrorPanel title="Error" message="msg" variant="inline" />)
    const root = container.firstChild as HTMLElement
    expect(root.className).toContain("text-sm")
  })

  it("applies full variant by default", () => {
    const { container } = render(<ErrorPanel title="Error" message="msg" />)
    const root = container.firstChild as HTMLElement
    expect(root.className).not.toContain("text-sm")
  })
})
