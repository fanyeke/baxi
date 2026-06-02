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
} from "./fixtures/api-mocks"

test.describe("Governance page", () => {
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
    await page.goto("/governance")
  })

  test("renders page title and description", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("治理中心")
    await expect(page.locator("text=数据资产治理")).toBeVisible()
  })

  test("renders 5 tab buttons", async ({ page }) => {
    await expect(page.locator("text=数据目录")).toBeVisible()
    await expect(page.locator("text=分类与标记")).toBeVisible()
    await expect(page.locator("text=血缘关系")).toBeVisible()
    await expect(page.locator("text=检查点")).toBeVisible()
    await expect(page.locator("text=健康检查")).toBeVisible()
  })

  test("defaults to catalog tab with data", async ({ page }) => {
    await expect(page.locator("text=orders").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=asset-001")).toBeVisible()

    // Should show asset table
    const rows = page.locator("table tbody tr")
    await expect(rows).toHaveCount(2)
  })

  test("catalog tab shows asset details", async ({ page }) => {
    await expect(page.locator("text=public.orders")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Customer orders")).toBeVisible()
  })

  test("switches to classification tab", async ({ page }) => {
    await page.locator("text=分类与标记").click()

    await expect(page.locator("text=sensitive").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=分类规则")).toBeVisible()
    await expect(page.locator("text=标记策略")).toBeVisible()
  })

  test("classification tab shows classification data", async ({ page }) => {
    await page.locator("text=分类与标记").click()

    await expect(page.locator("text=asset-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=Contains PII data")).toBeVisible()
  })

  test("classification tab shows marking data", async ({ page }) => {
    await page.locator("text=分类与标记").click()

    await expect(page.locator("text=mark-finance").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=role_based")).toBeVisible()
    await expect(page.locator("text=finance_team_only")).toBeVisible()
  })

  test("switches to lineage tab", async ({ page }) => {
    await page.locator("text=血缘关系").click()

    await expect(page.locator("text=节点").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=边").first()).toBeVisible()

    // Should show node and edge tables
    const tables = page.locator("table")
    await expect(tables).toHaveCount(2)
  })

  test("lineage tab shows node details", async ({ page }) => {
    await page.locator("text=血缘关系").click()

    await expect(page.locator("text=daily_metrics").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=raw_events")).toBeVisible()
  })

  test("lineage tab shows edge details", async ({ page }) => {
    await page.locator("text=血缘关系").click()

    await expect(page.locator("text=daily_agg").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=aggregation")).toBeVisible()
  })

  test("switches to checkpoints tab", async ({ page }) => {
    await page.locator("text=检查点").click()

    await expect(page.locator("text=chk-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=order_creation")).toBeVisible()
  })

  test("checkpoints tab shows checkpoint details", async ({ page }) => {
    await page.locator("text=检查点").click()

    await expect(page.locator("text=Justify the order amount change")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=threshold, approval")).toBeVisible()
  })

  test("switches to health tab", async ({ page }) => {
    await page.locator("text=健康检查").click()

    await expect(page.locator("text=监控视图").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=健康检查").first()).toBeVisible()
  })

  test("health tab shows monitoring views and health checks", async ({ page }) => {
    await page.locator("text=健康检查").click()

    // Monitoring views
    await expect(page.locator("text=mv-001").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=data_freshness")).toBeVisible()

    // Health checks
    await expect(page.locator("text=hc-001")).toBeVisible()
    await expect(page.locator("text=database").first()).toBeVisible()
  })

  test("displays summary stats at bottom", async ({ page }) => {
    // Catalog has 2 assets, classification has 2, checkpoints 2, health has 1+2=3
    await expect(page.locator("text=数据资产")).toBeVisible()
    await expect(page.locator("text=分类规则")).toBeVisible()
    await expect(page.locator("text=检查点")).toBeVisible()
  })

  test("switching tabs updates visible content", async ({ page }) => {
    // Catalog is visible by default
    await expect(page.locator("text=orders").first()).toBeVisible({ timeout: 5000 })

    // Switch to lineage
    await page.locator("text=血缘关系").click()
    await expect(page.locator("text=节点").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=asset-001")).not.toBeVisible()

    // Switch back to catalog
    await page.locator("text=数据目录").click()
    await expect(page.locator("text=asset-001")).toBeVisible({ timeout: 5000 })
  })
})
