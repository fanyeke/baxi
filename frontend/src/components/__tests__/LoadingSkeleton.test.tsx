import { describe, it, expect } from "vitest"
import { render } from "@testing-library/react"
import { LoadingSkeleton } from "../LoadingSkeleton"

describe("LoadingSkeleton", () => {
  it("renders cards type with default count of 4", () => {
    const { container } = render(<LoadingSkeleton type="cards" />)
    const cards = container.querySelectorAll(".h-24")
    expect(cards.length).toBe(4)
  })

  it("renders cards type with custom count", () => {
    const { container } = render(<LoadingSkeleton type="cards" count={2} />)
    const cards = container.querySelectorAll(".h-24")
    expect(cards.length).toBe(2)
  })

  it("renders table type with default count of 3", () => {
    const { container } = render(<LoadingSkeleton type="table" />)
    const rows = container.querySelectorAll(".h-10")
    expect(rows.length).toBe(3)
  })

  it("renders table type with custom count", () => {
    const { container } = render(<LoadingSkeleton type="table" count={5} />)
    const rows = container.querySelectorAll(".h-10")
    expect(rows.length).toBe(5)
  })

  it("renders text type with default count of 3", () => {
    const { container } = render(<LoadingSkeleton type="text" />)
    const lines = container.querySelectorAll(".h-4")
    expect(lines.length).toBe(3)
  })

  it("renders text type with custom count", () => {
    const { container } = render(<LoadingSkeleton type="text" count={1} />)
    const lines = container.querySelectorAll(".h-4")
    expect(lines.length).toBe(1)
  })

  it("text items have decreasing widths", () => {
    const { container } = render(<LoadingSkeleton type="text" count={3} />)
    const lines = container.querySelectorAll<HTMLElement>(".h-4")
    expect(lines[0].style.width).toBe("80%")
    expect(lines[1].style.width).toBe("65%")
    expect(lines[2].style.width).toBe("50%")
  })
})
