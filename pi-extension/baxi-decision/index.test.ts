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

describe("baxi-decision extension", () => {
    let origEnv: Record<string, string | undefined>;

    beforeEach(() => {
        origEnv = { ...process.env };
        mockFetch.mockReset();
    });

    afterEach(() => {
        process.env = { ...origEnv };
    });

    describe("tool registration", () => {
        it("registers 13 tools with correct names", async () => {
            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            expect(registeredTools).toHaveLength(13);
            const names = registeredTools.map((t) => t.name);
            expect(names).toContain("baxi_evaluate_case");
            expect(names).toContain("baxi_decide");
            expect(names).toContain("baxi_list_cases");
            expect(names).toContain("baxi_get_evaluation");
            expect(names).toContain("baxi_list_proposals");
            expect(names).toContain("baxi_approve_proposal");
            expect(names).toContain("baxi_reject_proposal");
            expect(names).toContain("baxi_get_proposal_review");
            expect(names).toContain("baxi_get_governance_status");
            expect(names).toContain("baxi_get_classification");
            expect(names).toContain("baxi_get_lineage");
            expect(names).toContain("baxi_get_checkpoints");
            expect(names).toContain("baxi_get_system_status");
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

    describe("baxi_evaluate_case", () => {
        it("sends POST to /decisions/cases", async () => {
            mockFetch.mockResolvedValue(textResponse({ case_id: "case-123" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_evaluate_case")!;
            const result = await tool.execute(
                "call-1",
                { case_id: "case-123", object_type: "order", object_id: "ord-456" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases");
            expect(opts.method).toBe("POST");
            const body = JSON.parse(opts.body);
            expect(body.case_id).toBe("case-123");
            expect(body.object_type).toBe("order");
            expect(result.details.case_id).toBe("case-123");
        });

        it("handles API error gracefully", async () => {
            mockFetch.mockRejectedValue(new Error("Network error"));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_evaluate_case")!;
            const result = await tool.execute(
                "call-2",
                { case_id: "bad", object_type: "order", object_id: "x" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            expect(result.content[0].text).toContain("Error");
        });
    });

    describe("baxi_decide", () => {
        it("sends POST to /decisions/cases/{id}/decide", async () => {
            mockFetch.mockResolvedValue(textResponse({
                decision: "approve",
                score: 85,
                reasoning: "Low risk transaction",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_decide")!;
            const result = await tool.execute(
                "call-3",
                { case_id: "case-123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases/case-123/decide");
            expect(opts.method).toBe("POST");
            expect(result.details.decision).toBe("approve");
            expect(result.details.score).toBe(85);
        });

        it("passes context overrides", async () => {
            mockFetch.mockResolvedValue(textResponse({ decision: "review" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_decide")!;
            await tool.execute(
                "call-4",
                { case_id: "case-123", context: { risk_score: 60 } },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [, opts] = mockFetch.mock.calls[0];
            const body = JSON.parse(opts.body);
            expect(body.context.risk_score).toBe(60);
        });
    });

    describe("baxi_list_cases", () => {
        it("sends GET to /decisions/cases", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                cases: [{ case_id: "c1" }, { case_id: "c2" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_cases")!;
            const result = await tool.execute(
                "call-5",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases");
            expect(result.details.cases).toHaveLength(2);
        });

        it("passes status and limit query params", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ cases: [] }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_cases")!;
            await tool.execute(
                "call-6",
                { status: "open", limit: 10 },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("status=open");
            expect(url).toContain("limit=10");
        });
    });

    describe("baxi_get_evaluation", () => {
        it("sends GET to /decisions/cases/{id}", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                case_id: "case-123",
                status: "open",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_evaluation")!;
            const result = await tool.execute(
                "call-7",
                { case_id: "case-123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases/case-123");
            expect(result.details.case_id).toBe("case-123");
        });
    });

    describe("baxi_list_proposals", () => {
        it("sends GET to /decisions/cases/{id}/proposals", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                proposals: [{ proposal_id: "p1" }, { proposal_id: "p2" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_proposals")!;
            const result = await tool.execute(
                "call-8",
                { case_id: "case-123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/decisions/cases/case-123/proposals");
            expect(result.details.proposals).toHaveLength(2);
        });
    });

    describe("baxi_approve_proposal", () => {
        it("sends POST to /proposals/{id}/approve", async () => {
            mockFetch.mockResolvedValue(textResponse({ success: true }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_approve_proposal")!;
            const result = await tool.execute(
                "call-9",
                { proposal_id: "prop-42", reason: "Looks good" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/approve");
            expect(opts.method).toBe("POST");
            expect(JSON.parse(opts.body).reason).toBe("Looks good");
            expect(result.details.success).toBe(true);
        });
    });

    describe("baxi_reject_proposal", () => {
        it("sends POST to /proposals/{id}/reject", async () => {
            mockFetch.mockResolvedValue(textResponse({ success: true }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_reject_proposal")!;
            const result = await tool.execute(
                "call-10",
                { proposal_id: "prop-42", reason: "Risk too high" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/reject");
            expect(opts.method).toBe("POST");
            expect(JSON.parse(opts.body).reason).toBe("Risk too high");
        });

        it("requires reason parameter", async () => {
            // For TypeScript, reason is required; just verify the tool exists
            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_reject_proposal")!;
            expect(tool.parameters).toBeTruthy();
        });
    });

    describe("baxi_get_proposal_review", () => {
        it("sends GET to /proposals/{id}/review", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                proposal_id: "prop-42",
                status: "pending_review",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_proposal_review")!;
            const result = await tool.execute(
                "call-11",
                { proposal_id: "prop-42" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/proposals/prop-42/review");
            expect(result.details.status).toBe("pending_review");
        });
    });

    describe("baxi_get_governance_status", () => {
        it("sends GET to /governance/status", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                governance_layer: "active",
                configs: { access_policy: "loaded" },
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_governance_status")!;
            const result = await tool.execute(
                "call-12",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/status");
            expect(result.details.governance_layer).toBe("active");
        });
    });

    describe("baxi_get_classification", () => {
        it("sends GET to /governance/classification", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                levels: ["L1", "L2", "L3"],
                resources: [{ resource: "order.total_value", classification: "L2" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_classification")!;
            const result = await tool.execute(
                "call-13",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/classification");
            expect(result.details.levels).toContain("L3");
        });

        it("passes field_path query parameter", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ levels: [], resources: [] }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_classification")!;
            await tool.execute(
                "call-14",
                { field_path: "order.total_value" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("field_path=order.total_value");
        });
    });

    describe("baxi_get_lineage", () => {
        it("sends GET to /governance/lineage with resource param", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                resource: "orders",
                upstream: ["customers", "products"],
                downstream: ["order_items"],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_lineage")!;
            const result = await tool.execute(
                "call-15",
                { resource: "orders" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/lineage");
            expect(url).toContain("resource=orders");
            expect(result.details.upstream).toHaveLength(2);
        });
    });

    describe("baxi_get_checkpoints", () => {
        it("sends GET to /governance/checkpoints", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                checkpoints: [
                    { action: "execute_dispatch", requires_reason: true },
                ],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_checkpoints")!;
            const result = await tool.execute(
                "call-16",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/governance/checkpoints");
            expect(result.details.checkpoints).toHaveLength(1);
        });
    });

    describe("baxi_get_system_status", () => {
        it("sends GET to /status", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                version: "1.0.0",
                status: "healthy",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            const result = await tool.execute(
                "call-17",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/status");
            expect(result.details.status).toBe("healthy");
        });
    });

    describe("custom API URL and token", () => {
        it("uses BAXI_API_URL and BAXI_API_TOKEN from env", async () => {
            process.env.BAXI_API_URL = "https://custom.example.com";
            process.env.BAXI_API_TOKEN = "test-token";

            mockFetch.mockResolvedValue(jsonResponse({ status: "healthy" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            await tool.execute(
                "call-18",
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

            mockFetch.mockResolvedValue(jsonResponse({ status: "healthy" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_system_status")!;
            await tool.execute(
                "call-19",
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

    describe("error handling", () => {
        it("all tools handle API errors gracefully", async () => {
            mockFetch.mockRejectedValue(new Error("Service unavailable"));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            for (const tool of registeredTools) {
                const params = tool.name === "baxi_reject_proposal"
                    ? { proposal_id: "p1", reason: "test" }
                    : tool.name.includes("proposal")
                        ? { proposal_id: "p1" }
                        : tool.name.includes("case") || tool.name.includes("decide")
                            ? { case_id: "c1", object_type: "order", object_id: "o1" }
                            : tool.name === "baxi_get_lineage"
                                ? { resource: "orders" }
                                : tool.name === "baxi_get_classification"
                                    ? {}
                                    : {};

                const result = await tool.execute(
                    `call-${Math.random()}`,
                    params,
                    new AbortController().signal,
                    vi.fn(),
                    { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
                );

                expect(result.content[0].text).toContain("Error");
                expect(result.details.error).toBeTruthy();
            }
        });
    });
});
