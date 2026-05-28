package action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"baxi/internal/outbox"
	"github.com/jackc/pgx/v5"
)

// CreateOutboxEventFromProposal builds an OutboxEvent from a successfully
// executed ActionProposal and persists it via the outbox repository.
// The event is created within the caller's transaction so that failure to
// insert rolls back the entire execution.
func CreateOutboxEventFromProposal(ctx context.Context, tx pgx.Tx, proposal *ActionProposal) (*outbox.OutboxEvent, error) {
	channel := actionChannel(proposal.ActionType)
	eventID := fmt.Sprintf("evt_%d", time.Now().UnixNano())

	envelope := map[string]interface{}{
		"proposal_id": proposal.ProposalID,
		"case_id":     proposal.CaseID,
		"action_type": proposal.ActionType,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	if proposal.Payload != nil {
		envelope["payload"] = proposal.Payload
	}

	payloadJSON, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox envelope: %w", err)
	}

	event := &outbox.OutboxEvent{
		EventID:          eventID,
		SourceType:       "action_execution",
		SourceID:         proposal.ProposalID,
		EventType:        proposal.ActionType,
		Status:           "pending",
		Payload:          payloadJSON,
		TargetChannel:    channel,
		DispatchAttempts: 0,
	}

	repo := outbox.NewOutboxRepository()
	_, err = repo.CreateEvent(ctx, tx, event)
	if err != nil {
		return nil, fmt.Errorf("create outbox event: %w", err)
	}

	return event, nil
}
