import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import * as Dialog from "@radix-ui/react-dialog"
import { apiClient } from "../api/client"
import type { Sandbox, ComparisonResult, ProposalListResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

function statusColor(status: string): string {
  switch (status) {
    case "resolved": return "bg-green-100 text-green-700"
    case "comparing": return "bg-blue-100 text-blue-700"
    case "draft": return "bg-yellow-100 text-yellow-700"
    default: return "bg-gray-100 text-gray-700"
  }
}

export default function SandboxCompare() {
  const queryClient = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [createCaseId, setCreateCaseId] = useState("")
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [addProposalSandboxId, setAddProposalSandboxId] = useState<string | null>(null)
  const [addProposalId, setAddProposalId] = useState("")

  const sandboxes = useQuery({
    queryKey: ["sandboxes"],
    queryFn: () => apiClient.get<{ items: Sandbox[] }>("/sandboxes"),
  })

  const comparison = useQuery({
    queryKey: ["sandbox-comparison", selectedIds[0], selectedIds[1]],
    queryKeyHashFn: () => `sandbox-comparison-${selectedIds[0]}-${selectedIds[1]}`,
    queryFn: () => apiClient.get<ComparisonResult>("/sandboxes/compare", {
      sandbox1_id: selectedIds[0],
      sandbox2_id: selectedIds[1],
    }),
    enabled: selectedIds.length === 2,
  })

  const caseProposals = useQuery({
    queryKey: ["case-proposals-for-sandbox", addProposalSandboxId],
    queryFn: () => apiClient.get<ProposalListResponse>(`/decisions/cases/${addProposalSandboxId}/proposals`),
    enabled: !!addProposalSandboxId,
  })

  const createMutation = useMutation({
    mutationFn: (caseId: string) =>
      apiClient.post("/sandboxes", { case_id: caseId, data: {} }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] })
      setShowCreate(false)
      setCreateCaseId("")
    },
  })

  const addProposalMutation = useMutation({
    mutationFn: (args: { sandboxId: string; proposalId: string }) =>
      apiClient.post(`/sandboxes/${args.sandboxId}/proposals`, { proposal_id: args.proposalId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["sandboxes"] })
      setAddProposalSandboxId(null)
      setAddProposalId("")
    },
  })

  function toggleSelect(id: string) {
    setSelectedIds(prev => {
      if (prev.includes(id)) return prev.filter(x => x !== id)
      if (prev.length >= 2) return [prev[1], id]
      return [...prev, id]
    })
  }

  const isLoading = sandboxes.isLoading
  const error = sandboxes.error

  return (
    <div className="space-y-4">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold">Sandbox</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Compare proposals in isolated sandbox environments
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="px-4 py-2 bg-primary text-primary-foreground rounded-md text-sm hover:bg-primary/90 transition-colors"
        >
          Create Sandbox
        </button>
      </div>

      {isLoading && <LoadingSkeleton type="table" count={5} />}
      {error && <ErrorPanel title="Failed to load" message={error.message || "Unknown error"} />}
      {!isLoading && !error && (!sandboxes.data?.items || sandboxes.data.items.length === 0) && (
        <EmptyState
          title="No sandboxes"
          description="Create a sandbox to start comparing proposals"
        />
      )}

      {!isLoading && !error && sandboxes.data && sandboxes.data.items.length > 0 && (
        <div className="space-y-4">
          <div className="border rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted">
                <tr>
                  <th className="p-2 text-left w-10" />
                  <th className="p-2 text-left">Sandbox ID</th>
                  <th className="p-2 text-left">Case ID</th>
                  <th className="p-2 text-left">Status</th>
                  <th className="p-2 text-left">Proposals</th>
                  <th className="p-2 text-left">Created</th>
                  <th className="p-2 text-left">Actions</th>
                </tr>
              </thead>
              <tbody>
                {sandboxes.data.items.map(s => (
                  <tr key={s.sandbox_id} className="border-t hover:bg-muted/50">
                    <td className="p-2">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(s.sandbox_id)}
                        onChange={() => toggleSelect(s.sandbox_id)}
                        className="cursor-pointer"
                      />
                    </td>
                    <td className="p-2 font-mono text-xs">{s.sandbox_id.slice(0, 8)}</td>
                    <td className="p-2 font-mono text-xs">{s.case_id.slice(0, 8)}</td>
                    <td className="p-2">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColor(s.status)}`}>
                        {s.status}
                      </span>
                    </td>
                    <td className="p-2 text-xs">
                      {s.compared_with?.length ?? 0}
                    </td>
                    <td className="p-2 text-xs text-muted-foreground">
                      {new Date(s.created_at).toLocaleDateString()}
                    </td>
                    <td className="p-2">
                      <button
                        onClick={() => setAddProposalSandboxId(s.sandbox_id)}
                        className="px-2 py-0.5 border rounded text-xs hover:bg-muted"
                      >
                        Add Proposal
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {selectedIds.length === 2 && (
            <div className="border rounded-lg p-4 space-y-4">
              <h2 className="text-lg font-semibold border-b pb-2">Comparison</h2>
              {comparison.isLoading && <LoadingSkeleton type="text" count={4} />}
              {comparison.error && (
                <ErrorPanel title="Comparison failed" message={comparison.error.message || "Unknown error"} />
              )}
              {comparison.data && comparison.data.differences.length === 0 && (
                <p className="text-sm text-muted-foreground">No differences found between selected sandboxes</p>
              )}
              {comparison.data && comparison.data.differences.length > 0 && (
                <div className="border rounded overflow-hidden">
                  <table className="w-full text-sm">
                    <thead className="bg-muted">
                      <tr>
                        <th className="p-2 text-left">Field</th>
                        <th className="p-2 text-left">Sandbox 1</th>
                        <th className="p-2 text-left">Sandbox 2</th>
                      </tr>
                    </thead>
                    <tbody>
                      {comparison.data.differences.map((d, i) => (
                        <tr key={i} className="border-t hover:bg-muted/50">
                          <td className="p-2 font-mono text-xs font-medium">{d.field}</td>
                          <td className="p-2 text-xs bg-red-50">
                            {typeof d.value_1 === "object" ? JSON.stringify(d.value_1) : String(d.value_1 ?? "—")}
                          </td>
                          <td className="p-2 text-xs bg-green-50">
                            {typeof d.value_2 === "object" ? JSON.stringify(d.value_2) : String(d.value_2 ?? "—")}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {selectedIds.length < 2 && (
            <p className="text-xs text-muted-foreground text-center py-2">
              Select 2 sandboxes to compare ({selectedIds.length}/2 selected)
            </p>
          )}
        </div>
      )}

      <Dialog.Root open={showCreate} onOpenChange={setShowCreate}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border rounded-lg shadow-lg p-6 z-50 w-96 max-w-[90vw]">
            <Dialog.Title className="font-semibold">Create Sandbox</Dialog.Title>
            <Dialog.Description className="text-xs text-muted-foreground mt-2">
              Create a new sandbox for a decision case
            </Dialog.Description>
            <input
              type="text"
              placeholder="Case ID"
              className="w-full px-3 py-2 border rounded text-sm mt-3"
              value={createCaseId}
              onChange={e => setCreateCaseId(e.target.value)}
            />
            <div className="flex gap-2 mt-4 justify-end">
              <Dialog.Close asChild>
                <button className="px-3 py-1 border rounded text-xs">Cancel</button>
              </Dialog.Close>
              <button
                onClick={() => createMutation.mutate(createCaseId)}
                disabled={!createCaseId || createMutation.isPending}
                className="px-3 py-1 bg-primary text-primary-foreground rounded text-xs disabled:opacity-50"
              >
                {createMutation.isPending ? "Creating..." : "Create"}
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <Dialog.Root open={!!addProposalSandboxId} onOpenChange={(v) => { if (!v) { setAddProposalSandboxId(null); setAddProposalId("") } }}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border rounded-lg shadow-lg p-6 z-50 w-96 max-w-[90vw]">
            <Dialog.Title className="font-semibold">Add Proposal to Sandbox</Dialog.Title>
            <Dialog.Description className="text-xs text-muted-foreground mt-2">
              Select a proposal to add to the sandbox
            </Dialog.Description>
            <input
              type="text"
              placeholder="Proposal ID"
              className="w-full px-3 py-2 border rounded text-sm mt-3"
              value={addProposalId}
              onChange={e => setAddProposalId(e.target.value)}
            />
            {caseProposals.data && caseProposals.data.proposals.length > 0 && (
              <div className="mt-2 max-h-32 overflow-auto border rounded text-xs">
                {caseProposals.data.proposals.map(p => (
                  <button
                    key={p.proposal_id}
                    onClick={() => setAddProposalId(p.proposal_id)}
                    className={`w-full text-left p-2 hover:bg-muted ${
                      addProposalId === p.proposal_id ? "bg-muted" : ""
                    }`}
                  >
                    <span className="font-mono">{p.proposal_id.slice(0, 8)}</span>
                    <span className="ml-2 text-muted-foreground">{p.title}</span>
                  </button>
                ))}
              </div>
            )}
            <div className="flex gap-2 mt-4 justify-end">
              <Dialog.Close asChild>
                <button className="px-3 py-1 border rounded text-xs">Cancel</button>
              </Dialog.Close>
              <button
                onClick={() => {
                  if (addProposalSandboxId && addProposalId) {
                    addProposalMutation.mutate({ sandboxId: addProposalSandboxId, proposalId: addProposalId })
                  }
                }}
                disabled={!addProposalId || addProposalMutation.isPending}
                className="px-3 py-1 bg-primary text-primary-foreground rounded text-xs disabled:opacity-50"
              >
                {addProposalMutation.isPending ? "Adding..." : "Add"}
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  )
}
