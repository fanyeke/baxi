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

// maxDimAlertsPerRun caps dimensional alerts per pipeline run to match the
// Python baseline suppression behaviour.
const maxDimAlertsPerRun = 50

// DetectAlertsStep evaluates global and dimensional alert rules against
// mart.metric_daily and mart.metric_dimension_daily, and writes triggered
// alerts into ops.metric_alert.
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

// Run evaluates all enabled global alert rules (against mart.metric_daily)
// and all dimensional alert rules (against mart.metric_dimension_daily),
// then writes triggered alerts to ops.metric_alert.
//
// Global and dimensional rules are evaluated independently and their results
// are merged into a single step output. The INSERT uses ON CONFLICT (alert_id)
// DO NOTHING so that re-running the pipeline is idempotent.
func (s *DetectAlertsStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	globalRules := alert.GlobalRules()
	dimRules := alert.DefaultDimensionalRules()
	totalInputRules := int64(countEnabledGlobalRules(globalRules) + len(dimRules))

	var outputCount int64

	globalResults, err := s.engine.EvaluateGlobalRules(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("detect_alerts: evaluate global rules: %w", err)
	}

	input.Logger.Info("global alert evaluation complete",
		zap.Int("triggered", len(globalResults)),
	)

	for _, r := range globalResults {
		evJSON := sanitizeJSON(r.EvidenceJSON)

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
			r.AlertID, r.RuleID, r.EventDate, string(r.Severity), r.MetricName,
			r.CurrentValue, r.BaselineValue, r.DeltaPct, r.SampleSize,
			evJSON, r.Message,
			deriveOwnerRole(r.RuleID),
		)
		if err != nil {
			return nil, fmt.Errorf("insert global alert %s: %w", r.AlertID, err)
		}
		outputCount++
	}

	dimAlerts, suppressed, err := alert.EvaluateDimensionRules(ctx, tx, nil, maxDimAlertsPerRun)
	if err != nil {
		return nil, fmt.Errorf("detect_alerts: evaluate dimensional rules: %w", err)
	}

	input.Logger.Info("dimensional alert evaluation complete",
		zap.Int("triggered", len(dimAlerts)),
		zap.Int("suppressed", suppressed),
	)

	for _, a := range dimAlerts {
		_, err := tx.Exec(ctx, `
			INSERT INTO ops.metric_alert (
				alert_id, rule_id, event_date, severity, metric_name,
				object_type, object_id, current_value, baseline_value,
				change_rate, sample_size, affected_orders, affected_gmv,
				impact_score, evidence_json, description,
				owner_role, status, created_at
			) VALUES (
				$1, $2, $3::DATE, $4, $5,
				$6, $7, $8, $9,
				$10, $11, $12, $13,
				$14, '{}'::JSONB, $15,
				$16, 'new', NOW()
			)
			ON CONFLICT (alert_id) DO NOTHING
		`,
			a.AlertID, a.RuleID, a.EventDate, a.Severity, a.MetricName,
			a.ObjectType, a.ObjectID, a.CurrentValue, a.BaselineValue,
			a.ChangeRate, a.SampleSize, a.AffectedOrders, a.AffectedGMV,
			a.ImpactScore, a.Description, a.OwnerRole,
		)
		if err != nil {
			return nil, fmt.Errorf("insert dimensional alert %s: %w", a.AlertID, err)
		}
		outputCount++
	}

	input.Logger.Info("alerts written",
		zap.Int64("alerts", outputCount),
		zap.Int64("global_rules", int64(len(globalResults))),
		zap.Int("dimensional_alerts", len(dimAlerts)),
	)

	return &pipeline.StepOutput{
		InputCount:  totalInputRules,
		OutputCount: outputCount,
	}, nil
}

func countEnabledGlobalRules(rules []alert.AlertRule) int {
	n := 0
	for _, r := range rules {
		if r.Enabled {
			n++
		}
	}
	return n
}

func sanitizeJSON(s string) string {
	if s == "" {
		return "{}"
	}
	if !json.Valid([]byte(s)) {
		return "{}"
	}
	return s
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
