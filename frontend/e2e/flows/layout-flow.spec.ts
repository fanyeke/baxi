import { test, expect } from "@playwright/test"
import {
  setupAuth,
  mockAllApis,
  HEALTH_RESPONSE,
} from "../fixtures/api-mocks"

test.describe("Layout Flow", () => {
  test.beforeEach(async ({ page }) => {
    await mockAllApis(page)
    await setupAuth(page)
  })

  test("sidebar shows all navigation links", async ({ page }) => {
    await page.goto("/")

    await expect(page.locator("aside")).toBeVisible({ timeout: 10000 })

    // Verify all nav items from Layout
    await expect(page.locator("aside").locator("text=总览")).toBeVisible()
    await expect(page.locator("aside").locator("text=告警中心")).toBeVisible()
    await expect(page.locator("aside").locator("text=任务中心")).toBeVisible()
    await expect(page.locator("aside").locator("text=Outbox 分发")).toBeVisible()
    await expect(page.locator("aside").locator("text=日志诊断")).toBeVisible()
    await expect(page.locator("aside").locator("text=飞书同步")).toBeVisible()
    await expect(page.locator("aside").locator("text=运行管道")).toBeVisible()
    await expect(page.locator("aside").locator("text=治理中心")).toBeVisible()
    await expect(page.locator("aside").locator("text=Agent 日志")).toBeVisible()
  })

  test("active nav link is highlighted on Dashboard", async ({ page }) => {
    await page.goto("/")

    // The active nav link should have the "bg-sidebar-accent" class
    const activeLink = page.locator("aside a").filter({ hasText: "总览" })
    await expect(activeLink).toBeVisible({ timeout: 10000 })
    await expect(activeLink).toHaveClass(/bg-sidebar-accent/)
    await expect(activeLink).toHaveClass(/font-medium/)
  })

  test("active nav link changes when navigating to Alerts", async ({ page }) => {
    await page.goto("/")

    await page.locator("aside").locator("text=告警中心").click()

    await expect(page).toHaveURL(/\/alerts/)
    const activeLink = page.locator("aside a").filter({ hasText: "告警中心" })
    await expect(activeLink).toHaveClass(/bg-sidebar-accent/)
    await expect(activeLink).toHaveClass(/font-medium/)
  })

  test("active nav link changes when navigating to Tasks", async ({ page }) => {
    await page.goto("/")

    await page.locator("aside").locator("text=任务中心").click()

    await expect(page).toHaveURL(/\/tasks/)
    const activeLink = page.locator("aside a").filter({ hasText: "任务中心" })
    await expect(activeLink).toHaveClass(/bg-sidebar-accent/)
  })

  test("token input is visible in sidebar", async ({ page }) => {
    await page.goto("/")

    await expect(page.locator("text=API Token")).toBeVisible({ timeout: 10000 })
    const tokenInput = page.locator('input[type="password"]')
    await expect(tokenInput).toBeVisible()
  })

  test("token input has current value from sessionStorage", async ({ page }) => {
    await page.goto("/")

    const tokenInput = page.locator('input[type="password"]')
    await expect(tokenInput).toHaveValue("test-token-long-enough-for-your-app-32", { timeout: 10000 })
  })

  test("changing token updates sessionStorage", async ({ page }) => {
    await page.goto("/")

    const tokenInput = page.locator('input[type="password"]')
    await expect(tokenInput).toHaveValue("test-token-long-enough-for-your-app-32", { timeout: 10000 })

    // Clear and type new token
    await tokenInput.clear()
    await tokenInput.fill("new-token-12345")

    // Verify sessionStorage was updated
    const storedToken = await page.evaluate(() => sessionStorage.getItem("API_BEARER_TOKEN"))
    expect(storedToken).toBe("new-token-12345")
  })

  test("sidebar has app title", async ({ page }) => {
    await page.goto("/")

    await expect(page.locator("aside").locator("text=Olist 决策中台")).toBeVisible({ timeout: 10000 })
  })

  test("navigating to different pages updates active state correctly", async ({ page }) => {
    await page.goto("/")

    // Start on Dashboard
    await expect(page.locator("aside a").filter({ hasText: "总览" })).toHaveClass(/bg-sidebar-accent/, { timeout: 10000 })

    // Navigate to Alerts
    await page.locator("aside").locator("text=告警中心").click()
    await expect(page).toHaveURL(/\/alerts/)
    await expect(page.locator("aside a").filter({ hasText: "告警中心" })).toHaveClass(/bg-sidebar-accent/)
    await expect(page.locator("aside a").filter({ hasText: "总览" })).not.toHaveClass(/bg-sidebar-accent/)

    // Navigate to Outbox
    await page.locator("aside").locator("text=Outbox 分发").click()
    await expect(page).toHaveURL(/\/outbox/)
    await expect(page.locator("aside a").filter({ hasText: "Outbox 分发" })).toHaveClass(/bg-sidebar-accent/)
  })

  test("main content area renders page content", async ({ page }) => {
    await page.goto("/")

    // Main content should show Dashboard content
    await expect(page.locator("main")).toBeVisible({ timeout: 10000 })
    await expect(page.locator("main h1")).toBeVisible()
  })

  test("unknown route redirects to dashboard", async ({ page }) => {
    await page.goto("/nonexistent-page")

    await expect(page).toHaveURL("/", { timeout: 10000 })
  })
})
