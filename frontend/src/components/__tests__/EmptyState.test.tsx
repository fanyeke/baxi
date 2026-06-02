import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { EmptyState } from "../EmptyState"

describe("EmptyState", () => {
  it("renders title", () => {
    render(<EmptyState title="No data" />)
    expect(screen.getByText("No data")).toBeInTheDocument()
  })

  it("renders default icon", () => {
    render(<EmptyState title="Empty" />)
    expect(screen.getByText("○")).toBeInTheDocument()
  })

  it("renders custom icon when provided", () => {
    render(<EmptyState title="Empty" icon="⚠" />)
    expect(screen.getByText("⚠")).toBeInTheDocument()
  })

  it("renders description when provided", () => {
    render(<EmptyState title="No results" description="Try adjusting your filters" />)
    expect(screen.getByText("Try adjusting your filters")).toBeInTheDocument()
  })

  it("does not render description when not provided", () => {
    const { container } = render(<EmptyState title="No data" />)
    const smallText = container.querySelectorAll(".text-xs")
    expect(smallText.length).toBe(0)
  })
})
