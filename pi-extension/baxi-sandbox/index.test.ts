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

describe("baxi-sandbox extension", () => {
    let origEnv: Record<string, string | undefined>;

    beforeEach(() => {
        origEnv = { ...process.env };
        mockFetch.mockReset();
    });

    afterEach(() => {
        process.env = { ...origEnv };
    });

    describe("tool registration", () => {
        it("registers 7 tools with correct names", async () => {
            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            expect(registeredTools).toHaveLength(7);
            const names = registeredTools.map((t) => t.name);
            expect(names).toContain("baxi_create_sandbox");
            expect(names).toContain("baxi_compare_sandboxes");
            expect(names).toContain("baxi_get_sandbox");
            expect(names).toContain("baxi_list_sandboxes");
            expect(names).toContain("baxi_describe_object");
            expect(names).toContain("baxi_get_linked_objects");
            expect(names).toContain("baxi_add_to_sandbox");
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

    describe("baxi_create_sandbox", () => {
        it("sends POST to /sandboxes with case_id", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ sandbox_id: "sb-1" }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_create_sandbox")!;
            const result = await tool.execute(
                "call-1",
                { case_id: "case-123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/sandboxes");
            expect(opts.method).toBe("POST");
            expect(JSON.parse(opts.body)).toEqual({ case_id: "case-123" });
            expect(result.details.sandbox_id).toBe("sb-1");
        });

        it("handles API error gracefully", async () => {
            mockFetch.mockRejectedValue(new Error("Network error"));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_create_sandbox")!;
            const result = await tool.execute(
                "call-2",
                { case_id: "bad" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            expect(result.content[0].text).toContain("Error");
        });
    });

    describe("baxi_compare_sandboxes", () => {
        it("sends GET to /sandboxes/compare with query params", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                sandbox_1_id: "sb-1",
                sandbox_2_id: "sb-2",
                differences: [
                    { field: "risk_score", value_1: 45, value_2: 60 },
                ],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_compare_sandboxes")!;
            const result = await tool.execute(
                "call-3",
                { sandbox_id_1: "sb-1", sandbox_id_2: "sb-2" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/sandboxes/compare?");
            expect(url).toContain("sandbox_id_1=sb-1");
            expect(url).toContain("sandbox_id_2=sb-2");
            expect(result.details.differences).toHaveLength(1);
        });

        it("handles error gracefully", async () => {
            mockFetch.mockRejectedValue(new Error("Not found"));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_compare_sandboxes")!;
            const result = await tool.execute(
                "call-4",
                { sandbox_id_1: "x", sandbox_id_2: "y" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            expect(result.content[0].text).toContain("Error");
        });
    });

    describe("baxi_get_sandbox", () => {
        it("sends GET to /sandboxes/{id}", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                sandbox_id: "sb-1",
                case_id: "case-123",
                status: "active",
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_sandbox")!;
            const result = await tool.execute(
                "call-5",
                { sandbox_id: "sb-1" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/sandboxes/sb-1");
            expect(opts.method).toBeUndefined();
            expect(result.details.sandbox_id).toBe("sb-1");
        });
    });

    describe("baxi_list_sandboxes", () => {
        it("sends GET to /sandboxes", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                items: [{ sandbox_id: "sb-1" }, { sandbox_id: "sb-2" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_sandboxes")!;
            const result = await tool.execute(
                "call-6",
                {},
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/sandboxes");
            expect(result.details.items).toHaveLength(2);
        });
    });

    describe("baxi_describe_object", () => {
        it("sends GET to /ontology/object/{type}/{id}", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                object_type: "seller",
                object_id: "shop_123",
                properties: { name: "Test Shop" },
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_describe_object")!;
            const result = await tool.execute(
                "call-7",
                { object_type: "seller", object_id: "shop_123" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/ontology/object/seller/shop_123");
            expect(result.details.object_type).toBe("seller");
        });
    });

    describe("baxi_get_linked_objects", () => {
        it("sends GET to /ontology/object/{type}/{id}/links/{link}", async () => {
            mockFetch.mockResolvedValue(jsonResponse({
                object_type: "seller",
                object_id: "shop_123",
                link_name: "orders",
                linked_objects: [{ order_id: "ord-1" }],
            }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_linked_objects")!;
            const result = await tool.execute(
                "call-8",
                { object_type: "seller", object_id: "shop_123", link_name: "orders" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/ontology/object/seller/shop_123/links/orders");
            expect(result.details.link_name).toBe("orders");
        });

        it("passes max_depth query parameter", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ linked_objects: [] }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_get_linked_objects")!;
            await tool.execute(
                "call-9",
                { object_type: "seller", object_id: "shop_123", link_name: "orders", max_depth: 2 },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("max_depth=2");
        });
    });

    describe("baxi_add_to_sandbox", () => {
        it("sends POST to /sandboxes/{id}/proposals", async () => {
            mockFetch.mockResolvedValue(jsonResponse({ success: true }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_add_to_sandbox")!;
            const result = await tool.execute(
                "call-10",
                { sandbox_id: "sb-1", proposal_id: "prop-42" },
                new AbortController().signal,
                vi.fn(),
                { ui: { notify: vi.fn(), setStatus: vi.fn() } } as any
            );

            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/sandboxes/sb-1/proposals");
            expect(opts.method).toBe("POST");
            expect(JSON.parse(opts.body)).toEqual({ proposal_id: "prop-42" });
            expect(result.details.success).toBe(true);
        });
    });

    describe("custom API URL and token", () => {
        it("uses BAXI_API_URL and BAXI_API_TOKEN from env", async () => {
            process.env.BAXI_API_URL = "https://custom.example.com";
            process.env.BAXI_API_TOKEN = "test-token";

            mockFetch.mockResolvedValue(jsonResponse({ items: [] }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_sandboxes")!;
            await tool.execute(
                "call-11",
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

            mockFetch.mockResolvedValue(jsonResponse({ items: [] }));

            const { pi, registeredTools } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const tool = registeredTools.find((t) => t.name === "baxi_list_sandboxes")!;
            await tool.execute(
                "call-12",
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
