import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockLayoutApis,
  MOCK_GOVERNANCE_STATUS,
} from "./fixtures/api-mocks"

test.describe("Policy Inspector page", () => {
  test.beforeEach(async ({ page }) => {
    await mockLayoutApis(page)
    await page.route("**/api/v1/governance/status**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(MOCK_GOVERNANCE_STATUS),
      })
    })

    await setupAuth(page)
    await page.goto("/policy-inspector")
  })

  test("renders page title", async ({ page }) => {
    await expect(page.locator("h1")).toContainText("策略检查器")
  })

  test("displays health badge", async ({ page }) => {
    await expect(page.locator("text=整体健康度")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=healthy").first()).toBeVisible()
  })

  test("health badge has green styling", async ({ page }) => {
    const badge = page.locator("span").filter({ hasText: "healthy" }).first()
    await expect(badge).toBeVisible()
    await expect(badge).toHaveClass(/bg-green-50/)
  })

  test("displays governance version", async ({ page }) => {
    await expect(page.locator("text=治理版本")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=2.1.0")).toBeVisible()
  })

  test("displays config versions section", async ({ page }) => {
    await expect(page.locator("text=配置版本")).toBeVisible({ timeout: 5000 })
  })

  test("shows all config badges as loaded", async ({ page }) => {
    await expect(page.locator("text=catalog: loaded").first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=classification: loaded")).toBeVisible()
    await expect(page.locator("text=markings: loaded")).toBeVisible()
    await expect(page.locator("text=lineage: loaded")).toBeVisible()
    await expect(page.locator("text=checkpoints: loaded")).toBeVisible()
  })

  test("config badges have green styling when loaded", async ({ page }) => {
    const badge = page.locator("span").filter({ hasText: "catalog: loaded" }).first()
    await expect(badge).toBeVisible()
    await expect(badge).toHaveClass(/bg-green-50/)
  })

  test("displays action allowlist section", async ({ page }) => {
    await expect(page.locator("text=操作白名单")).toBeVisible({ timeout: 5000 })
  })

  test("allowlist table shows configs", async ({ page }) => {
    const table = page.locator("table").last()
    await expect(table.locator("text=catalog").first()).toBeVisible({ timeout: 5000 })
    await expect(table.locator("text=loaded").first()).toBeVisible()
  })

  test("shows degraded health styling when degraded", async ({ page }) => {
    await page.route("**/api/v1/governance/status**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ...MOCK_GOVERNANCE_STATUS, overall_health: "degraded" }),
      })
    })

    await page.goto("/policy-inspector")
    const badge = page.locator("span").filter({ hasText: "degraded" }).first()
    await expect(badge).toBeVisible({ timeout: 5000 })
    await expect(badge).toHaveClass(/bg-yellow-50/)
  })

  test("shows unhealthy styling when unhealthy", async ({ page }) => {
    await page.route("**/api/v1/governance/status**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ...MOCK_GOVERNANCE_STATUS, overall_health: "unhealthy" }),
      })
    })

    await page.goto("/policy-inspector")
    const badge = page.locator("span").filter({ hasText: "unhealthy" }).first()
    await expect(badge).toBeVisible({ timeout: 5000 })
    await expect(badge).toHaveClass(/bg-red-50/)
  })

  test("shows error styling for error config status", async ({ page }) => {
    await page.route("**/api/v1/governance/status**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          ...MOCK_GOVERNANCE_STATUS,
          configs: { catalog: "loaded", classification: "error" },
        }),
      })
    })

    await page.goto("/policy-inspector")
    await expect(page.locator("text=classification: error").first()).toBeVisible({ timeout: 5000 })
    const badge = page.locator("span").filter({ hasText: "classification: error" }).first()
    await expect(badge).toHaveClass(/bg-red-50/)
  })

  test("shows empty state for no configs", async ({ page }) => {
    await page.route("**/api/v1/governance/status**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ...MOCK_GOVERNANCE_STATUS, configs: {} }),
      })
    })

    await page.goto("/policy-inspector")
    await expect(page.locator("text=暂无配置数据")).toBeVisible({ timeout: 5000 })
    await expect(page.locator("text=暂无操作白名单数据")).toBeVisible()
  })
})
