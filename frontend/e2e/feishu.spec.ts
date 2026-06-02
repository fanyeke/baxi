import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_FEISHU_EXPORT,
  MOCK_FEISHU_SYNC,
  MOCK_FEISHU_IMPORT,
} from "./fixtures/api-mocks"

test.describe("Feishu page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/feishu/export", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_FEISHU_EXPORT),
        })
      } else {
        await route.fulfill({ status: 405 })
      }
    })
    await page.route("**/api/v1/feishu/sync", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_FEISHU_SYNC),
        })
      } else {
        await route.fulfill({ status: 405 })
      }
    })
    await page.route("**/api/v1/feishu/status/import", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(MOCK_FEISHU_IMPORT),
        })
      } else {
        await route.fulfill({ status: 405 })
      }
    })

    await setupAuth(page)
    await page.goto("/feishu")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("飞书同步")
  })

  test("renders three sections: export, sync, import", async ({ page }) => {
    await expect(page.locator("h2").filter({ hasText: "导出" })).toBeVisible()
    await expect(page.locator("h2").filter({ hasText: "同步到飞书" })).toBeVisible()
    await expect(page.locator("h2").filter({ hasText: "状态导入" })).toBeVisible()
  })

  test("export section has dry-run and real export buttons", async ({ page }) => {
    const exportSection = page.locator("h2").filter({ hasText: "导出" }).locator("..")
    await expect(exportSection.locator("text=Dry-Run 导出")).toBeVisible()
    await expect(exportSection.locator("text=真实导出")).toBeVisible()
  })

  test("sync section has dry-run and real sync buttons", async ({ page }) => {
    const syncSection = page.locator("h2").filter({ hasText: "同步到飞书" }).locator("..")
    await expect(syncSection.locator("text=Dry-Run 同步")).toBeVisible()
    await expect(syncSection.locator("text=真实同步")).toBeVisible()
  })

  test("import section has dry-run import button", async ({ page }) => {
    const importSection = page.locator("h2").filter({ hasText: "状态导入" }).locator("..")
    await expect(importSection.locator("text=Dry-Run 导入")).toBeVisible()
  })

  test("dry-run export shows result status", async ({ page }) => {
    await page.locator("text=Dry-Run 导出").click()

    await expect(page.locator("text=导出:").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Exported 2 tables")).toBeVisible()
  })

  test("dry-run sync shows result status", async ({ page }) => {
    await page.locator("text=Dry-Run 同步").click()

    await expect(page.locator("text=同步:").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Synced 2 tables to feishu")).toBeVisible()
  })

  test("dry-run import shows result status", async ({ page }) => {
    await page.locator("text=Dry-Run 导入").click()

    await expect(page.locator("text=导入:").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Imported 0 status changes")).toBeVisible()
  })

  test("real export opens confirmation dialog", async ({ page }) => {
    await page.locator("text=真实导出").click()

    await expect(page.locator("text=确认导出？")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=确认导出文件到 data/feishu/？")).toBeVisible()
  })

  test("real sync opens confirmation dialog", async ({ page }) => {
    await page.locator("text=真实同步").click()

    await expect(page.locator("text=确认同步到飞书？")).toBeVisible({ timeout: 5000 })
  })

  test("export dialog confirm executes and closes", async ({ page }) => {
    await page.locator("text=真实导出").click()
    await expect(page.locator("text=确认导出？")).toBeVisible({ timeout: 5000 })

    await page.locator("text=确认执行").last().click()

    // Dialog should close and result should show
    await expect(page.locator("text=确认导出？")).not.toBeVisible()
    await expect(page.locator("text=导出:").first()).toBeVisible({ timeout: 5000 })
  })

  test("export dialog cancel closes dialog", async ({ page }) => {
    await page.locator("text=真实导出").click()
    await expect(page.locator("text=确认导出？")).toBeVisible({ timeout: 5000 })

    await page.locator("text=取消").click()
    await expect(page.locator("text=确认导出？")).not.toBeVisible()
  })

  test("import section shows hint text", async ({ page }) => {
    await expect(page.locator("text=状态修改建议通过飞书完成")).toBeVisible()
  })
})
