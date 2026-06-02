import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_CATALOG,
  MOCK_CLASSIFICATION,
  MOCK_MARKINGS,
  MOCK_LINEAGE,
  MOCK_CHECKPOINTS,
  MOCK_GOVERNANCE_HEALTH,
} from "../fixtures/api-mocks"

test.describe("Governance Tabs Flow", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/governance/catalog**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CATALOG),
      })
    })
    await page.route("**/api/v1/governance/classification**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CLASSIFICATION),
      })
    })
    await page.route("**/api/v1/governance/markings**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_MARKINGS),
      })
    })
    await page.route("**/api/v1/governance/lineage**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_LINEAGE),
      })
    })
    await page.route("**/api/v1/governance/checkpoints**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_CHECKPOINTS),
      })
    })
    await page.route("**/api/v1/governance/health**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_GOVERNANCE_HEALTH),
      })
    })

    await setupAuth(page)
  })

  test("navigate through all 5 governance tabs", async ({ page }) => {
    await page.goto("/governance")

    // 1. Catalog tab (default)
    await expect(page.locator("text=asset-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=orders").first()).toBeVisible()

    // 2. Switch to Classification
    await page.locator("text=分类与标记").click()
    await expect(page.locator("text=分类规则")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=sensitive").first()).toBeVisible()
    await expect(page.locator("text=mark-finance")).toBeVisible()

    // 3. Switch to Lineage
    await page.locator("text=血缘关系").click()
    await expect(page.locator("text=节点").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=daily_metrics").first()).toBeVisible()
    await expect(page.locator("text=边").first()).toBeVisible()

    // 4. Switch to Checkpoints
    await page.locator("text=检查点").click()
    await expect(page.locator("text=chk-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=order_creation")).toBeVisible()

    // 5. Switch to Health
    await page.locator("text=健康检查").click()
    await expect(page.locator("text=监控视图").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=mv-001")).toBeVisible()
    await expect(page.locator("text=hc-001")).toBeVisible()

    // 6. Verify summary stats are always visible
    await expect(page.locator("text=数据资产")).toBeVisible()
    await expect(page.locator("text=分类规则")).toBeVisible()
    await expect(page.locator("text=检查点")).toBeVisible()
  })

  test("tab switching does not lose data", async ({ page }) => {
    await page.goto("/governance")

    // Load catalog
    await expect(page.locator("text=asset-001").first()).toBeVisible({ timeout: 5000 })

    // Switch away and back
    await page.locator("text=分类与标记").click()
    await expect(page.locator("text=sensitive").first()).toBeVisible({ timeout: 5000 })

    await page.locator("text=数据目录").click()
    // Data should still be visible (from cache)
    await expect(page.locator("text=asset-001").first()).toBeVisible({ timeout: 5000 })
  })

  test("all tabs show data tables", async ({ page }) => {
    await page.goto("/governance")

    // Catalog: 1 table
    await expect(page.locator("table").first()).toBeVisible({ timeout: 5000 })

    // Classification: 2 tables (classifications + markings)
    await page.locator("text=分类与标记").click()
    await expect(page.locator("table").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("table").nth(1)).toBeVisible()

    // Lineage: 2 tables (nodes + edges)
    await page.locator("text=血缘关系").click()
    await expect(page.locator("table").first()).toBeVisible({ timeout: 5000 })

    // Checkpoints: 1 table
    await page.locator("text=检查点").click()
    await expect(page.locator("table").first()).toBeVisible({ timeout: 5000 })

    // Health: 2 tables (monitoring views + health checks)
    await page.locator("text=健康检查").click()
    await expect(page.locator("table").first()).toBeVisible({ timeout: 5000 })
  })
})
