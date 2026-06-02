import { test, expect } from "@playwright/test"

test.describe("Dashboard", () => {
  test.beforeEach(async ({ page }) => {
    // Set auth token in sessionStorage before navigation
    await page.goto("/")
    await page.evaluate(() => {
      sessionStorage.setItem("API_BEARER_TOKEN", "test-token-long-enough-for-your-app-32")
    })

    // Mock API endpoints that Dashboard and Layout call
    await page.route("**/api/v1/health", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ status: "ok", version: "0.6.0" }),
      })
    })
    await page.route("**/api/v1/status", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          schema_version: "1.0.0",
          migration_version: "5",
          alert_count: 12,
          pipeline_run: { status: "completed", last_run: "2026-05-30T00:00:00Z" },
          table_counts: {
            ops_metric_alert: 12,
            audit_pipeline_run: 5,
            mart_metric_daily: 1500,
          },
          recent_errors: [],
        }),
      })
    })
  })

  test("renders dashboard title and status cards", async ({ page }) => {
    await page.goto("/")
    await expect(page).toHaveURL("/")

    // Sidebar navigation should be visible
    await expect(page.locator("text=Dashboard")).toBeVisible({ timeout: 10000 })

    // Status cards should render with data
    await expect(page.locator("text=12").first()).toBeVisible({ timeout: 5000 })
  })

  test("navigates to all main pages and renders content", async ({ page }) => {
    // Mock other API endpoints
    await page.route("**/api/v1/alerts", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([
          { alert_id: "alert-1", severity: "high", rule_id: "gmv_drop", status: "new", created_at: "2026-05-30T00:00:00Z" },
          { alert_id: "alert-2", severity: "medium", rule_id: "late_delivery_spike", status: "new", created_at: "2026-05-30T00:00:00Z" },
        ]),
      })
    })
    await page.route("**/api/v1/tasks", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([
          { task_id: "task-1", task_type: "investigate", status: "pending", created_at: "2026-05-30T00:00:00Z" },
        ]),
      })
    })

    // Navigate to alerts
    await page.click("text=Alerts")
    await expect(page).toHaveURL(/\/alerts/)
    await expect(page.locator("text=alert-1")).toBeVisible({ timeout: 5000 })

    // Navigate to tasks
    await page.click("text=Tasks")
    await expect(page).toHaveURL(/\/tasks/)
    await expect(page.locator("text=task-1")).toBeVisible({ timeout: 5000 })

    // Navigate to agent logs
    await page.route("**/api/v1/agent-logs*", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          items: [
            { execution_id: "exec-1", tool: "decide", status: "success", created_at: "2026-05-30T00:00:00Z" },
          ],
          total: 1,
        }),
      })
    })
    await page.click("text=Agent Logs")
    await expect(page).toHaveURL(/\/agent-logs/)
    await expect(page.locator("text=exec-1")).toBeVisible({ timeout: 5000 })
  })

  test("shows error state when API fails", async ({ page }) => {
    // Override the health/status routes to fail
    await page.route("**/api/v1/status", async (route) => {
      await route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({
          error_code: "INTERNAL_ERROR",
          message: "Server error",
          diagnosis: "Internal server error",
          suggested_action: "Try again later",
          request_id: "req-001",
        }),
      })
    })

    await page.goto("/")
    // Should show error message or empty state gracefully
    await expect(page.locator("text=Dashboard")).toBeVisible({ timeout: 10000 })
  })
})
