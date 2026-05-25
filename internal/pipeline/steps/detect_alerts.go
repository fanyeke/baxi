package steps

import (
	"context"
	"encoding/json"
	"fmt"

	"baxi/internal/alert"
	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// DetectAlertsStep evaluates global alert rules against mart.metric_daily
// and writes triggered alerts into ops.metric_alert.
//
// The step uses INSERT ... ON CONFLICT DO NOTHING for idempotency —
// re-running the step produces the same alerts without duplicates.
type DetectAlertsStep struct {
	engine *alert.Engine
}

// NewDetectAlertsStep creates a new DetectAlertsStep.
func NewDetectAlertsStep() *DetectAlertsStep {
	return &DetectAlertsStep{
		engine: alert.NewEngine(),
	}
}

// Name returns the step name for audit logging.
func (s *DetectAlertsStep) Name() string {
	return "detect_alerts"
}

// Run evaluates all enabled global alert rules and writes triggered alerts
// to ops.metric_alert. Dead rules (review_score_drop, seller_activation_gap)
// are silently skipped because their Enabled flag is false.
//
// The INSERT uses ON CONFLICT (alert_id) DO NOTHING so that re-running
// the pipeline is idempotent.
func (s *DetectAlertsStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	results, err := s.engine.EvaluateGlobalRules(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("detect_alerts: evaluate global rules: %w", err)
	}

	input.Logger.Info("global alert evaluation complete",
		zap.Int("triggered", len(results)),
	)

	if len(results) == 0 {
		return &pipeline.StepOutput{
			InputCount:  int64(len(alert.GlobalRules())),
			OutputCount: 0,
		}, nil
	}

	var outputCount int64
	for _, r := range results {
		evJSON := r.EvidenceJSON
		if evJSON == "" {
			evJSON = "{}"
		}

		// Validate JSON
		if !json.Valid([]byte(evJSON)) {
			evJSON = "{}"
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO ops.metric_alert (
				alert_id, rule_id, event_date, severity, metric_name,
				object_type, object_id, current_value, baseline_value,
				change_rate, sample_size, evidence_json, description,
				owner_role, status, created_at
			) VALUES (
				$1, $2, $3::DATE, $4, $5,
				'global', 'global', $6, $7,
				$8, $9, $10::JSONB, $11,
				$12, 'new', NOW()
			)
			ON CONFLICT (alert_id) DO NOTHING
		`,
			r.AlertID,
			r.RuleID,
			r.EventDate,
			string(r.Severity),
			r.MetricName,
			r.CurrentValue,
			r.BaselineValue,
			r.DeltaPct,
			r.SampleSize,
			evJSON,
			r.Message,
			deriveOwnerRole(r.RuleID),
		)
		if err != nil {
			return nil, fmt.Errorf("insert alert %s: %w", r.AlertID, err)
		}
		outputCount++
	}

	input.Logger.Info("alerts written",
		zap.Int64("alerts", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  int64(len(alert.GlobalRules())),
		OutputCount: outputCount,
	}, nil
}

// deriveOwnerRole maps rule_id to the owner_role from the Python baseline.
func deriveOwnerRole(ruleID string) string {
	switch ruleID {
	case "gmv_drop":
		return "business_ops"
	case "late_delivery_spike":
		return "logistics_ops"
	case "cancel_rate_spike":
		return "logistics_ops"
	case "review_score_drop":
		return "category_ops"
	case "seller_activation_gap":
		return "seller_ops"
	default:
		return "unassigned"
	}
}
