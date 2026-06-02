import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_SANDBOXES,
  MOCK_COMPARISON,
} from "./fixtures/api-mocks"

test.describe("Sandbox Compare page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/sandboxes/compare**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_COMPARISON),
      })
    })
    await page.route("**/api/v1/sandboxes**", async (route) => {
      if (route.request().url().includes("/compare")) return
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            sandbox_id: "sb-new",
            case_id: "case-001",
            data: {},
            status: "draft",
            compared_with: [],
            created_at: new Date().toISOString(),
          }),
        })
      } else {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_SANDBOXES),
        })
      }
    })

    await setupAuth(page)
    await page.goto("/sandbox")
  })

  test("renders page title and description", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("Sandbox")
    await expect(page.locator("text=Compare proposals in isolated sandbox")).toBeVisible()
  })

  test("renders Create Sandbox button", async ({ page }) => {
    await expect(page.locator("text=Create Sandbox")).toBeVisible()
  })

  test("renders sandbox table with data", async ({ page }) => {
    await expect(page.locator("text=sb-001").first()).toBeVisible({ timeout: 5000 })

    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
  })

  test("sandbox table shows status badges", async ({ page }) => {
    await expect(page.locator("text=draft").first()).toBeVisible({ timeout: 5000 })
  })

  test("sandbox table has checkboxes for selection", async ({ page }) => {
    const checkboxes = page.locator('table tbody input[type="checkbox"]')
    await expect(checkboxes).toHaveCount(2)
  })

  test("shows selection count message", async ({ page }) => {
    await expect(page.locator("text=0/2 selected")).toBeVisible({ timeout: 5000 })
  })

  test("selecting one sandbox updates count", async ({ page }) => {
    await page.locator('table tbody input[type="checkbox"]').first().click()

    await expect(page.locator("text=1/2 selected")).toBeVisible({ timeout: 5000 })
  })

  test("selecting two sandboxes shows comparison panel", async ({ page }) => {
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await page.locator('table tbody input[type="checkbox"]').nth(1).click()

    await expect(page.locator("text=Comparison")).toBeVisible({ timeout: 5000 })
  })

  test("comparison panel shows differences", async ({ page }) => {
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await page.locator('table tbody input[type="checkbox"]').nth(1).click()

    await expect(page.locator("text=gmv_target").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=50000").first()).toBeVisible()
    await expect(page.locator("text=60000").first()).toBeVisible()
  })

  test("comparison shows field, sandbox1, sandbox2 columns", async ({ page }) => {
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await page.locator('table tbody input[type="checkbox"]').nth(1).click()

    await expect(page.locator("text=Field").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Sandbox 1")).toBeVisible()
    await expect(page.locator("text=Sandbox 2")).toBeVisible()
  })

  test("Create Sandbox opens dialog", async ({ page }) => {
    await page.locator("text=Create Sandbox").click()

    await expect(page.locator("text=Create Sandbox").last()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Create a new sandbox for a decision case")).toBeVisible()
  })

  test("Create Sandbox dialog has case ID input", async ({ page }) => {
    await page.locator("text=Create Sandbox").click()

    const input = page.locator('input[placeholder="Case ID"]')
    await expect(input).toBeVisible({ timeout: 5000 })
  })

  test("Create Sandbox dialog has cancel and create buttons", async ({ page }) => {
    await page.locator("text=Create Sandbox").click()

    await expect(page.locator("text=Cancel").last()).toBeVisible({ timeout: 5000 })
  })

  test("create button is disabled when case ID is empty", async ({ page }) => {
    await page.locator("text=Create Sandbox").click()

    const createBtn = page.locator("button:has-text('Create'):not(:has-text('Sandbox'))").last()
    await expect(createBtn).toBeDisabled({ timeout: 5000 })
  })

  test("Add Proposal button exists in sandbox rows", async ({ page }) => {
    const addButtons = page.locator("button:has-text('Add Proposal')")
    await expect(addButtons).toHaveCount(2)
  })

  test("Add Proposal button opens dialog", async ({ page }) => {
    await page.locator("button:has-text('Add Proposal')").first().click()

    await expect(page.locator("text=Add Proposal to Sandbox")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Select a proposal to add to the sandbox")).toBeVisible()
  })

  test("shows empty state when no sandboxes", async ({ page }) => {
    await page.route("**/api/v1/sandboxes**", async (route) => {
      if (route.request().url().includes("/compare")) return
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [] }),
      })
    })

    await page.goto("/sandbox")
    await expect(page.locator("text=No sandboxes")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Create a sandbox to start comparing")).toBeVisible()
  })
})
