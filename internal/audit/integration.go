package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Integration provides standardized audit logging for review, action, and outbox operations.
// All methods insert into audit.audit_log using the provided transaction.
type Integration struct{}

// NewIntegration creates a new audit integration helper.
func NewIntegration() *Integration {
	return &Integration{}
}

// LogProposalReviewed inserts an audit log entry for a proposal review operation.
// category: "review", action: "proposal_reviewed"
// resource_type: "action_proposal"
// metadata: {"verdict": verdict, "feedback": feedback}
func (i *Integration) LogProposalReviewed(ctx context.Context, tx pgx.Tx, proposalID, reviewerID, verdict, feedback string) error {
	actor := defaultActor(reviewerID)

	metadata := map[string]interface{}{
		"verdict":  verdict,
		"feedback": feedback,
	}
	return insertAuditLog(ctx, tx, "review", "proposal_reviewed", actor, "action_proposal", proposalID, metadata)
}

// LogProposalExecuted inserts an audit log entry for a proposal execution operation.
// category: "action_apply"
// action: "proposal_executed" (if success) or "proposal_execution_failed" (if !success)
// resource_type: "action_proposal"
// metadata: {"success": success, "dry_run": dry_run, "error": errorMsg} (error omitted if empty)
func (i *Integration) LogProposalExecuted(ctx context.Context, tx pgx.Tx, proposalID, actorID string, success, dryRun bool, errorMsg string) error {
	actor := defaultActor(actorID)

	action := "proposal_executed"
	if !success {
		action = "proposal_execution_failed"
	}

	metadata := map[string]interface{}{
		"success": success,
		"dry_run": dryRun,
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	return insertAuditLog(ctx, tx, "action_apply", action, actor, "action_proposal", proposalID, metadata)
}

// LogOutboxDispatched inserts an audit log entry for an outbox dispatch operation.
// category: "outbox"
// action: "outbox_dispatched" (if success) or "outbox_dispatch_failed" (if !success)
// resource_type: "outbox_event"
// metadata: {"channel": channel, "success": success, "error": errorMsg} (error omitted if empty)
func (i *Integration) LogOutboxDispatched(ctx context.Context, tx pgx.Tx, eventID, channel string, success bool, errorMsg string) error {
	action := "outbox_dispatched"
	if !success {
		action = "outbox_dispatch_failed"
	}

	metadata := map[string]interface{}{
		"channel": channel,
		"success": success,
	}
	if errorMsg != "" {
		metadata["error"] = errorMsg
	}
	return insertAuditLog(ctx, tx, "outbox", action, "system", "outbox_event", eventID, metadata)
}

// insertAuditLog is the shared helper that performs the actual INSERT into audit.audit_log.
func insertAuditLog(ctx context.Context, tx pgx.Tx, category, action, actor, resourceType, resourceID string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	query := `
		INSERT INTO audit.audit_log (category, action, actor, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, query, category, action, actor, resourceType, resourceID, metadataJSON)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// defaultActor returns "system" if actorID is empty, otherwise returns actorID.
func defaultActor(actorID string) string {
	if actorID == "" {
		return "system"
	}
	return actorID
}
