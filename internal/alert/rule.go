// Package alert defines alert rules and provides a rule engine that evaluates
// global-level metrics from mart.metric_daily and produces deterministic,
// idempotent alerts in ops.metric_alert.
package alert

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Severity levels for alert classification.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// AlertRule defines a single global alert rule with its metadata and condition.
// For dead rules, set Enabled to false — they remain defined but are never
// evaluated, matching the Python baseline behaviour (review_score_drop and
// seller_activation_gap are dimension-level rules that never run at global
// scope).
type AlertRule struct {
	RuleID    string
	Name      string
	Severity  Severity
	Metric    string
	Condition func(ctx context.Context, tx pgx.Tx, date string) (*AlertResult, error)
	Enabled   bool
}

// AlertResult holds the output of a single rule evaluation.
type AlertResult struct {
	AlertID       string
	RuleID        string
	EventDate     string
	MetricName    string
	CurrentValue  float64
	BaselineValue float64
	DeltaValue    float64
	DeltaPct      float64
	Severity      Severity
	Message       string
	EvidenceJSON  string
	SampleSize    int64
}

// GlobalRules returns all global alert rules.
//
// Working rules (Enabled=true):
//   - gmv_drop:             fires when 7d avg drops ≥15% vs prev 14d avg
//   - late_delivery_spike:  fires when latest late_delivery_rate > 0.25
//   - cancel_rate_spike:    fires when |change_rate| > 0.5 AND value > 0.05
//
// Dead rules (Enabled=false, never trigger):
//   - review_score_drop:     defined but never evaluated (dimension-only in Python)
//   - seller_activation_gap: defined but never evaluated (no data available)
func GlobalRules() []AlertRule {
	return []AlertRule{
		{
			RuleID:    "gmv_drop",
			Name:      "GMV下降",
			Severity:  SeverityHigh,
			Metric:    "gmv",
			Condition: evaluateGMVDrop,
			Enabled:   true,
		},
		{
			RuleID:    "late_delivery_spike",
			Name:      "延迟飙升",
			Severity:  SeverityHigh,
			Metric:    "late_delivery_rate",
			Condition: evaluateLateDeliverySpike,
			Enabled:   true,
		},
		{
			RuleID:    "cancel_rate_spike",
			Name:      "取消率上升",
			Severity:  SeverityMedium,
			Metric:    "cancel_rate",
			Condition: evaluateCancelRateSpike,
			Enabled:   true,
		},
		{
			RuleID:    "review_score_drop",
			Name:      "评分下降",
			Severity:  SeverityMedium,
			Metric:    "avg_review_score",
			Condition: nil, // DEAD RULE — never evaluated
			Enabled:   false,
		},
		{
			RuleID:    "seller_activation_gap",
			Name:      "卖家活跃度下降",
			Severity:  SeverityLow,
			Metric:    "active_sellers",
			Condition: nil, // DEAD RULE — never evaluated
			Enabled:   false,
		},
	}
}

// GenerateAlertID creates a deterministic alert ID using SHA-256.
//
// The algorithm matches the Python dimensional rule engine:
//
//	raw = f"{rule_id}\xff{metric_date}\xff{dim_type}\xff{dim_value}"
//	return f"dim-{hashlib.sha256(raw.encode()).hexdigest()[:12]}"
//
// For global alerts dimType and dimValue are both "global", producing an ID
// like "76085bfcd31d". The caller may prefix as needed (the Python engine
// prefixes "dim-" for dimensional alerts).
func GenerateAlertID(ruleID, metricDate, dimType, dimValue string) string {
	// NOTE: Python f"..." encodes \xff (U+00FF) as UTF-8 \xc3\xbf.
	// We must match exactly: raw.encode() → UTF-8 bytes.
	raw := ruleID + "\xc3\xbf" + metricDate + "\xc3\xbf" + dimType + "\xc3\xbf" + dimValue
	hash := sha256.Sum256([]byte(raw))
	// First 12 hex chars of the full 64-char hex string, matching Python's hexdigest()[:12]
	fullHex := fmt.Sprintf("%x", hash)
	return fullHex[:12]
}

// GlobalAlertID returns the event_id used for global alerts.
// Python baseline uses: f"{rule_id}_{event_date}"
func GlobalAlertID(ruleID, eventDate string) string {
	return ruleID + "_" + eventDate
}

// formatTimestamp returns the current UTC time in ISO-8601 format matching
// the Python baseline output.
func formatTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000")
}
