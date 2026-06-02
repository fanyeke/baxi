import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_TASKS,
} from "./fixtures/api-mocks"

test.describe("Tasks page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/tasks**", async (route) => {
      const url = new URL(route.request().url())
      const status = url.searchParams.get("status") || ""
      const priority = url.searchParams.get("priority") || ""

      let items = MOCK_TASKS.items
      if (status) items = items.filter((t) => t.status === status)
      if (priority) items = items.filter((t) => t.priority === priority)

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items, total: items.length }),
      })
    })

    await setupAuth(page)
    await page.goto("/tasks")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("任务中心")
  })

  test("renders tasks table with data", async ({ page }) => {
    await expect(page.locator("text=task-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Investigate GMV drop").first()).toBeVisible()

    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(3)
  })

  test("renders status and priority filter dropdowns", async ({ page }) => {
    const selects = page.locator("select")
    await expect(selects).toHaveCount(2)
  })

  test("filters by status", async ({ page }) => {
    await page.locator("select").nth(0).selectOption("todo")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=Investigate GMV drop")).toBeVisible({ timeout: 5000 })
  })

  test("filters by priority", async ({ page }) => {
    await page.locator("select").nth(1).selectOption("medium")
    await expect(page.locator("table tbody tr")).toHaveCount(1)
    await expect(page.locator("text=Review delivery SLA")).toBeVisible({ timeout: 5000 })
  })

  test("shows priority color badges", async ({ page }) => {
    const highBadge = page.locator("table tbody tr").first().locator("span").filter({ hasText: "high" })
    await expect(highBadge).toBeVisible()
    await expect(highBadge).toHaveClass(/bg-red-100/)
  })

  test("displays due date or dash for null due dates", async ({ page }) => {
    // task-003 has no due_at, should show dash
    const rows = page.locator("table tbody tr")
    await expect(rows.nth(2)).toContainText("—")
  })

  test("shows empty state when no tasks", async ({ page }) => {
    await page.route("**/api/v1/tasks**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ items: [], total: 0 }),
      })
    })

    await page.locator("select").nth(0).selectOption("done")
    await page.locator("select").nth(1).selectOption("high")

    await expect(page.locator("text=暂无任务")).toBeVisible({ timeout: 5000 })
  })
})
