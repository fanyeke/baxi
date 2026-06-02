import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ConfirmApplyDialog } from "../ConfirmApplyDialog"

describe("ConfirmApplyDialog", () => {
  it("renders dialog content when open", () => {
    render(
      <ConfirmApplyDialog
        open={true}
        title="Confirm deletion"
        description="Are you sure?"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.getByText("Confirm deletion")).toBeInTheDocument()
    expect(screen.getByText("Are you sure?")).toBeInTheDocument()
  })

  it("does not render when closed", () => {
    render(
      <ConfirmApplyDialog
        open={false}
        title="Hidden"
        description="Not visible"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.queryByText("Hidden")).not.toBeInTheDocument()
  })

  it("calls onConfirm when confirm button is clicked", () => {
    const onConfirm = vi.fn()
    const onCancel = vi.fn()

    render(
      <ConfirmApplyDialog
        open={true}
        title="Test"
        description="Test desc"
        onConfirm={onConfirm}
        onCancel={onCancel}
      />
    )

    const confirmBtn = screen.getByText("确认执行")
    fireEvent.click(confirmBtn)
    expect(onConfirm).toHaveBeenCalledTimes(1)
    expect(onCancel).not.toHaveBeenCalled()
  })

  it("calls onCancel when cancel button is clicked", () => {
    const onCancel = vi.fn()

    render(
      <ConfirmApplyDialog
        open={true}
        title="Cancel Test"
        description="Cancel desc"
        onConfirm={vi.fn()}
        onCancel={onCancel}
      />
    )

    const cancelBtn = screen.getByText("取消")
    fireEvent.click(cancelBtn)
    // Radix Dialog calls onOpenChange(false) which triggers onCancel,
    // and the cancel button's onClick also calls onCancel
    expect(onCancel).toHaveBeenCalled()
    expect(onCancel.mock.calls.length).toBeGreaterThanOrEqual(1)
  })

  it("uses custom confirm label when provided", () => {
    render(
      <ConfirmApplyDialog
        open={true}
        title="Custom Label"
        description="With custom label"
        confirmLabel="Deploy"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    expect(screen.getByText("Deploy")).toBeInTheDocument()
  })

  it("applies destructive styling by default", () => {
    render(
      <ConfirmApplyDialog
        open={true}
        title="Default Style"
        description="destructive check"
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    const title = screen.getByRole("heading", { name: "Default Style" })
    expect(title.className).toContain("text-destructive")
  })

  it("applies primary styling when destructive is false", () => {
    render(
      <ConfirmApplyDialog
        open={true}
        title="Primary Style"
        description="primary check"
        destructive={false}
        onConfirm={vi.fn()}
        onCancel={vi.fn()}
      />
    )

    const confirmBtn = screen.getByText("确认执行")
    expect(confirmBtn.className).toContain("bg-primary")
  })
})
