import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { StatusCard } from "../Feishu"

describe("StatusCard", () => {
  it("shows success/neutral styling for synced status", () => {
    const { container } = render(
      <StatusCard title="同步" result={{ status: "synced" }} />
    )
    const card = container.firstChild as HTMLElement
    expect(card).toHaveClass("bg-muted/30")
    expect(card).not.toHaveClass("bg-destructive/10")
    expect(screen.getByText("synced")).not.toHaveClass("text-destructive")
  })

  it("shows success/neutral styling for imported status", () => {
    const { container } = render(
      <StatusCard title="导入" result={{ status: "imported" }} />
    )
    const card = container.firstChild as HTMLElement
    expect(card).toHaveClass("bg-muted/30")
    expect(card).not.toHaveClass("bg-destructive/10")
  })

  it("shows success/neutral styling for exported status", () => {
    const { container } = render(
      <StatusCard title="导出" result={{ status: "exported" }} />
    )
    const card = container.firstChild as HTMLElement
    expect(card).toHaveClass("bg-muted/30")
    expect(card).not.toHaveClass("bg-destructive/10")
  })
})
