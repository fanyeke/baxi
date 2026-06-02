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
  // Tool: baxi_create_sandbox
  pi.registerTool({
    name: "baxi_create_sandbox",
    label: "Create Proposal Sandbox",
    description: "Create a sandbox for testing governance proposals before applying them",
    parameters: Type.Object({
      case_id: Type.String({ description: "Decision case ID to create sandbox for" }),
      data: Type.Optional(Type.Any({ description: "Optional initial sandbox data" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const body: Record<string, any> = { case_id: params.case_id };
        if (params.data) body.data = params.data;
        const result = await apiPost("/sandboxes", body);
        ctx.ui.notify(`Created sandbox ${result.sandbox_id}`, "info");
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

  // Tool: baxi_compare_sandboxes
  pi.registerTool({
    name: "baxi_compare_sandboxes",
    label: "Compare Sandboxes",
    description: "Compare two sandboxes and return structured differences",
    parameters: Type.Object({
      sandbox_id_1: Type.String({ description: "First sandbox ID" }),
      sandbox_id_2: Type.String({ description: "Second sandbox ID" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const query = new URLSearchParams({
          sandbox_id_1: params.sandbox_id_1,
          sandbox_id_2: params.sandbox_id_2,
        });
        const result = await apiGet(`/sandboxes/compare?${query.toString()}`);
        const diffCount = result.differences?.length ?? 0;
        ctx.ui.notify(`Compared sandboxes: ${diffCount} difference(s) found`, "info");
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

  // Tool: baxi_get_sandbox
  pi.registerTool({
    name: "baxi_get_sandbox",
    label: "Get Sandbox",
    description: "Get details of a specific sandbox by ID",
    parameters: Type.Object({
      sandbox_id: Type.String({ description: "Sandbox ID to retrieve" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet(`/sandboxes/${params.sandbox_id}`);
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

  // Tool: baxi_list_sandboxes
  pi.registerTool({
    name: "baxi_list_sandboxes",
    label: "List Sandboxes",
    description: "List all existing proposal sandboxes",
    parameters: Type.Object({}),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet("/sandboxes");
        const count = result.items?.length ?? 0;
        ctx.ui.notify(`Found ${count} sandbox(es)`, "info");
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

  // Tool: baxi_describe_object
  pi.registerTool({
    name: "baxi_describe_object",
    label: "Describe Ontology Object",
    description: "Describe an ontology object type and its properties/relationships",
    parameters: Type.Object({
      object_type: Type.String({ description: "Ontology object type (e.g. seller, order, category)" }),
      object_id: Type.String({ description: "Specific object ID to describe" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiGet(`/ontology/object/${params.object_type}/${params.object_id}`);
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

  // Tool: baxi_get_linked_objects
  pi.registerTool({
    name: "baxi_get_linked_objects",
    label: "Get Linked Objects",
    description: "Get objects linked to an ontology object via a relationship",
    parameters: Type.Object({
      object_type: Type.String({ description: "Source object type" }),
      object_id: Type.String({ description: "Source object ID" }),
      link_name: Type.String({ description: "Relationship name (e.g. orders, reviews)" }),
      max_depth: Type.Optional(Type.Number({ description: "Max traversal depth (default 1)" })),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const query = new URLSearchParams();
        if (params.max_depth) query.set("max_depth", String(params.max_depth));
        const result = await apiGet(
          `/ontology/object/${params.object_type}/${params.object_id}/links/${params.link_name}?${query.toString()}`
        );
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

  // Tool: baxi_add_to_sandbox
  pi.registerTool({
    name: "baxi_add_to_sandbox",
    label: "Add Proposal to Sandbox",
    description: "Add an action proposal to an existing sandbox for comparison",
    parameters: Type.Object({
      sandbox_id: Type.String({ description: "Sandbox ID to add to" }),
      proposal_id: Type.String({ description: "Proposal ID to add" }),
    }),
    async execute(toolCallId, params, signal, onUpdate, ctx) {
      try {
        const result = await apiPost(`/sandboxes/${params.sandbox_id}/proposals`, {
          proposal_id: params.proposal_id,
        });
        ctx.ui.notify(`Added proposal ${params.proposal_id} to sandbox ${params.sandbox_id}`, "info");
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
