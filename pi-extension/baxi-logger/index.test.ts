import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createMockExtensionAPI } from "../test/mock.js";

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

async function loadExtension() {
    vi.resetModules();
    const mod = await import("./index.js");
    return mod.default;
}

describe("baxi-logger extension", () => {
    let origEnv: Record<string, string | undefined>;

    beforeEach(() => {
        origEnv = { ...process.env };
        mockFetch.mockReset();
    });

    afterEach(() => {
        process.env = { ...origEnv };
    });

    describe("event registration", () => {
        it("registers tool_execution_end event handler", async () => {
            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            expect(registeredEvents).toHaveLength(1);
            expect(registeredEvents[0].event).toBe("tool_execution_end");
        });
    });

    describe("logging execution", () => {
        it("sends a POST request to /logs/agent with execution data", async () => {
            mockFetch.mockResolvedValue({ ok: true, status: 200 });

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "baxi_create_case",
                args: { alert_id: "alert-123" },
                result: { decision_case_id: "case-123" },
                isError: false,
                error: null,
                durationMs: 150,
            };

            const ctx = {
                sessionId: "session-abc",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            await registeredEvents[0].handler(event, ctx as any);

            expect(mockFetch).toHaveBeenCalledTimes(1);
            const [url, opts] = mockFetch.mock.calls[0];
            expect(url).toContain("/api/v1/logs/agent");
            expect(opts.method).toBe("POST");

            const body = JSON.parse(opts.body);
            expect(body.tool_name).toBe("baxi_create_case");
            expect(body.session_id).toBe("session-abc");
            expect(body.status).toBe("success");
            expect(body.duration_ms).toBe(150);
        });

        it("sends error status when execution failed", async () => {
            mockFetch.mockResolvedValue({ ok: true, status: 200 });

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "baxi_list_cases",
                args: {},
                result: null,
                isError: true,
                error: "Something went wrong",
                durationMs: null,
            };

            const ctx = {
                sessionId: "session-xyz",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            await registeredEvents[0].handler(event, ctx as any);

            const [, opts] = mockFetch.mock.calls[0];
            const body = JSON.parse(opts.body);
            expect(body.status).toBe("error");
            expect(body.error_message).toBe("Something went wrong");
            expect(body.duration_ms).toBeNull();
        });

        it("includes auth header when BAXI_API_TOKEN is set", async () => {
            process.env.BAXI_API_TOKEN = "my-secret-token";
            mockFetch.mockResolvedValue({ ok: true, status: 200 });

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "baxi_decide",
                args: { case_id: "case-1" },
                result: { decision_id: "dec-1" },
                isError: false,
                error: null,
                durationMs: 100,
            };

            const ctx = {
                sessionId: "session-sec",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            await registeredEvents[0].handler(event, ctx as any);

            const [, opts] = mockFetch.mock.calls[0];
            expect(opts.headers["Authorization"]).toBe("Bearer my-secret-token");
        });

        it("handles fetch failure silently (no throw)", async () => {
            mockFetch.mockRejectedValue(new Error("Network error"));

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "test_tool",
                args: {},
                result: {},
                isError: false,
                error: null,
                durationMs: 0,
            };

            const ctx = {
                sessionId: "session-fail",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            // Should not throw
            await expect(
                registeredEvents[0].handler(event, ctx as any)
            ).resolves.toBeUndefined();
        });

        it("uses default API URL when BAXI_API_URL not set", async () => {
            delete process.env.BAXI_API_URL;
            mockFetch.mockResolvedValue({ ok: true, status: 200 });

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "test_tool",
                args: {},
                result: {},
                isError: false,
                error: null,
                durationMs: 0,
            };

            const ctx = {
                sessionId: "session-abc",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            await registeredEvents[0].handler(event, ctx as any);

            const [url] = mockFetch.mock.calls[0];
            expect(url).toContain("http://localhost:8080");
        });
    });

    describe("execution_id generation", () => {
        it("generates a unique execution_id", async () => {
            mockFetch.mockResolvedValue({ ok: true, status: 200 });

            const { pi, registeredEvents } = createMockExtensionAPI();
            const ext = await loadExtension();
            ext(pi);

            const event = {
                toolName: "test_tool",
                args: {},
                result: {},
                isError: false,
                error: null,
                durationMs: 0,
            };

            const ctx = {
                sessionId: "session-abc",
                ui: { notify: vi.fn(), setStatus: vi.fn() },
            };

            await registeredEvents[0].handler(event, ctx as any);

            const [, opts] = mockFetch.mock.calls[0];
            const body = JSON.parse(opts.body);
            expect(body.execution_id).toBeDefined();
            expect(typeof body.execution_id).toBe("string");
            // Should have timestamp prefix (13 digits + dash)
            expect(body.execution_id).toMatch(/^\d+-/);
        });
    });
});
