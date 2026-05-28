import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

export default function (pi: ExtensionAPI) {
    // Register decision tools
    pi.registerTool({
        name: "baxi_create_case",
        label: "Create Decision Case",
        description: "Create a decision case from a Baxi alert",
        parameters: Type.Object({
            alert_id: Type.String({ description: "Alert ID to create case from" }),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            // Use pi-mcp-adapter's mcp proxy tool via the conversation
            ctx.sendMessage(`Please use the mcp tool to call baxi_mcp with: create_decision_case with alert_id=${params.alert_id}`);
            return {
                content: [{ type: "text", text: "Attempting to create decision case..." }],
                details: {},
            };
        },
    });

    pi.registerTool({
        name: "baxi_decide",
        label: "Execute Decision",
        description: "Generate a decision for a case",
        parameters: Type.Object({
            case_id: Type.String({ description: "Decision case ID" }),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            ctx.sendMessage(`Please use the mcp tool to call decide with case_id=${params.case_id}`);
            return {
                content: [{ type: "text", text: "Attempting to run decision..." }],
                details: {},
            };
        },
    });

    pi.registerTool({
        name: "baxi_list_cases",
        label: "List Decision Cases",
        description: "List decision cases with optional status filter",
        parameters: Type.Object({
            status: Type.Optional(Type.String({ description: "Filter by status" })),
            limit: Type.Optional(Type.Number({ description: "Max results", default: 10 })),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            // Implementation via pi-mcp-adapter
            return {
                content: [{ type: "text", text: "Querying decision cases..." }],
                details: params,
            };
        },
    });

    pi.registerTool({
        name: "baxi_list_alerts",
        label: "List Alerts",
        description: "List alerts with optional severity filter",
        parameters: Type.Object({
            severity: Type.Optional(Type.String({ description: "Filter by severity", enum: ["high", "medium", "low"] })),
            limit: Type.Optional(Type.Number({ description: "Max results", default: 10 })),
        }),
        async execute(toolCallId, params, signal, onUpdate, ctx) {
            return {
                content: [{ type: "text", text: "Querying alerts..." }],
                details: params,
            };
        },
    });

    // Threshold trigger: auto-create decision cases for high-severity alerts
    pi.on("session_start", async (_event, ctx) => {
        ctx.ui.notify("Baxi decision monitor started", "info");
        startThresholdMonitor(ctx);
    });

    function startThresholdMonitor(ctx: any) {
        setInterval(async () => {
            ctx.ui.setStatus("baxi-monitor", "Checking alerts...");
            try {
                // Check for high-severity alerts
                // In real usage, this would call the baxi MCP server
                ctx.ui.setStatus("baxi-monitor", "Monitor active");
            } catch (err) {
                ctx.ui.notify(`Monitor error: ${err}`, "error");
            }
        }, 60000); // Every minute
    }
}
