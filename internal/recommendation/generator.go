// Package recommendation generates recommendation and task records from alerts.
//
// Two generators coexist in this package:
//   - Generate() — reads ops.metric_alert, produces ops.recommendation (template-based)
//   - TaskGenerator.GenerateTasks() — reads ops.recommendation, produces ops.task
//
// Both match the Python baseline (db_generate_recommendations.py).
package recommendation

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// TaskRecord holds the fields for a single ops.task row to be inserted.
type TaskRecord struct {
	TaskID           string
	RecommendationID string
	AlertID          string
	TaskTitle        string
	TaskDescription  string
	TargetObjectType string
	TargetObjectID   string
	TaskSource       string
	OwnerRole        string
	Priority         string
	Status           string
}

// TaskGenerator reads ops.recommendation and produces ops.task records.
type TaskGenerator struct{}

// NewTaskGenerator creates a new TaskGenerator.
func NewTaskGenerator() *TaskGenerator {
	return &TaskGenerator{}
}

// GenerateTasks queries all recommendations from ops.recommendation and
// returns corresponding task records with status='pending'.
//
// The output matches the Python baseline: 1 task per recommendation (36 total).
// Task IDs are derived from recommendation IDs:
//   - "rec-..."       → "task-..."       (global heuristic strategy)
//   - "dimrec-..."    → "dimtask-..."    (dimensional rule)
func (g *TaskGenerator) GenerateTasks(ctx context.Context, tx pgx.Tx) ([]TaskRecord, error) {
	rows, err := tx.Query(ctx, `
		SELECT recommendation_id,
		       COALESCE(alert_id, '') AS alert_id,
		       strategy_title,
		       COALESCE(strategy_detail, '') AS strategy_detail,
		       COALESCE(target_object_type, '') AS target_object_type,
		       COALESCE(target_object_id, '') AS target_object_id,
		       COALESCE(risk_level, 'medium') AS risk_level,
		       COALESCE(owner_role, '') AS owner_role
		FROM ops.recommendation
		ORDER BY recommendation_id
	`)
	if err != nil {
		return nil, fmt.Errorf("query recommendations: %w", err)
	}
	defer rows.Close()

	var tasks []TaskRecord
	for rows.Next() {
		var (
			recID, alertID       string
			title, detail        string
			objType, objID       string
			riskLevel, ownerRole string
		)
		if err := rows.Scan(
			&recID, &alertID,
			&title, &detail,
			&objType, &objID,
			&riskLevel, &ownerRole,
		); err != nil {
			return nil, fmt.Errorf("scan recommendation: %w", err)
		}

		task := TaskRecord{
			TaskID:           deriveTaskID(recID),
			RecommendationID: recID,
			AlertID:          alertID,
			TaskTitle:        title,
			TaskDescription:  detail,
			TargetObjectType: objType,
			TargetObjectID:   objID,
			TaskSource:       deriveTaskSource(recID),
			OwnerRole:        ownerRole,
			Priority:         derivePriority(riskLevel),
			Status:           "pending",
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return tasks, nil
}

// =========================================================================
// Recommendation Generator (from alerts)
// =========================================================================

// AlertData holds the relevant fields from ops.metric_alert needed to
// generate a recommendation record.
type AlertData struct {
	AlertID        string
	RuleID         string
	EventDate      string
	Severity       string
	MetricName     string
	ObjectType     string
	ObjectID       string
	CurrentValue   *float64
	BaselineValue  *float64
	ChangeRate     *float64
	SampleSize     *int64
	AffectedOrders *int64
	AffectedGMV    *float64
	Description    string
	OwnerRole      string
}

// ruleTemplate renders recommendation fields for a given rule_id.
// Returns: title, detail, expectedImpact, successMetric, confidence, requiresApproval.
type ruleTemplate func(alert AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool)

// ruleTemplateMap maps rule_id to its template renderer. Matches Python
// RULE_TO_TEMPLATE in db_generate_recommendations.py.
var ruleTemplateMap = map[string]ruleTemplate{
	"seller_late_delivery_spike":  templateSellerDelivery,
	"seller_review_score_drop":    templateSellerReview,
	"category_gmv_drop":           templateCategoryGMV,
	"category_low_review_cluster": templateCategoryReview,
	"region_cancel_rate_spike":    templateRegionCancel,
	"region_late_delivery_spike":  templateRegionDelivery,
}

// confidenceFromSample computes confidence from sample_size threshold.
// Matches Python _compute_confidence(sample_size, min_sample_size=20).
func confidenceFromSample(sampleSize *int64, minSample int64) string {
	if sampleSize == nil {
		return "low"
	}
	s := *sampleSize
	if s > minSample*2 {
		return "high"
	} else if s > minSample {
		return "medium"
	}
	return "low"
}

// ---------------------------------------------------------------------------
// Dimension-level templates (matching action_templates.yml)
// ---------------------------------------------------------------------------

func templateSellerDelivery(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, gmv, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), ptrFmt2d(a.AffectedGMV), a.ObjectID
	title = fmt.Sprintf("seller %s: late_delivery_rate anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】卖家 %s 延迟配送率达到 %s，高于阈值 0.25。\n"+
			"【证据】样本订单数 %s，影响 GMV %s，规则 %s。\n"+
			"【判断】该卖家可能存在发货、库存或物流交接问题。\n"+
			"【建议动作】请卖家运营检查该卖家近期订单履约链路，确认是否需要限制流量或人工沟通。\n"+
			"【预期收益】降低延迟配送率，减少差评与取消风险。\n"+
			"【验收指标】未来 7 日该卖家 late_delivery_rate 低于 20。",
		oid, cv, ss, gmv, a.RuleID,
	)
	return title, detail, "Stabilize late_delivery_rate", "late_delivery_rate", confidenceFromSample(a.SampleSize, 20), false
}

func templateSellerReview(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, gmv, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), ptrFmt2d(a.AffectedGMV), a.ObjectID
	title = fmt.Sprintf("seller %s: avg_review_score anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】卖家 %s 平均评分 %s，低于阈值 3.5。\n"+
			"【证据】样本订单数 %s，影响 GMV %s，规则 %s。\n"+
			"【判断】该卖家可能存在商品质量、描述不符或服务问题。\n"+
			"【建议动作】请卖家运营审查该卖家商品评价，确认是否存在质量问题。\n"+
			"【预期收益】提升卖家评分，改善买家体验。\n"+
			"【验收指标】未来 7 日该卖家 avg_review_score 高于 3.5。",
		oid, cv, ss, gmv, a.RuleID,
	)
	return title, detail, "Stabilize avg_review_score", "avg_review_score", confidenceFromSample(a.SampleSize, 20), false
}

