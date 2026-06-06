import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createMockExtensionAPI } from "../test/mock.js";

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

async function loadExtension() {
    vi.resetModules();
    const mod = await import("./index.js");
    return mod.default;
}

function jsonResponse(data: unknown) {
    return {
        ok: true,
        json: async () => data,
        text: async () => JSON.stringify(data),
    };
}

function textResponse(data: unknown) {
    return {
        ok: true,
        text: async () => JSON.stringify(data),
    };
}

describe("baxi-operations extension", () => {
    let origEnv: Record<string, string | undefined>;

    beforeEach(() => {
        origEnv = { ...process.env };
        mockFetch.mockReset();
    });

    afterEach(() => {
        process.env = { ...origEnv };
    });

    describe("tool registration", () => {
        it("registers 12 tools with correct names", async () => {
            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            expect(registeredTools).toHaveLength(12);
            const names = registeredTools.map((t) => t.name);
            expect(names).toContain("baxi_get_decision_context");
            expect(names).toContain("baxi_execute_proposal");
            expect(names).toContain("baxi_check_access");
            expect(names).toContain("baxi_get_classification");
            expect(names).toContain("baxi_list_outbox_events");
            expect(names).toContain("baxi_get_pipeline_status");
            expect(names).toContain("baxi_process_data");
            expect(names).toContain("baxi_approve_proposal");
            expect(names).toContain("baxi_reject_proposal");
            expect(names).toContain("baxi_cancel_proposal");
            expect(names).toContain("baxi_get_system_status");
            expect(names).toContain("baxi_search_objects");
        });

        it("tools have descriptions and parameters", async () => {
            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            for (const tool of registeredTools) {
                expect(tool.description).toBeTruthy();
                expect(tool.parameters).toBeTruthy();
            }
        });
    });

    describe("baxi_get_decision_context (Action)", () => {
        it("sends POST to /decisions/cases/:id/context", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                case_id: "case-123",
                alerts: [{ alert_id: "alert-1" }],
                ontology: { object_type: "order", object_id: "ORD-001" },
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_decision_context")!;
            const result = await tool.execute(
                "call-1",
                { case_id: "case-123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases/case-123/context");
            expect(opts.method).toBe("POST");
            expect(result.details.case_id).toBe("case-123");
            expect(result.details.ontology).toBeDefined();
        });
    });

    describe("baxi_execute_proposal (Action)", () => {
        it("sends POST to /proposals/:id/execute with dry_run default true", async () => {
            mockFetch.mockResolvedValue(textResponse({
                execution_id: "exec-1",
                status: "dry_run_completed",
                simulated_changes: [{ field: "risk_score", old: 45, new: 60 }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_execute_proposal")!;
            const result = await tool.execute(
                "call-2",
                { proposal_id: "prop-42" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/execute");
            expect(opts.method).toBe("POST");
            const body = JSON.parse(opts.body);
            expect(body.dry_run).toBe(true);
            expect(result.details.status).toBe("dry_run_completed");
        });

        it("passes dry_run=false when requested", async () => {
            mockFetch.mockResolvedValue(textResponse({
                execution_id: "exec-2",
                status: "executed",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_execute_proposal")!;
            await tool.execute(
                "call-3",
                { proposal_id: "prop-42", dry_run: false },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [, opts] = mockFetch.mock.calls[0];
            const body = JSON.parse(opts.body);
            expect(body.dry_run).toBe(false);
        });
    });

    describe("baxi_check_access (Governance)", () => {
        it("sends GET to /governance/access with query params", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                role: "admin",
                object_type: "order",
                action: "write",
                allowed: true,
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_check_access")!;
            const result = await tool.execute(
                "call-4",
                { role: "admin", object_type: "order", action: "write" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/access?");
            expect(url).toContain("role=admin");
            expect(url).toContain("object_type=order");
            expect(url).toContain("action=write");
            expect(result.details.allowed).toBe(true);
        });
    });

    describe("baxi_get_classification (Governance)", () => {
        it("sends GET to /governance/classification with query param", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                field_path: "user.email",
                classification: "pii",
                sensitivity: "high",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_classification")!;
            const result = await tool.execute(
                "call-5",
                { field_path: "user.email" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/classification?field_path=user.email");
            expect(result.details.classification).toBe("pii");
        });
    });

    describe("baxi_list_outbox_events (Outbox)", () => {
        it("sends GET to /outbox with query params", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                items: [
                    { event_id: "evt-1", event_type: "notify_owner", status: "pending" },
                ],
                total: 1,
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_outbox_events")!;
            const result = await tool.execute(
                "call-6",
                { status: "pending", limit: 10, offset: 0 },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/outbox?");
            expect(url).toContain("status=pending");
            expect(url).toContain("limit=10");
            expect(url).toContain("offset=0");
            expect(result.details.items).toHaveLength(1);
        });

        it("uses defaults when no params provided", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ items: [], total: 0 }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_outbox_events")!;
            await tool.execute(
                "call-7",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("limit=20");
            expect(url).toContain("offset=0");
        });
    });

    describe("baxi_get_pipeline_status (Outbox)", () => {
        it("sends GET to /pipeline/status", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                status: "completed",
                last_run: "2026-05-30T00:00:00Z",
                recent_runs: [{ run_id: "run-1", status: "completed" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_pipeline_status")!;
            const result = await tool.execute(
                "call-8",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/pipeline/status");
            expect(result.details.status).toBe("completed");
        });
    });

    describe("baxi_process_data (Pipeline)", () => {
        it("sends POST to /pipeline/run with config", async () => {
            mockFetch.mockResolvedValue(textResponse({
                result_id: "run-42",
                config: "default",
                status: "started",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_process_data")!;
            const result = await tool.execute(
                "call-9",
                { config: "default" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/pipeline/run");
            expect(opts.method).toBe("POST");
            expect(JSON.parse(opts.body)).toEqual({ config: "default" });
            expect(result.details.status).toBe("started");
        });
    });

    describe("baxi_approve_proposal (Review)", () => {
        it("sends POST to /proposals/:id/approve", async () => {
            mockFetch.mockResolvedValue(textResponse({
                success: true,
                status: "approved",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_approve_proposal")!;
            const result = await tool.execute(
                "call-10",
                { proposal_id: "prop-42", reviewer_id: "user-1", feedback: "Looks good" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/approve");
            expect(opts.method).toBe("POST");
            const body = JSON.parse(opts.body);
            expect(body.reviewer_id).toBe("user-1");
            expect(body.feedback).toBe("Looks good");
            expect(result.details.success).toBe(true);
        });
    });

    describe("baxi_reject_proposal (Review)", () => {
        it("sends POST to /proposals/:id/reject", async () => {
            mockFetch.mockResolvedValue(textResponse({
                success: true,
                status: "rejected",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_reject_proposal")!;
            const result = await tool.execute(
                "call-11",
                { proposal_id: "prop-42", reviewer_id: "user-1", feedback: "Needs revision" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/reject");
            expect(opts.method).toBe("POST");
            const body = JSON.parse(opts.body);
            expect(body.reviewer_id).toBe("user-1");
            expect(body.feedback).toBe("Needs revision");
            expect(result.details.success).toBe(true);
        });
    });

    describe("baxi_cancel_proposal (Review)", () => {
        it("sends POST to /proposals/:id/cancel", async () => {
            mockFetch.mockResolvedValue(textResponse({
                success: true,
                status: "cancelled",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_cancel_proposal")!;
            const result = await tool.execute(
                "call-12",
                { proposal_id: "prop-42", reason: "No longer needed" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/cancel");
            expect(opts.method).toBe("POST");
            const body = JSON.parse(opts.body);
            expect(body.reason).toBe("No longer needed");
            expect(result.details.success).toBe(true);
        });

        it("works without reason", async () => {
            mockFetch.mockResolvedValue(textResponse({ success: true, status: "cancelled" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_cancel_proposal")!;
            await tool.execute(
                "call-13",
                { proposal_id: "prop-42" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [, opts] = mockFetch.mock.calls[0];
            const body = JSON.parse(opts.body);
            expect(body.reason).toBeUndefined();
        });
    });

    describe("baxi_get_system_status (Status)", () => {
        it("sends GET to /status", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                alert_count: 12,
                schema_version: "1.0.0",
                pipeline_run: { status: "completed" },
                table_counts: { ops_metric_alert: 12 },
                recent_errors: [],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            const result = await tool.execute(
                "call-14",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/status");
            expect(result.details.alert_count).toBe(12);
        });
    });

    describe("baxi_search_objects (Status)", () => {
        it("sends GET to /search with query params", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                results: [
                    { object_type: "order", object_id: "ORD-001", summary: "Order #1001" },
                ],
                total: 1,
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_search_objects")!;
            const result = await tool.execute(
                "call-15",
                { object_type: "order", query: "1001", limit: 10, offset: 0 },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/search?");
            expect(url).toContain("object_type=order");
            expect(url).toContain("query=1001");
            expect(url).toContain("limit=10");
            expect(result.details.results).toHaveLength(1);
        });
    });

    describe("error handling", () => {
        const toolNames = [
            "baxi_get_decision_context",
            "baxi_execute_proposal",
            "baxi_check_access",
            "baxi_get_classification",
            "baxi_list_outbox_events",
            "baxi_get_pipeline_status",
            "baxi_process_data",
            "baxi_approve_proposal",
            "baxi_reject_proposal",
            "baxi_cancel_proposal",
            "baxi_get_system_status",
            "baxi_search_objects",
        ];

        for (const toolName of toolNames) {
            it(`handles network error for ${toolName} gracefully`, async () => {
                mockFetch.mockRejectedValue(new Error("Network error"));

                const { pi, registeredTools } = createMockExtensionAPI();
                const ext = await loadExtension();
                ext(pi);

                const tool = registeredTools.find((t) => t.name === toolName)!;
                const result = await tool.execute(
                    "call-error",
                    {},
                    new AbortController().signal,
                    vi.fn(),
                    { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
                );

                expect(result.content[0].text).toContain("Error");
                expect(result.details.error).toBeDefined();
            });
        }
    });

    describe("environment configuration", () => {
        it("uses BAXI_API_URL and BAXI_API_TOKEN from env", async () => {
            process.env.BAXI_API_URL = "https://custom.example.com";
            process.env.BAXI_API_TOKEN = "test-token";

            mockFetch.mockResolvedValue(jsonResponse({ status: "ok" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            await tool.execute(
                "call-16",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("https://custom.example.com");
            expect(opts.headers["Authorization"]).toBe("Bearer test-token");
        });

        it("uses default localhost URL when env not set", async () => {
            delete process.env.BAXI_API_URL;
            delete process.env.BAXI_API_TOKEN;

            mockFetch.mockResolvedValue(jsonResponse({ status: "ok" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            await tool.execute(
                "call-17",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("http://localhost:8080");
            expect(opts.headers["Authorization"]).toBeUndefined();
        });
    });
});
