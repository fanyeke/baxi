import { useState } from "react"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import * as Dialog from "@radix-ui/react-dialog"
import { apiClient } from "../api/client"
import type { CaseListResponse, ProposalListResponse, ActionProposal, ReviewResponse } from "../api/types"
import { EmptyState } from "../components/EmptyState"
import { LoadingSkeleton } from "../components/LoadingSkeleton"
import { ErrorPanel } from "../components/ErrorPanel"

const STATUS_OPTIONS = ["", "proposed", "approved", "rejected", "cancelled"] as const

function riskColor(level: string): string {
  switch (level) {
    case "low": return "bg-green-100 text-green-700"
    case "medium": return "bg-yellow-100 text-yellow-700"
    case "high": return "bg-orange-100 text-orange-700"
    case "critical": return "bg-red-100 text-red-700"
    default: return "bg-gray-100 text-gray-700"
  }
}

function statusColor(status: string): string {
  switch (status) {
    case "approved": return "bg-green-100 text-green-700"
    case "rejected": return "bg-red-100 text-red-700"
    case "cancelled": return "bg-gray-100 text-gray-700"
    case "proposed": return "bg-blue-100 text-blue-700"
    default: return "bg-gray-100 text-gray-700"
  }
}

export default function DecisionReview() {
  const queryClient = useQueryClient()
  const [statusFilter, setStatusFilter] = useState("")
  const [caseSearch, setCaseSearch] = useState("")
  const [selectedProposal, setSelectedProposal] = useState<ActionProposal | null>(null)

  const cases = useQuery({
    queryKey: ["decision-cases"],
    queryFn: () => apiClient.get<CaseListResponse>("/decisions/cases", { limit: "100" }),
  })

  const filteredCases = cases.data?.cases.filter(c =>
    !caseSearch || c.case_id.includes(caseSearch)
  ) ?? []

  const selectedCaseId = selectedProposal?.case_id ?? filteredCases[0]?.case_id ?? ""

  const proposals = useQuery({
    queryKey: ["proposals", selectedCaseId],
    queryKeyHashFn: () => `proposals-${selectedCaseId}`,
    queryFn: () => apiClient.get<ProposalListResponse>(`/decisions/cases/${selectedCaseId}/proposals`),
    enabled: !!selectedCaseId,
  })

  const filteredProposals = proposals.data?.proposals.filter(p =>
    !statusFilter || p.apply_status === statusFilter
  ) ?? []

  const reviewRecord = useQuery({
    queryKey: ["review-record", selectedProposal?.proposal_id],
    queryFn: () => apiClient.get<ReviewResponse>(`/proposals/${selectedProposal!.proposal_id}/review`),
    enabled: !!selectedProposal?.proposal_id,
  })

  const approveMutation = useMutation({
    mutationFn: (args: { proposalId: string; feedback: string }) =>
      apiClient.post(`/proposals/${args.proposalId}/approve`, { reviewer_id: "operator", feedback: args.feedback }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] })
      setSelectedProposal(null)
    },
  })

  const rejectMutation = useMutation({
    mutationFn: (args: { proposalId: string; feedback: string }) =>
      apiClient.post(`/proposals/${args.proposalId}/reject`, { reviewer_id: "operator", feedback: args.feedback }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] })
      setSelectedProposal(null)
    },
  })

  const cancelMutation = useMutation({
    mutationFn: (args: { proposalId: string; reason: string }) =>
      apiClient.post(`/proposals/${args.proposalId}/cancel`, { reviewer_id: "operator", feedback: args.reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["proposals"] })
      setSelectedProposal(null)
    },
  })

  const [dialogAction, setDialogAction] = useState<"approve" | "reject" | "cancel" | null>(null)
  const [dialogFeedback, setDialogFeedback] = useState("")

  function handleActionConfirm() {
    if (!selectedProposal || !dialogAction) return
    if (dialogAction === "approve") {
      approveMutation.mutate({ proposalId: selectedProposal.proposal_id, feedback: dialogFeedback })
    } else if (dialogAction === "reject") {
      rejectMutation.mutate({ proposalId: selectedProposal.proposal_id, feedback: dialogFeedback })
    } else {
      cancelMutation.mutate({ proposalId: selectedProposal.proposal_id, reason: dialogFeedback })
    }
    setDialogAction(null)
    setDialogFeedback("")
  }

  const isLoading = cases.isLoading || (selectedCaseId && proposals.isLoading)
  const error = cases.error || proposals.error

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-2xl font-bold">Decision Review</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Review and manage action proposals from the decision engine
        </p>
      </div>

      <div className="flex gap-2">
        <select
          className="px-3 py-1 border rounded text-sm"
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value)}
        >
          <option value="">All Statuses</option>
          {STATUS_OPTIONS.filter(Boolean).map(s => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
        <input
          type="text"
          placeholder="Search case ID..."
          className="px-3 py-1 border rounded text-sm flex-1"
          value={caseSearch}
          onChange={e => setCaseSearch(e.target.value)}
        />
      </div>

      {isLoading && <LoadingSkeleton type="table" count={5} />}
      {error && <ErrorPanel title="Failed to load" message={error.message || "Unknown error"} />}
      {!isLoading && !error && filteredCases.length === 0 && (
        <EmptyState title="No cases found" description="Create a decision case to see proposals" />
      )}

      {!isLoading && !error && filteredCases.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div className="lg:col-span-2 border rounded-lg overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-muted">
                <tr>
                  <th className="p-2 text-left">Proposal ID</th>
                  <th className="p-2 text-left">Case ID</th>
                  <th className="p-2 text-left">Action</th>
                  <th className="p-2 text-left">Title</th>
                  <th className="p-2 text-left">Risk</th>
                  <th className="p-2 text-left">Status</th>
                  <th className="p-2 text-left">HITL</th>
                  <th className="p-2 text-left">Created</th>
                </tr>
              </thead>
              <tbody>
                {filteredProposals.map(p => (
                  <tr
                    key={p.proposal_id}
                    className={`border-t hover:bg-muted/50 cursor-pointer ${
                      selectedProposal?.proposal_id === p.proposal_id ? "bg-muted/70" : ""
                    }`}
                    onClick={() => setSelectedProposal(p)}
                  >
                    <td className="p-2 font-mono text-xs">{p.proposal_id.slice(0, 8)}</td>
                    <td className="p-2 font-mono text-xs">{p.case_id.slice(0, 8)}</td>
                    <td className="p-2 text-xs">{p.action_type}</td>
                    <td className="p-2 font-medium max-w-[200px] truncate">{p.title}</td>
                    <td className="p-2">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${riskColor(p.risk_level)}`}>
                        {p.risk_level}
                      </span>
                    </td>
                    <td className="p-2">
                      <span className={`px-2 py-0.5 rounded text-xs font-medium ${statusColor(p.apply_status)}`}>
                        {p.apply_status}
                      </span>
                    </td>
                    <td className="p-2">
                      {p.requires_human_review ? (
                        <span className="text-orange-600 text-xs font-medium">Yes</span>
                      ) : (
                        <span className="text-muted-foreground text-xs">No</span>
                      )}
                    </td>
                    <td className="p-2 text-xs text-muted-foreground">
                      {new Date(p.created_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {filteredProposals.length === 0 && selectedCaseId && (
              <div className="p-6 text-center text-muted-foreground text-sm">
                No proposals match the current filter
              </div>
            )}
          </div>

          <div className="border rounded-lg p-4 space-y-4">
            {selectedProposal ? (
              <>
                <h2 className="text-lg font-semibold border-b pb-2">Proposal Details</h2>

                <div className="space-y-2">
                  <div>
                    <label className="text-xs text-muted-foreground block">Proposal ID</label>
                    <span className="font-mono text-sm">{selectedProposal.proposal_id}</span>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground block">Case ID</label>
                    <span className="font-mono text-sm">{selectedProposal.case_id}</span>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground block">Decision ID</label>
                    <span className="font-mono text-xs text-muted-foreground">{selectedProposal.decision_id}</span>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground block">Action Type</label>
                    <span className="text-sm">{selectedProposal.action_type}</span>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground block">Risk Level</label>
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${riskColor(selectedProposal.risk_level)}`}>
                      {selectedProposal.risk_level}
                    </span>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground block">Status</label>
                    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${statusColor(selectedProposal.apply_status)}`}>
                      {selectedProposal.apply_status}
                    </span>
                  </div>
                </div>

                {selectedProposal.description && (
                  <div>
                    <label className="text-xs text-muted-foreground block mb-1">Description</label>
                    <p className="text-sm">{selectedProposal.description}</p>
                  </div>
                )}

                {selectedProposal.payload && Object.keys(selectedProposal.payload).length > 0 && (
                  <div>
                    <label className="text-xs text-muted-foreground block mb-1">Payload</label>
                    <pre className="text-xs bg-muted p-2 rounded overflow-auto max-h-48">
                      {JSON.stringify(selectedProposal.payload, null, 2)}
                    </pre>
                  </div>
                )}

                <div className="flex gap-2 pt-2">
                  <button
                    onClick={() => setDialogAction("approve")}
                    className="px-3 py-1 bg-green-600 text-white rounded text-xs hover:bg-green-700"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => setDialogAction("reject")}
                    className="px-3 py-1 bg-red-600 text-white rounded text-xs hover:bg-red-700"
                  >
                    Reject
                  </button>
                  <button
                    onClick={() => setDialogAction("cancel")}
                    className="px-3 py-1 border border-gray-300 rounded text-xs hover:bg-muted"
                  >
                    Cancel
                  </button>
                </div>

                <div className="border-t pt-3 mt-3">
                  <h3 className="text-sm font-semibold mb-2">Review History</h3>
                  {reviewRecord.isLoading && <LoadingSkeleton type="text" count={2} />}
                  {reviewRecord.error && (
                    <p className="text-xs text-muted-foreground">No review records</p>
                  )}
                  {reviewRecord.data && (
                    <div className="bg-muted rounded p-3 text-xs space-y-1">
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Verdict</span>
                        <span className="font-medium">{reviewRecord.data.verdict}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Reviewer</span>
                        <span>{reviewRecord.data.reviewer_id}</span>
                      </div>
                      {reviewRecord.data.feedback && (
                        <div>
                          <span className="text-muted-foreground">Feedback</span>
                          <p className="mt-1">{reviewRecord.data.feedback}</p>
                        </div>
                      )}
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Time</span>
                        <span>{new Date(reviewRecord.data.created_at).toLocaleString()}</span>
                      </div>
                    </div>
                  )}
                </div>
              </>
            ) : (
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <span className="text-3xl text-muted-foreground/40 mb-3">→</span>
                <p className="text-muted-foreground font-medium">Select a proposal to view details</p>
              </div>
            )}
          </div>
        </div>
      )}

      <Dialog.Root open={dialogAction !== null} onOpenChange={(v) => { if (!v) { setDialogAction(null); setDialogFeedback("") } }}>
        <Dialog.Portal>
          <Dialog.Overlay className="fixed inset-0 bg-black/40 z-40" />
          <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-background border rounded-lg shadow-lg p-6 z-50 w-96 max-w-[90vw]">
            <Dialog.Title className="font-semibold">
              {dialogAction === "approve" ? "Approve" : dialogAction === "reject" ? "Reject" : "Cancel"} Proposal
            </Dialog.Title>
            <Dialog.Description className="text-xs text-muted-foreground mt-2">
              Are you sure you want to {dialogAction} this proposal?
            </Dialog.Description>
            <input
              type="text"
              placeholder={dialogAction === "cancel" ? "Reason (optional)" : "Feedback (optional)"}
              className="w-full px-3 py-2 border rounded text-sm mt-3"
              value={dialogFeedback}
              onChange={e => setDialogFeedback(e.target.value)}
            />
            <div className="flex gap-2 mt-4 justify-end">
              <Dialog.Close asChild>
                <button onClick={() => { setDialogAction(null); setDialogFeedback("") }} className="px-3 py-1 border rounded text-xs">
                  Cancel
                </button>
              </Dialog.Close>
              <button
                onClick={handleActionConfirm}
                className={`px-3 py-1 rounded text-xs text-white ${
                  dialogAction === "reject" ? "bg-destructive" : "bg-primary"
                }`}
              >
                {dialogAction === "approve" ? "Approve" : dialogAction === "reject" ? "Reject" : "Confirm"}
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </div>
  )
}
