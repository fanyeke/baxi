import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

const API_URL = process.env.BAXI_API_URL || "http://localhost:8080";
const API_TOKEN = process.env.BAXI_API_TOKEN || "";

function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (API_TOKEN) headers["Authorization"] = `Bearer ${API_TOKEN}`;
  return headers;
}

async function apiGet(path: string): Promise<any> {
  const res = await fetch(`${API_URL}/api/v1${path}`, { headers: authHeaders() });
  if (!res.ok) throw new Error(`GET ${path} failed: ${res.status} ${res.statusText}`);
  return res.json();
}

async function apiPost(path: string, body: unknown): Promise<any> {
  const res = await fetch(`${API_URL}/api/v1${path}`, {
    method: "POST",
    headers: authHeaders(),
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`POST ${path} failed: ${res.status} ${res.statusText}`);
  const text = await res.text();
  return text ? JSON.parse(text) : {};
}


export default function (pi: ExtensionAPI) {
  // ── Action Tools ──

  // Tool: baxi_get_decision_context
  pi.registerTool({
    name: "baxi_get_decision_context",
    label: "Get Decision Context",
    description: "Get the full decision context for a case, including alerts, ontology objects, and governance info",
    parameters: Type.Object({
      case_id: Type.String({ description: "The ID of the case to get context for" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiPost(`/decisions/cases/${params.case_id}/context`, {});
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_execute_proposal
  pi.registerTool({
    name: "baxi_execute_proposal",
    label: "Execute Proposal",
    description: "Execute an approved action proposal (default dry_run=true for safety)",
    parameters: Type.Object({
      proposal_id: Type.String({ description: "The ID of the proposal to execute" }),
      dry_run: Type.Optional(Type.Boolean({ description: "When true (default), simulate execution without side effects" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const body: Record<string, any> = {};
        body.dry_run = params.dry_run !== undefined ? params.dry_run : true;
        const result = await apiPost(`/proposals/${params.proposal_id}/execute`, body);
        const status = result.status || "completed";
        ctx.ui.notify(`Proposal ${params.proposal_id} executed (${status})`, "info");
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // ── Governance Tools ──

  // Tool: baxi_check_access
  pi.registerTool({
    name: "baxi_check_access",
    label: "Check Access",
    description: "Check if a role has access to perform an action on an object type",
    parameters: Type.Object({
      role: Type.String({ description: "The role to check access for (e.g. admin, analyst, viewer)" }),
      object_type: Type.String({ description: "The type of object to check access on (e.g. order, seller, category)" }),
      action: Type.String({ description: "The action to check (e.g. read, write, delete)" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const query = new URLSearchParams({
          role: params.role,
          object_type: params.object_type,
          action: params.action,
        });
        const result = await apiGet(`/governance/access?${query.toString()}`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_get_classification
  pi.registerTool({
    name: "baxi_get_classification",
    label: "Get Classification",
    description: "Get classification information for a field path (e.g. user.email → pii/high)",
    parameters: Type.Object({
      field_path: Type.String({ description: "The field path to get classification for (e.g. user.email)" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet(`/governance/classification?field_path=${encodeURIComponent(params.field_path)}`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // ── Outbox Tools ──

  // Tool: baxi_list_outbox_events
  pi.registerTool({
    name: "baxi_list_outbox_events",
    label: "List Outbox Events",
    description: "List outbox events with optional status filter and pagination",
    parameters: Type.Object({
      status: Type.Optional(Type.String({ description: "Filter by status (e.g. pending, dispatched, failed)" })),
      limit: Type.Optional(Type.Number({ description: "Max results (default 20)" })),
      offset: Type.Optional(Type.Number({ description: "Pagination offset (default 0)" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const query = new URLSearchParams();
        if (params.status) query.set("status", params.status);
        query.set("limit", String(params.limit ?? 20));
        query.set("offset", String(params.offset ?? 0));
        const result = await apiGet(`/outbox?${query.toString()}`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_get_pipeline_status
  pi.registerTool({
    name: "baxi_get_pipeline_status",
    label: "Get Pipeline Status",
    description: "Get pipeline status including last run info and recent runs",
    parameters: Type.Object({}),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet(`/pipeline/status`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // ── Pipeline Tools ──

  // Tool: baxi_process_data
  pi.registerTool({
    name: "baxi_process_data",
    label: "Process Data",
    description: "Process data through the data pipeline with the specified configuration",
    parameters: Type.Object({
      config: Type.String({ description: "The pipeline configuration name or path" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiPost(`/pipeline/run`, { config: params.config });
        ctx.ui.notify(`Pipeline started with config "${params.config}"`, "info");
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // ── Review Tools ──

  // Tool: baxi_approve_proposal
  pi.registerTool({
    name: "baxi_approve_proposal",
    label: "Approve Proposal",
    description: "Approve an action proposal",
    parameters: Type.Object({
      proposal_id: Type.String({ description: "The ID of the proposal to approve" }),
      reviewer_id: Type.String({ description: "The ID of the reviewer" }),
      feedback: Type.Optional(Type.String({ description: "Optional feedback for the approval" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const body: Record<string, any> = { reviewer_id: params.reviewer_id };
        if (params.feedback) body.feedback = params.feedback;
        const result = await apiPost(`/proposals/${params.proposal_id}/approve`, body);
        ctx.ui.notify(`Proposal ${params.proposal_id} approved`, "success");
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_reject_proposal
  pi.registerTool({
    name: "baxi_reject_proposal",
    label: "Reject Proposal",
    description: "Reject an action proposal",
    parameters: Type.Object({
      proposal_id: Type.String({ description: "The ID of the proposal to reject" }),
      reviewer_id: Type.String({ description: "The ID of the reviewer" }),
      feedback: Type.Optional(Type.String({ description: "Optional feedback for the rejection" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const body: Record<string, any> = { reviewer_id: params.reviewer_id };
        if (params.feedback) body.feedback = params.feedback;
        const result = await apiPost(`/proposals/${params.proposal_id}/reject`, body);
        ctx.ui.notify(`Proposal ${params.proposal_id} rejected`, "info");
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_cancel_proposal
  pi.registerTool({
    name: "baxi_cancel_proposal",
    label: "Cancel Proposal",
    description: "Cancel an action proposal",
    parameters: Type.Object({
      proposal_id: Type.String({ description: "The ID of the proposal to cancel" }),
      reason: Type.Optional(Type.String({ description: "Optional reason for the cancellation" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const body: Record<string, any> = {};
        if (params.reason) body.reason = params.reason;
        const result = await apiPost(`/proposals/${params.proposal_id}/cancel`, body);
        ctx.ui.notify(`Proposal ${params.proposal_id} cancelled`, "info");
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // ── Status Tools ──

  // Tool: baxi_get_system_status
  pi.registerTool({
    name: "baxi_get_system_status",
    label: "Get System Status",
    description: "Get the current system status including alert counts, pipeline state, and table row counts",
    parameters: Type.Object({}),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet(`/status`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });

  // Tool: baxi_search_objects
  pi.registerTool({
    name: "baxi_search_objects",
    label: "Search Objects",
    description: "Search for ontology objects by type and query string",
    parameters: Type.Object({
      object_type: Type.String({ description: "The type of object to search for (e.g. order, seller, category)" }),
      query: Type.String({ description: "The search query string" }),
      limit: Type.Optional(Type.Number({ description: "Max results (default 20)" })),
      offset: Type.Optional(Type.Number({ description: "Pagination offset (default 0)" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const query = new URLSearchParams({
          object_type: params.object_type,
          query: params.query,
          limit: String(params.limit ?? 20),
          offset: String(params.offset ?? 0),
        });
        const result = await apiGet(`/search?${query.toString()}`);
        return {
          content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
          details: result,
        };
      } catch (err: any) {
        return {
          content: [{ type: "text", text: `Error: ${err.message}` }],
          details: { error: err.message },
        };
      }
    },
  });
}
