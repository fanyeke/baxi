import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_SANDBOXES,
  MOCK_COMPARISON,
} from "../fixtures/api-mocks"

test.describe("Sandbox Flow", () => {
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

    await setupAuth(page)
  })

  test("create sandbox then compare flow", async ({ page }) => {
    await page.goto("/sandbox")

    // 1. Verify existing sandboxes
    await expect(page.locator("text=sb-001").first()).toBeVisible({ timeout: 5000 })
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)

    // 2. Select two sandboxes to compare
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await page.locator('table tbody input[type="checkbox"]').nth(1).click()

    // 3. Verify comparison panel appears
    await expect(page.locator("text=Comparison")).toBeVisible({ timeout: 5000 })

    // 4. Verify comparison data
    await expect(page.locator("text=gmv_target").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=50000").first()).toBeVisible()
    await expect(page.locator("text=60000").first()).toBeVisible()
    await expect(page.locator("text=pricing").first()).toBeVisible()
  })

  test("create sandbox via dialog flow", async ({ page }) => {
    await page.goto("/sandbox")

    await expect(page.locator("text=sb-001").first()).toBeVisible({ timeout: 5000 })

    // 1. Click Create Sandbox
    await page.locator("text=Create Sandbox").click()
    await expect(page.locator('input[placeholder="Case ID"]')).toBeVisible({ timeout: 5000 })

    // 2. Enter case ID
    await page.locator('input[placeholder="Case ID"]').fill("case-003")

    // 3. Click Create
    const createBtn = page.locator("button:has-text('Create'):not(:has-text('Sandbox'))").last()
    await expect(createBtn).toBeEnabled()
    await createBtn.click()

    // 4. Dialog should close
    await expect(page.locator('input[placeholder="Case ID"]')).not.toBeVisible({ timeout: 5000 })
  })

  test("add proposal to sandbox flow", async ({ page }) => {
    await page.goto("/sandbox")

    await expect(page.locator("text=sb-001").first()).toBeVisible({ timeout: 5000 })

    // 1. Click Add Proposal on first sandbox
    await page.locator("button:has-text('Add Proposal')").first().click()
    await expect(page.locator("text=Add Proposal to Sandbox")).toBeVisible({ timeout: 5000 })

    // 2. Enter proposal ID
    await page.locator('input[placeholder="Proposal ID"]').fill("prop-001")

    // 3. Click Add
    const addBtn = page.locator("button:has-text('Add'):not(:has-text('Proposal'))").last()
    await addBtn.click()

    // 4. Dialog should close
    await expect(page.locator("text=Add Proposal to Sandbox")).not.toBeVisible({ timeout: 5000 })
  })

  test("deselecting sandbox updates selection count", async ({ page }) => {
    await page.goto("/sandbox")

    await expect(page.locator("text=sb-001").first()).toBeVisible({ timeout: 5000 })

    // Select first
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await expect(page.locator("text=1/2 selected")).toBeVisible({ timeout: 5000 })

    // Select second
    await page.locator('table tbody input[type="checkbox"]').nth(1).click()
    await expect(page.locator("text=Comparison")).toBeVisible({ timeout: 5000 })

    // Deselect first
    await page.locator('table tbody input[type="checkbox"]').first().click()
    await expect(page.locator("text=1/2 selected")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Comparison")).not.toBeVisible()
  })
})
