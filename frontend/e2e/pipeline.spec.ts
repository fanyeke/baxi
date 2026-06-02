import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_PIPELINE_DAILY,
  MOCK_PIPELINE_FULL,
  MOCK_PIPELINE_DB_FULL,
} from "./fixtures/api-mocks"

test.describe("Pipeline page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/pipeline/run", async (route) => {
      if (route.request().method() === "POST") {
        const body = JSON.parse(route.request().postData() || "{}")
        const mockData =
          body.pipeline_type === "full"
            ? MOCK_PIPELINE_FULL
            : body.pipeline_type === "db_full"
              ? MOCK_PIPELINE_DB_FULL
              : MOCK_PIPELINE_DAILY
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(mockData),
        })
      } else {
        await route.fulfill({ status: 405 })
      }
    })

    await setupAuth(page)
    await page.goto("/pipeline")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("运行管道")
  })

  test("renders three radio options", async ({ page }) => {
    await expect(page.locator("text=Daily Pipeline")).toBeVisible()
    await expect(page.locator("text=Full Pipeline")).toBeVisible()
    await expect(page.locator("text=DB Full Pipeline")).toBeVisible()

    const radios = page.locator('input[type="radio"]')
    await expect(radios).toHaveCount(3)
  })

  test("daily pipeline is selected by default", async ({ page }) => {
    const dailyRadio = page.locator('input[type="radio"][value="daily"]')
    await expect(dailyRadio).toBeChecked()
  })

  test("can select full pipeline", async ({ page }) => {
    await page.locator('input[type="radio"][value="full"]').click({ force: true })
    const fullRadio = page.locator('input[type="radio"][value="full"]')
    await expect(fullRadio).toBeChecked()
  })

  test("can select db_full pipeline", async ({ page }) => {
    await page.locator('input[type="radio"][value="db_full"]').click({ force: true })
    const dbFullRadio = page.locator('input[type="radio"][value="db_full"]')
    await expect(dbFullRadio).toBeChecked()
  })

  test("preview button is visible", async ({ page }) => {
    await expect(page.locator("text=查看预览")).toBeVisible()
  })

  test("clicking preview for daily shows result", async ({ page }) => {
    // Daily is already selected
    await page.locator("text=查看预览").click()

    await expect(page.locator("text=python -m baxi.pipeline --type daily")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=15 minutes")).toBeVisible()
    await expect(page.locator("text=daily").first()).toBeVisible()
  })

  test("switching to full and preview shows full result", async ({ page }) => {
    await page.locator('input[type="radio"][value="full"]').click({ force: true })
    await page.locator("text=查看预览").click()

    await expect(page.locator("text=--days 634")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=2 hours")).toBeVisible()
  })

  test("preview shows warnings when present", async ({ page }) => {
    await page.locator('input[type="radio"][value="full"]').click({ force: true })
    await page.locator("text=查看预览").click()

    await expect(page.locator("text=Full pipeline runs for 634 days")).toBeVisible({ timeout: 5000 })
  })

  test("preview shows required env vars", async ({ page }) => {
    await page.locator("text=查看预览").click()

    await expect(page.locator("text=DATABASE_URL").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=OPENAI_API_KEY")).toBeVisible()
  })

  test("preview shows command in pre element", async ({ page }) => {
    await page.locator("text=查看预览").click()

    await expect(page.locator("pre")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("pre")).toContainText("python -m baxi.pipeline")
  })

  test("shows manual execution hint", async ({ page }) => {
    await page.locator("text=查看预览").click()

    await expect(page.locator("text=管道执行需在服务器终端手动运行")).toBeVisible({ timeout: 5000 })
  })

  test("switching radio updates command on re-preview", async ({ page }) => {
    // First: daily
    await page.locator("text=查看预览").click()
    await expect(page.locator("pre")).toContainText("--type daily", { timeout: 5000 })

    // Switch to db_full
    await page.locator('input[type="radio"][value="db_full"]').click({ force: true })
    await page.locator("text=查看预览").click()
    await expect(page.locator("pre")).toContainText("--dimensional", { timeout: 5000 })
  })
})
