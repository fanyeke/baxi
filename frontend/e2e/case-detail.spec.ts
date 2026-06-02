import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_CASE,
  MOCK_CASE_ACTION_EXECUTED,
} from "./fixtures/api-mocks"

const CASE_ID = "case-001"

test.describe("Case Detail page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/decisions/cases**", async (route) => {
      const url = route.request().url()
      if (url.includes("/proposals")) {
        await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CASE),
      })
    })

    await setupAuth(page)
    await page.goto(`/cases/${CASE_ID}`)
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("案件详情")
  })

  test("renders basic info section", async ({ page }) => {
    await expect(page.locator("text=基本信息")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=案件 ID")).toBeVisible()
    await expect(page.locator("text=case-001").first()).toBeVisible()
  })

  test("displays case status with badge", async ({ page }) => {
    await expect(page.locator("text=completed").first()).toBeVisible({ timeout: 5000 })
  })

  test("displays object type and id", async ({ page }) => {
    await expect(page.locator("text=metric").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=gmv_daily").first()).toBeVisible()
  })

  test("displays severity badge", async ({ page }) => {
    const severityBadge = page.locator("span").filter({ hasText: "high" }).first()
    await expect(severityBadge).toBeVisible()
  })

  test("displays context hash", async ({ page }) => {
    await expect(page.locator("text=abc123def456")).toBeVisible({ timeout: 5000 })
  })

  test("displays source info", async ({ page }) => {
    await expect(page.locator("text=来源类型")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=alert").first()).toBeVisible()
  })

  test("renders policy results section", async ({ page }) => {
    await expect(page.locator("text=策略执行结果")).toBeVisible({ timeout: 5000 })
  })

  test("shows human approval required", async ({ page }) => {
    await expect(page.locator("text=人工审批")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=需要")).toBeVisible()
  })

  test("shows allowed actions", async ({ page }) => {
    await expect(page.locator("text=允许的操作")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=notify").first()).toBeVisible()
    await expect(page.locator("text=create_task").first()).toBeVisible()
  })

  test("shows blocked actions", async ({ page }) => {
    await expect(page.locator("text=阻止的操作")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=delete_order")).toBeVisible()
    await expect(page.locator("text=Requires VP approval")).toBeVisible()
  })

  test("shows risk levels", async ({ page }) => {
    await expect(page.locator("text=风险等级")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=gmv_daily").first()).toBeVisible()
  })

  test("shows requires approval actions", async ({ page }) => {
    await expect(page.locator("text=需审批的操作")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=update_pricing").first()).toBeVisible()
  })

  test("shows evidence sources", async ({ page }) => {
    await expect(page.locator("text=证据来源")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=gmv_drop_event")).toBeVisible()
    await expect(page.locator("text=historical_comparison")).toBeVisible()
  })

  test("does not show outbox link when status is not action_executed", async ({ page }) => {
    await expect(page.locator("text=查看 Outbox 分发")).not.toBeVisible()
  })

  test("shows outbox link when status is action_executed", async ({ page }) => {
    await page.route("**/api/v1/decisions/cases**", async (route) => {
      if (route.request().url().includes("/proposals")) {
        await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CASE_ACTION_EXECUTED),
      })
    })

    await page.goto(`/cases/${CASE_ID}`)

    await expect(page.locator("text=查看 Outbox 分发")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=该案件的操作已执行")).toBeVisible()
  })

  test("outbox link points to /outbox", async ({ page }) => {
    await page.route("**/api/v1/decisions/cases**", async (route) => {
      if (route.request().url().includes("/proposals")) {
        await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
        return
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CASE_ACTION_EXECUTED),
      })
    })

    await page.goto(`/cases/${CASE_ID}`)

    const link = page.locator('a[href="/outbox"]')
    await expect(link).toBeVisible({ timeout: 5000 })
  })

  test("shows empty state for unknown case", async ({ page }) => {
    await page.route("**/api/v1/decisions/cases**", async (route) => {
      if (route.request().url().includes("/proposals")) {
        await route.fulfill({ status: 200, contentType: "application/json", body: "{}" })
        return
      }
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ error_code: "NOT_FOUND", message: "Case not found" }),
      })
    })

    await page.goto("/cases/nonexistent")
    await expect(page.locator("text=加载失败").or(page.locator("text=未找到案件"))).toBeVisible({ timeout: 10000 })
  })
})
