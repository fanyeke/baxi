import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
    const API_URL = process.env.BAXI_API_URL || "http://localhost:8080";
    const API_TOKEN = process.env.BAXI_API_TOKEN || "";

    // Log all tool executions to Baxi
    pi.on("tool_execution_end", async (event, ctx) => {
        try {
            const headers: Record<string, string> = {
                "Content-Type": "application/json",
            };
            if (API_TOKEN) {
                headers["Authorization"] = `Bearer ${API_TOKEN}`;
            }

            await fetch(`${API_URL}/api/v1/logs/agent`, {
                method: "POST",
                headers,
                body: JSON.stringify({
                    execution_id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
                    session_id: ctx.sessionId,
                    tool_name: event.toolName,
                    input_args: event.args,
                    output_result: event.result,
                    status: event.isError ? "error" : "success",
                    error_message: event.error || null,
                    duration_ms: event.durationMs || null,
                }),
            });
        } catch (err) {
            // Silent fail - logging should not interrupt the main flow
            console.error("Failed to log to Baxi:", err);
        }
    });
}
