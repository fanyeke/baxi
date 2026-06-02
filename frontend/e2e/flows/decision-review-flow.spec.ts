import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockAllApis,
  MOCK_CASE_LIST,
  MOCK_PROPOSALS,
} from "../fixtures/api-mocks"

test.describe("Decision Review Flow", () => {
  test.beforeEach(async ({ page }) => {
    await mockAllApis(page)
    await setupAuth(page)
  })

  test("full approve flow: select proposal, approve, verify dialog", async ({ page }) => {
    await page.goto("/decision-review")

    // 1. Wait for proposals to load
    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })

    // 2. Select first proposal
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Proposal Details")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=prop-001").first()).toBeVisible()

    // 3. Click Approve button
    await page.locator("button:has-text('Approve')").first().click()
    await expect(page.locator("text=Approve Proposal")).toBeVisible({ timeout: 5000 })

    // 4. Enter feedback
    await page.locator('input[placeholder="Feedback (optional)"]').fill("Looks good")

    // 5. Confirm approval
    await page.locator("button:has-text('Approve')").last().click()

    // 6. Dialog should close
    await expect(page.locator("text=Approve Proposal")).not.toBeVisible({ timeout: 5000 })
  })

  test("full reject flow: select proposal, reject with feedback", async ({ page }) => {
    await page.goto("/decision-review")

    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })

    // Select first proposal
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Proposal Details")).toBeVisible({ timeout: 5000 })

    // Click the Reject button in details panel
    const detailsPanel = page.locator("div").filter({ hasText: "Proposal Details" }).last()
    await detailsPanel.locator("button:has-text('Reject')").click()

    await expect(page.locator("text=Reject Proposal")).toBeVisible({ timeout: 5000 })

    // Enter rejection reason
    await page.locator('input[placeholder="Feedback (optional)"]').fill("Too risky")

    // Confirm rejection
    await page.locator("button:has-text('Reject')").last().click()

    // Dialog should close
    await expect(page.locator("text=Reject Proposal")).not.toBeVisible({ timeout: 5000 })
  })

  test("cancel flow: select proposal, cancel with reason", async ({ page }) => {
    await page.goto("/decision-review")

    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })

    // Select first proposal
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Proposal Details")).toBeVisible({ timeout: 5000 })

    // Click the Cancel button in details panel
    const detailsPanel = page.locator("div").filter({ hasText: "Proposal Details" }).last()
    await detailsPanel.locator("button:has-text('Cancel')").click()

    await expect(page.locator("text=Cancel Proposal")).toBeVisible({ timeout: 5000 })

    // Enter reason
    await page.locator('input[placeholder="Reason (optional)"]').fill("Changed mind")

    // Confirm cancel
    await page.locator("button:has-text('Confirm')").click()

    // Dialog should close
    await expect(page.locator("text=Cancel Proposal")).not.toBeVisible({ timeout: 5000 })
  })

  test("dialog cancel closes without action", async ({ page }) => {
    await page.goto("/decision-review")

    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })

    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Proposal Details")).toBeVisible({ timeout: 5000 })

    // Open approve dialog
    await page.locator("button:has-text('Approve')").first().click()
    await expect(page.locator("text=Approve Proposal")).toBeVisible({ timeout: 5000 })

    // Click Cancel in dialog
    await page.locator("button:has-text('Cancel')").last().click()

    // Dialog should be closed
    await expect(page.locator("text=Approve Proposal")).not.toBeVisible()
  })

  test("search and filter flow: filter by status, search case", async ({ page }) => {
    await page.goto("/decision-review")

    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })

    // Search for a specific case
    await page.locator('input[placeholder="Search case ID..."]').fill("case-001")
    await expect(page.locator("table tbody tr").first()).toBeVisible({ timeout: 5000 })

    // Filter by status
    await page.locator("select").selectOption("proposed")
    await expect(page.locator("table tbody tr").first()).toBeVisible({ timeout: 5000 })

    // Search for non-existent case
    await page.locator('input[placeholder="Search case ID..."]').fill("nonexistent")
    await expect(page.locator("text=No proposals match the current filter").or(
      page.locator("text=No cases found")
    )).toBeVisible({ timeout: 5000 })
  })
})