func templateCategoryGMV(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), a.ObjectID
	title = fmt.Sprintf("category %s: gmv anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】品类 %s GMV 环比下降 %s，跌幅超过 20。\n"+
			"【证据】样本订单数 %s，规则 %s。\n"+
			"【判断】该品类可能存在需求下降、库存不足或竞品冲击。\n"+
			"【建议动作】请品类运营分析该品类近期销售趋势，确认是否需要调整营销策略。\n"+
			"【预期收益】恢复品类 GMV 至正常水平。\n"+
			"【验收指标】未来 14 日该品类 GMV 环比跌幅收窄至 10%% 以内。",
		oid, cv, ss, a.RuleID,
	)
	return title, detail, "Stabilize gmv", "gmv", confidenceFromSample(a.SampleSize, 30), false
}

func templateCategoryReview(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, gmv, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), ptrFmt2d(a.AffectedGMV), a.ObjectID
	title = fmt.Sprintf("category %s: low_review_rate anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】品类 %s 差评率达到 %s，超过阈值 15。\n"+
			"【证据】样本评价数 %s，影响 GMV %s，规则 %s。\n"+
			"【判断】该品类可能存在普遍质量问题或描述不符。\n"+
			"【建议动作】请品类运营抽查该品类近期差评，确认是否存在系统性问题。\n"+
			"【预期收益】降低品类差评率，改善买家满意度。\n"+
			"【验收指标】未来 7 日该品类 low_review_rate 低于 12。",
		oid, cv, ss, gmv, a.RuleID,
	)
	return title, detail, "Stabilize low_review_rate", "low_review_rate", confidenceFromSample(a.SampleSize, 30), false
}

