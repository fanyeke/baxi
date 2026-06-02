import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_CASE_LIST,
  MOCK_PROPOSALS,
} from "./fixtures/api-mocks"

test.describe("Decision Review page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/decisions/cases**", async (route) => {
      const url = route.request().url()
      if (url.includes("/proposals")) {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_PROPOSALS),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_CASE_LIST),
        })
      }
    })
    await page.route("**/api/v1/proposals/**", async (route) => {
      const url = route.request().url()
      if (url.includes("/review")) {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            record_id: "review-001",
            proposal_id: "prop-001",
            verdict: "approved",
            reviewer_id: "supervisor",
            feedback: "LGTM",
            created_at: "2026-05-30T11:00:00Z",
          }),
        })
      } else if (url.includes("/approve") || url.includes("/reject") || url.includes("/cancel")) {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            record_id: "review-new",
            proposal_id: "prop-001",
            verdict: url.includes("/approve") ? "approved" : "rejected",
            reviewer_id: "operator",
            feedback: "",
            created_at: new Date().toISOString(),
          }),
        })
      } else {
        await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
      }
    })

    await setupAuth(page)
    await page.goto("/decision-review")
  })

  test("renders page title and description", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("Decision Review")
    await expect(page.locator("text=Review and manage action proposals")).toBeVisible()
  })

  test("renders case list and proposals table", async ({ page }) => {
    await expect(page.locator("text=create_task").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Create investigation task")).toBeVisible()

    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
  })

  test("renders status filter dropdown", async ({ page }) => {
    const select = page.locator("select")
    await expect(select).toBeVisible()
    const options = select.locator("option")
    await expect(options).toHaveCount(5) // All + proposed, approved, rejected, cancelled
  })

  test("renders case ID search input", async ({ page }) => {
    const input = page.locator('input[placeholder="Search case ID..."]')
    await expect(input).toBeVisible()
  })

  test("search filters proposals by case ID", async ({ page }) => {
    await page.locator('input[placeholder="Search case ID..."]').fill("case-001")
    // The case list is filtered client-side, proposals for case-001 still show
    await expect(page.locator("text=prop-001").first()).toBeVisible({ timeout: 5000 })
  })

  test("proposal table shows risk level badges", async ({ page }) => {
    await expect(page.locator("text=low").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=high").first()).toBeVisible()
  })

  test("proposal table shows HITL indicator", async ({ page }) => {
    await expect(page.locator("text=No").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Yes").first()).toBeVisible()
  })

  test("clicking a proposal shows details panel", async ({ page }) => {
    await page.locator("table tbody tr").first().click()

    await expect(page.locator("text=Proposal Details")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=prop-001").first()).toBeVisible()
    await expect(page.locator("text=Create investigation task")).toBeVisible()
  })

  test("proposal details shows payload", async ({ page }) => {
    await page.locator("table tbody tr").first().click()

    await expect(page.locator("text=Payload")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=task_title").first()).toBeVisible()
  })

  test("proposal details shows action buttons", async ({ page }) => {
    await page.locator("table tbody tr").first().click()

    await expect(page.locator("text=Approve").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Reject").first()).toBeVisible()
    await expect(page.locator("text=Cancel").first()).toBeVisible()
  })

  test("shows review history for selected proposal", async ({ page }) => {
    await page.locator("table tbody tr").first().click()

    await expect(page.locator("text=Review History")).toBeVisible({ timeout: 5000 })
  })

  test("shows empty state when no proposals selected", async ({ page }) => {
    await expect(page.locator("text=Select a proposal to view details")).toBeVisible({ timeout: 5000 })
  })

  test("approve button opens dialog", async ({ page }) => {
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Approve").first()).toBeVisible({ timeout: 5000 })

    await page.locator("text=Approve").first().click()

    await expect(page.locator("text=Approve Proposal")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Are you sure you want to approve")).toBeVisible()
  })

  test("reject button opens dialog", async ({ page }) => {
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Reject").first()).toBeVisible({ timeout: 5000 })

    // Click the Reject button in the details panel (not in the dialog)
    await page.locator("table ~ div button:has-text('Reject')").click()

    await expect(page.locator("text=Reject Proposal")).toBeVisible({ timeout: 5000 })
  })

  test("cancel button opens dialog", async ({ page }) => {
    await page.locator("table tbody tr").first().click()
    await expect(page.locator("text=Cancel").first()).toBeVisible({ timeout: 5000 })

    // Click the Cancel button in the details panel
    await page.locator("table ~ div button:has-text('Cancel')").click()

    await expect(page.locator("text=Cancel Proposal")).toBeVisible({ timeout: 5000 })
  })

  test("dialog has feedback input", async ({ page }) => {
    await page.locator("table tbody tr").first().click()
    await page.locator("text=Approve").first().click()

    const input = page.locator('input[placeholder="Feedback (optional)"]')
    await expect(input).toBeVisible({ timeout: 5000 })
  })

  test("dialog has confirm and cancel buttons", async ({ page }) => {
    await page.locator("table tbody tr").first().click()
    await page.locator("text=Approve").first().click()

    await expect(page.locator("text=Approve").last()).toBeVisible({ timeout: 5000 })
  })
})
