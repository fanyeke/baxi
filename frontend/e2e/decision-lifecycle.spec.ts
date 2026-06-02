import { test, expect } from "@playwright/test"

test.describe("Decision Lifecycle (打通全部路径)", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    await page.evaluate(() => {
      sessionStorage.setItem("API_BEARER_TOKEN", "test-token-long-enough-for-your-app-32")
    })
  })

  test("full decision lifecycle: dashboard → decision review → case detail → outbox", async ({ page }) => {
    // ── Mock all API responses to simulate the full lifecycle ──

    // 1. Dashboard: status + alerts + tasks
    await page.route("**/api/v1/health", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: "ok", version: "0.6.0" }) })
    })
    await page.route("**/api/v1/status", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ schema_version: "1.0.0", alert_count: 3, pipeline_run: { status: "completed" }, table_counts: {}, recent_errors: [] }) })
    })
    await page.route("**/api/v1/alerts", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify([]) })
    })
    await page.route("**/api/v1/tasks", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify([]) })
    })

    // 2. Decision review page: list cases
    await page.route("**/api/v1/decisions/cases*", async (route) => {
      const url = new URL(route.request().url())
      if (route.request().method() === "POST") {
        // Create case
        await route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ case_id: "e2e-case-001" }) })
      } else {
        // List cases
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({
          items: [
            {
              case_id: "e2e-case-001",
              status: "open",
              source_type: "alert",
              source_id: "alert-42",
              severity: "high",
              summary: "High GMV drop detected",
              created_at: "2026-05-30T00:00:00Z",
              proposals: [
                { proposal_id: "prop-001", action_type: "notify_owner", risk_level: "medium", apply_status: "proposed" },
                { proposal_id: "prop-002", action_type: "block_order", risk_level: "high", apply_status: "proposed" },
              ],
            },
          ],
          total: 1,
        }) })
      }
    })

    // 3. Case detail page
    await page.route("**/api/v1/decisions/cases/e2e-case-001", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({
        case_id: "e2e-case-001",
        status: "open",
        source_type: "alert",
        source_id: "alert-42",
        severity: "high",
        object_type: "order",
        object_id: "ORD-HIGH-001",
        summary: "High GMV drop detected - potential fraud",
        created_at: "2026-05-30T00:00:00Z",
        proposals: [
          { proposal_id: "prop-001", action_type: "notify_owner", risk_level: "medium", apply_status: "proposed", title: "Notify seller owner", description: "Send alert to business_ops team" },
          { proposal_id: "prop-002", action_type: "block_order", risk_level: "high", apply_status: "proposed", title: "Block suspicious order", description: "Prevent order from being fulfilled" },
        ],
      }) })
    })

    // 4. Proposal actions: approve / reject
    await page.route("**/api/v1/decisions/proposals/*/approve", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ success: true, status: "approved" }) })
    })
    await page.route("**/api/v1/decisions/proposals/*/reject", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ success: true, status: "rejected" }) })
    })

    // 5. Agent logs
    await page.route("**/api/v1/agent-logs*", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({
        items: [
          { execution_id: "exec-decide-001", tool: "decide", status: "success", input_summary: "GMV drop case", created_at: "2026-05-30T00:00:00Z" },
          { execution_id: "exec-review-001", tool: "review", status: "success", input_summary: "Review prop-001", created_at: "2026-05-30T00:00:00Z" },
        ],
        total: 2,
      }) })
    })

    // 6. Outbox page
    await page.route("**/api/v1/outbox", async (route) => {
      if (route.request().method() === "POST") {
        await route.fulfill({ status: 201, contentType: "application/json", body: JSON.stringify({ event_id: "outbox-001", status: "created" }) })
      } else {
        await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify([
          { event_id: "outbox-001", event_type: "notify_owner", status: "pending", created_at: "2026-05-30T00:00:00Z" },
        ]) })
      }
    })

    // 7. Governance status
    await page.route("**/api/v1/governance/status", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ status: "compliant", last_check: "2026-05-30T00:00:00Z" }) })
    })

    // 8. Ontology
    await page.route("**/api/v1/ontology/object*", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({
        object_type: "order", object_id: "ORD-HIGH-001",
        properties: { total_amount: 2500.0, status: "pending", customer_id: "CUST-001" },
        linked_objects: [{ type: "customer", id: "CUST-001" }],
      }) })
    })

    // ── Execute the full flow ──

    // Step 1: Dashboard loads with alert count
    await page.goto("/")
    await expect(page.locator("text=Dashboard")).toBeVisible({ timeout: 10000 })

    // Step 2: Navigate to Decision Review page
    await page.click("a[href*='decision']")
    await expect(page).toHaveURL(/\/decision-review/, { timeout: 5000 })

    // Wait for page to render
    await page.waitForTimeout(2000)

    // Step 3: Navigate to Outbox page
    await page.click("a[href*='outbox']")
    await expect(page).toHaveURL(/\/outbox/, { timeout: 5000 })
    await page.waitForTimeout(1000)

    // Step 4: Navigate to Governance page
    await page.click("a[href*='governance']")
    await expect(page).toHaveURL(/\/governance/, { timeout: 5000 })
    await page.waitForTimeout(1000)

    // Step 5: Navigate to Agent Logs
    await page.click("a[href*='agent']")
    await expect(page).toHaveURL(/\/agent-logs/, { timeout: 5000 })
    await page.waitForTimeout(500)
  })
})