func templateRegionCancel(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, gmv, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), ptrFmt2d(a.AffectedGMV), a.ObjectID
	title = fmt.Sprintf("region %s: cancel_rate anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】区域 %s 取消率达到 %s，超过阈值 5。\n"+
			"【证据】样本订单数 %s，影响 GMV %s，规则 %s。\n"+
			"【判断】该区域可能存在物流覆盖不足或卖家集中取消问题。\n"+
			"【建议动作】请物流运营分析该区域取消原因，确认是否需要调整物流策略。\n"+
			"【预期收益】降低区域取消率，提升订单完成率。\n"+
			"【验收指标】未来 7 日该区域 cancel_rate 低于 4。",
		oid, cv, ss, gmv, a.RuleID,
	)
	return title, detail, "Stabilize cancel_rate", "cancel_rate", confidenceFromSample(a.SampleSize, 30), false
}

func templateRegionDelivery(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	cv, ss, gmv, oid := ptrFmt6d(a.CurrentValue), ptrFmtInt(a.SampleSize), ptrFmt2d(a.AffectedGMV), a.ObjectID
	title = fmt.Sprintf("region %s: late_delivery_rate anomaly", oid)
	detail = fmt.Sprintf(
		"【问题】区域 %s 延迟配送率达到 %s，超过阈值 20。\n"+
			"【证据】样本订单数 %s，影响 GMV %s，规则 %s。\n"+
			"【判断】该区域可能存在物流配送瓶颈或基础设施问题。\n"+
			"【建议动作】请物流运营排查该区域物流链路，确认是否需要增加配送资源。\n"+
			"【预期收益】降低区域延迟配送率，改善履约体验。\n"+
			"【验收指标】未来 7 日该区域 late_delivery_rate 低于 18。",
		oid, cv, ss, gmv, a.RuleID,
	)
	return title, detail, "Stabilize late_delivery_rate", "late_delivery_rate", confidenceFromSample(a.SampleSize, 30), false
}

// templateGlobalRule generates recommendations for global-level alerts that
// don't have a specific dimensional template (gmv_drop, late_delivery_spike, etc.).
func templateGlobalRule(a AlertData) (title, detail, impact, successMetric, confidence string, requiresApproval bool) {
	title = fmt.Sprintf("Investigate: %s", truncStr(a.Description, 60))
	cv := orElseFmt(a.CurrentValue, "N/A", func(v float64) string { return fmt.Sprintf("%.4f", v) })
	bv := orElseFmt(a.BaselineValue, "N/A", func(v float64) string { return fmt.Sprintf("%.4f", v) })
	cr := orElseFmt(a.ChangeRate, "N/A", func(v float64) string { return fmt.Sprintf("%.2f%%", v*100) })
	detail = fmt.Sprintf("Rule '%s' triggered for %s on %s. Current: %s, Baseline: %s, Change: %s.", a.RuleID, a.MetricName, a.EventDate, cv, bv, cr)
	return title, detail, fmt.Sprintf("Stabilize %s", a.MetricName), a.MetricName, "medium", false
}

// Generate reads all alerts from ops.metric_alert and inserts recommendations
// into ops.recommendation using template-based generation.
//
// Uses INSERT … ON CONFLICT (recommendation_id) DO NOTHING for idempotency.
// Returns the number of inserted recommendations.
func Generate(ctx context.Context, tx pgx.Tx, logger *zap.Logger) (int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			alert_id, rule_id, event_date::TEXT, severity, metric_name,
			object_type, object_id,
			current_value, baseline_value, change_rate,
			sample_size, affected_orders, affected_gmv,
			COALESCE(description, '') AS description,
			COALESCE(owner_role, '') AS owner_role
		FROM ops.metric_alert
		ORDER BY alert_id
	`)
	if err != nil {
		return 0, fmt.Errorf("query ops.metric_alert: %w", err)
	}
	defer rows.Close()

	var alerts []AlertData
	for rows.Next() {
		var a AlertData
		if err := rows.Scan(
			&a.AlertID, &a.RuleID, &a.EventDate, &a.Severity, &a.MetricName,
			&a.ObjectType, &a.ObjectID,
			&a.CurrentValue, &a.BaselineValue, &a.ChangeRate,
			&a.SampleSize, &a.AffectedOrders, &a.AffectedGMV,
			&a.Description, &a.OwnerRole,
		); err != nil {
			return 0, fmt.Errorf("scan alert row: %w", err)
		}
		alerts = append(alerts, a)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate alert rows: %w", err)
	}

	if logger != nil {
		logger.Info("loaded alerts for recommendation generation", zap.Int("alert_count", len(alerts)))
	}

	var inserted int64
	for _, a := range alerts {
		recID := "rec-" + a.AlertID
		isDimensional := a.ObjectType != "" && a.ObjectType != "global"
		tmpl, hasT := ruleTemplateMap[a.RuleID]

		var title, detail, impact, successMetric, confidence string
		var requiresApproval bool

		if isDimensional && hasT {
			title, detail, impact, successMetric, confidence, requiresApproval = tmpl(a)
		} else {
			title, detail, impact, successMetric, confidence, requiresApproval = templateGlobalRule(a)
		}

		riskLevel := a.Severity
		if riskLevel == "" {
			riskLevel = "medium"
		}

		detail = truncStr(detail, 1000)

		_, err := tx.Exec(ctx, `
			INSERT INTO ops.recommendation (
				recommendation_id, alert_id, decision_source, rule_id,
				strategy_title, strategy_detail,
				target_object_type, target_object_id,
				expected_impact, risk_level, confidence,
				requires_approval, approval_status, execution_status,
				owner_role, success_metric, created_at
			) VALUES (
				$1, $2, 'rule_based', $3,
				$4, $5,
				$6, $7,
				$8, $9, $10,
				$11, 'draft', 'draft',
				$12, $13, NOW()
			)
			ON CONFLICT (recommendation_id) DO NOTHING
		`,
			recID, a.AlertID, a.RuleID,
			title, detail,
			a.ObjectType, a.ObjectID,
			impact, riskLevel, confidence,
			requiresApproval, a.OwnerRole, successMetric,
		)
		if err != nil {
			return 0, fmt.Errorf("insert recommendation for alert %s: %w", a.AlertID, err)
		}
		inserted++
	}

	if logger != nil {
		logger.Info("recommendations generated", zap.Int64("inserted", inserted))
	}

	return inserted, nil
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

func ptrFmt6d(p *float64) string {
	if p == nil {
		return "0.0000"
	}
	return fmt.Sprintf("%.4f", *p)
}

func ptrFmt2d(p *float64) string {
	if p == nil {
		return "0.00"
	}
	return fmt.Sprintf("%.2f", *p)
}

func ptrFmtInt(p *int64) string {
	if p == nil {
		return "0"
	}
	return fmt.Sprintf("%d", *p)
}

func orElseFmt[T float64](p *T, fallback string, fn func(T) string) string {
	if p == nil {
		return fallback
	}
	return fn(*p)
}

func truncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return strings.TrimSpace(s[:maxLen])
}

// roundTo rounds v to the specified number of decimal places.
func roundTo(v float64, decimals int) float64 {
	pow := math.Pow10(decimals)
	return math.Round(v*pow) / pow
}

// deriveTaskID transforms recommendation_id to task_id:
//   - "rec-xyz"    → "task-xyz"
//   - "dimrec-xyz" → "dimtask-xyz"
//   - fallback:     "task-" + recID
func deriveTaskID(recID string) string {
	if strings.HasPrefix(recID, "dimrec-") {
		return "dimtask-" + recID[7:]
	}
	if strings.HasPrefix(recID, "rec-") {
		return "task-" + recID[4:]
	}
	return "task-" + recID
}

// deriveTaskSource returns the task source based on the recommendation_id prefix.
// Dimensional recommendations (dimrec-) → "dimensional_rule".
// Global recommendations (rec-) → "heuristic_strategy".
func deriveTaskSource(recID string) string {
	if strings.HasPrefix(recID, "dimrec-") {
		return "dimensional_rule"
	}
	return "heuristic_strategy"
}

// derivePriority maps the recommendation risk_level to task priority.
func derivePriority(riskLevel string) string {
	switch riskLevel {
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}
