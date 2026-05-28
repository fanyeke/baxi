package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

// Engine evaluates alert rules against mart.metric_daily data.
type Engine struct{}

// NewEngine creates a new alert rule Engine.
func NewEngine() *Engine {
	return &Engine{}
}

// metricRow holds one day of metric_daily data.
type metricRow struct {
	Date             string
	GMV              float64
	LateDeliveryRate float64
	CancelRate       float64
	AvgReviewScore   float64
	OrderCount       int64
}

// EvaluateGlobalRules iterates all enabled global rules, queries the
// mart.metric_daily table, evaluates each rule against the time series,
// and returns any triggered AlertResult values.
//
// Dead rules (Enabled=false) are silently skipped.
func (e *Engine) EvaluateGlobalRules(ctx context.Context, tx pgx.Tx) ([]AlertResult, error) {
	rules := GlobalRules()

	var results []AlertResult
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.Condition == nil {
			continue
		}

		lastDate, err := e.getLatestDate(ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("%s: get latest date: %w", rule.RuleID, err)
		}

		result, err := rule.Condition(ctx, tx, lastDate)
		if err != nil {
			return nil, fmt.Errorf("%s: evaluate: %w", rule.RuleID, err)
		}
		if result != nil {
			result.RuleID = rule.RuleID
			result.Severity = rule.Severity
			result.MetricName = rule.Metric
			result.EventDate = lastDate
			result.AlertID = GlobalAlertID(rule.RuleID, lastDate)
			results = append(results, *result)
		}
	}

	return results, nil
}

// getLatestDate returns the most recent metric_date in mart.metric_daily.
func (e *Engine) getLatestDate(ctx context.Context, tx pgx.Tx) (string, error) {
	var date time.Time
	err := tx.QueryRow(ctx, `SELECT MAX(metric_date) FROM mart.metric_daily`).Scan(&date)
	if err != nil {
		return "", fmt.Errorf("query max metric_date: %w", err)
	}
	if date.IsZero() {
		return "", fmt.Errorf("mart.metric_daily is empty")
	}
	return date.Format("2006-01-02"), nil
}

// queryMetricSeries returns the metric values for the last N days up to
// and including the given date.
func (e *Engine) queryMetricSeries(ctx context.Context, tx pgx.Tx, metricCol string, latestDate string, n int) ([]float64, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		SELECT %s::TEXT FROM mart.metric_daily
		WHERE metric_date <= $1::DATE
		ORDER BY metric_date DESC
		LIMIT $2
	`, pgx.Identifier{metricCol}.Sanitize()), latestDate, n)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", metricCol, err)
	}
	defer rows.Close()

	var vals []float64
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan %s: %w", metricCol, err)
		}
		var v float64
		if s != "" {
			fmt.Sscanf(s, "%f", &v)
		}
		vals = append(vals, v)
	}
	return vals, rows.Err()
}

// avg computes the mean of a float64 slice. Returns 0 for empty slice.
func avg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// ---------------------------------------------------------------------------
// Rule evaluation functions
// ---------------------------------------------------------------------------

// evaluateGMVDrop fires when the 7-day rolling average GMV drops >= 15 %
// compared to the previous 14-day average.
//
// Python baseline condition (from config/alert_rules.yml):
//
//	current_7d_avg < prev_14d_avg * 0.85
//
// Requires >= 21 days of data (7 + 14).
func evaluateGMVDrop(ctx context.Context, tx pgx.Tx, latestDate string) (*AlertResult, error) {
	series, err := (&Engine{}).queryMetricSeries(ctx, tx, "gmv", latestDate, 21)
	if err != nil {
		return nil, err
	}
	if len(series) < 7 {
		return nil, nil
	}

	cur := series[:7]
	var prev []float64
	if len(series) >= 21 {
		prev = series[7:21]
	} else {
		return nil, nil
	}

	currentAvg := avg(cur)
	prevAvg := avg(prev)

	if prevAvg == 0 {
		return nil, nil
	}

	changeRate := (currentAvg - prevAvg) / prevAvg
	if currentAvg < prevAvg*0.85 {
		evidence := map[string]interface{}{
			"current_7d_avg":   roundTo(currentAvg, 4),
			"prev_14d_avg":     roundTo(prevAvg, 4),
			"current_value":    roundTo(series[0], 4),
			"baseline_value":   roundTo(prevAvg, 4),
			"change_rate":      roundTo(changeRate, 4),
			"window_7d_count":  7,
			"window_14d_count": 14,
			"total_samples":    len(series),
			"triggered":        true,
		}
		evJSON, _ := json.Marshal(evidence)
		msg := fmt.Sprintf("GMV 7日均值较前14天均值下降超过15%% | 7d_avg=%.2f, 14d_avg=%.2f", currentAvg, prevAvg)

		return &AlertResult{
			CurrentValue:  roundTo(currentAvg, 4),
			BaselineValue: roundTo(prevAvg, 4),
			DeltaValue:    roundTo(currentAvg-prevAvg, 4),
			DeltaPct:      roundTo(changeRate, 4),
			Message:       msg,
			EvidenceJSON:  string(evJSON),
			SampleSize:    int64(len(series)),
		}, nil
	}

	return nil, nil
}

// evaluateLateDeliverySpike fires when the latest late_delivery_rate > 0.25.
//
// Python baseline condition (from config/alert_rules.yml):
//
//	value > 0.25 and order_count >= 20
//
// At global scope this rarely triggers -- the overall late delivery rate is
// typically below 0.25 even when regional spikes exist.
func evaluateLateDeliverySpike(ctx context.Context, tx pgx.Tx, latestDate string) (*AlertResult, error) {
	eng := &Engine{}
	series, err := eng.queryMetricSeries(ctx, tx, "late_delivery_rate", latestDate, 21)
	if err != nil {
		return nil, err
	}
	if len(series) < 7 {
		return nil, nil
	}

	latestVal := series[0]

	orderSeries, err := eng.queryMetricSeries(ctx, tx, "order_count", latestDate, 1)
	if err != nil {
		return nil, err
	}
	totalSamples := len(series)
	if len(orderSeries) > 0 {
		totalSamples = int(orderSeries[0])
	}

	if latestVal > 0.25 && totalSamples >= 20 {
		var prevAvg float64
		if len(series) >= 21 {
			prevAvg = avg(series[7:21])
		} else {
			prevAvg = latestVal
		}
		changeRate := 0.0
		if prevAvg != 0 {
			changeRate = (latestVal - prevAvg) / prevAvg
		}

		evidence := map[string]interface{}{
			"current_value":    roundTo(latestVal, 4),
			"baseline_value":   roundTo(prevAvg, 4),
			"change_rate":      roundTo(changeRate, 4),
			"window_7d_count":  7,
			"window_14d_count": 14,
			"total_samples":    totalSamples,
			"triggered":        true,
		}
		evJSON, _ := json.Marshal(evidence)
		msg := fmt.Sprintf("延迟配送率超过25%% | value=%.4f, samples=%d", latestVal, totalSamples)

		return &AlertResult{
			CurrentValue:  roundTo(latestVal, 4),
			BaselineValue: roundTo(prevAvg, 4),
			DeltaValue:    roundTo(latestVal-prevAvg, 4),
			DeltaPct:      roundTo(changeRate, 4),
			Message:       msg,
			EvidenceJSON:  string(evJSON),
			SampleSize:    int64(totalSamples),
		}, nil
	}

	return nil, nil
}

// evaluateCancelRateSpike fires when |change_rate| > 0.5 AND value > 0.05.
//
// Python baseline condition (from config/alert_rules.yml):
//
//	change_rate > 0.5 and value > 0.05
//
// At global scope the overall cancel rate (~1-2 %) stays below 5 %, so this
// rule typically does not trigger.
func evaluateCancelRateSpike(ctx context.Context, tx pgx.Tx, latestDate string) (*AlertResult, error) {
	eng := &Engine{}
	series, err := eng.queryMetricSeries(ctx, tx, "cancel_rate", latestDate, 21)
	if err != nil {
		return nil, err
	}
	if len(series) < 7 {
		return nil, nil
	}

	latestVal := series[0]

	currentAvg := avg(series[:7])
	var prevAvg float64
	if len(series) >= 21 {
		prevAvg = avg(series[7:21])
	} else {
		prevAvg = latestVal
	}

	changeRate := 0.0
	if prevAvg != 0 {
		changeRate = (currentAvg - prevAvg) / prevAvg
	}

	if absFloat(changeRate) > 0.5 && latestVal > 0.05 {
		evidence := map[string]interface{}{
			"current_7d_avg":   roundTo(currentAvg, 4),
			"prev_14d_avg":     roundTo(prevAvg, 4),
			"current_value":    roundTo(latestVal, 4),
			"baseline_value":   roundTo(prevAvg, 4),
			"change_rate":      roundTo(changeRate, 4),
			"window_7d_count":  7,
			"window_14d_count": 14,
			"total_samples":    len(series),
			"triggered":        true,
		}
		evJSON, _ := json.Marshal(evidence)
		msg := fmt.Sprintf("取消率变化超过50%%且当前值超过5%% | change_rate=%.4f, value=%.4f", changeRate, latestVal)

		return &AlertResult{
			CurrentValue:  roundTo(latestVal, 4),
			BaselineValue: roundTo(prevAvg, 4),
			DeltaValue:    roundTo(currentAvg-prevAvg, 4),
			DeltaPct:      roundTo(changeRate, 4),
			Message:       msg,
			EvidenceJSON:  string(evJSON),
			SampleSize:    int64(len(series)),
		}, nil
	}

	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func roundTo(v float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int64(v*pow+0.5)) / pow
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// =========================================================================
// Dimensional Rule Engine
// =========================================================================

// severityWeights maps severity strings to impact score multipliers.
// Matches the Python baseline: high=3, medium=2, low=1.
var severityWeights = map[string]float64{
	"high":   3.0,
	"medium": 2.0,
	"low":    1.0,
}

// severityOrder defines sort priority for alert ordering.
var severityOrder = map[string]int{
	"high":   0,
	"medium": 1,
	"low":    2,
}

// Default limits for dimensional alert suppression.
const (
	DefaultMaxAlertsPerRun    = 50
	DefaultMaxAlertsPerDimVal = 5
)

// DimensionalRuleConfig mirrors config/dimensional_alert_rules.yml.
type DimensionalRuleConfig struct {
	RuleID         string
	DimensionType  string
	MetricName     string
	Condition      string
	MinSampleSize  int
	Severity       string
	OwnerRole      string
	TargetChannel  string
	Description    string
	BaselineWindow int
}

// DimensionalAlert represents a triggered dimensional alert.
type DimensionalAlert struct {
	AlertID        string
	RuleID         string
	EventDate      string
	Severity       string
	MetricName     string
	ObjectType     string
	ObjectID       string
	CurrentValue   float64
	BaselineValue  *float64
	ChangeRate     float64
	SampleSize     int64
	AffectedOrders int64
	AffectedGMV    *float64
	ImpactScore    float64
	Description    string
	OwnerRole      string
}

// makeDimAlertID generates "dim-" + SHA-256(rule\xffdate\xfftype\xffvalue)[:12].
func makeDimAlertID(ruleID, metricDate, dimType, dimValue string) string {
	return "dim-" + GenerateAlertID(ruleID, metricDate, dimType, dimValue)
}

// evaluateCondition evaluates a dimensional rule condition.
func evaluateCondition(condition string, currentValue float64, baselineValue *float64, changeRate float64) bool {
	var threshold float64
	if _, err := fmt.Sscanf(condition, "value_gt: %f", &threshold); err == nil {
		return currentValue > threshold
	}
	if _, err := fmt.Sscanf(condition, "value_lt: %f", &threshold); err == nil {
		return currentValue < threshold
	}
	if _, err := fmt.Sscanf(condition, "change_rate_lt: %f", &threshold); err == nil {
		if baselineValue == nil || *baselineValue == 0 {
			return false
		}
		return changeRate < threshold
	}
	if _, err := fmt.Sscanf(condition, "change_rate_gt: %f", &threshold); err == nil {
		if baselineValue == nil || *baselineValue == 0 {
			return false
		}
		return changeRate > threshold
	}
	return false
}

// ExecuteDimensionalRule evaluates a single dimensional rule against
// mart.metric_dimension_daily and returns triggered alerts. Per-dim-value
// capping at DefaultMaxAlertsPerDimVal is applied.
func ExecuteDimensionalRule(ctx context.Context, tx pgx.Tx, rule DimensionalRuleConfig) ([]DimensionalAlert, error) {
	rows, err := tx.Query(ctx, `
		SELECT metric_date::TEXT, dimension_value, metric_value, sample_size
		FROM mart.metric_dimension_daily
		WHERE dimension_type = $1
		  AND metric_name = $2
		  AND dimension_value IS NOT NULL
		  AND dimension_value != ''
		ORDER BY dimension_value, metric_date DESC
	`, rule.DimensionType, rule.MetricName)
	if err != nil {
		return nil, fmt.Errorf("query %s/%s: %w", rule.DimensionType, rule.MetricName, err)
	}
	defer rows.Close()

	type seriesEntry struct {
		metricDate  string
		metricValue float64
		sampleSize  int64
	}
	seriesByDim := make(map[string][]seriesEntry)
	for rows.Next() {
		var metricDate, dimValue string
		var metricValue float64
		var sampleSize int64
		if err := rows.Scan(&metricDate, &dimValue, &metricValue, &sampleSize); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		seriesByDim[dimValue] = append(seriesByDim[dimValue],
			seriesEntry{metricDate: metricDate, metricValue: metricValue, sampleSize: sampleSize})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	// Check if rule needs change_rate baseline
	needsBaseline := false
	baselineWindow := rule.BaselineWindow
	if baselineWindow <= 0 {
		baselineWindow = 14
	}
	var tmp float64
	if _, err := fmt.Sscanf(rule.Condition, "change_rate_lt: %f", &tmp); err == nil {
		needsBaseline = true
	}
	if !needsBaseline {
		if _, err := fmt.Sscanf(rule.Condition, "change_rate_gt: %f", &tmp); err == nil {
			needsBaseline = true
		}
	}

	var alerts []DimensionalAlert
	for dimValue, series := range seriesByDim {
		alertsPerDim := 0
		for i, entry := range series {
			if alertsPerDim >= DefaultMaxAlertsPerDimVal {
				break
			}
			if entry.sampleSize < int64(rule.MinSampleSize) {
				continue
			}
			currentValue := entry.metricValue
			var baselineValue *float64
			changeRate := 0.0
			if needsBaseline {
				end := i + 1 + baselineWindow
				if end > len(series) {
					end = len(series)
				}
				var sum float64
				var count int
				for j := i + 1; j < end; j++ {
					sum += series[j].metricValue
					count++
				}
				if count > 0 {
					avg := sum / float64(count)
					baselineValue = &avg
					if avg != 0 {
						changeRate = (currentValue - avg) / avg
					}
				}
			}
			triggered := evaluateCondition(rule.Condition, currentValue, baselineValue, changeRate)
			if !triggered {
				continue
			}
			alertsPerDim++
			var bv *float64
			if baselineValue != nil {
				v := *baselineValue
				bv = &v
			}
			alerts = append(alerts, DimensionalAlert{
				AlertID:       makeDimAlertID(rule.RuleID, entry.metricDate, rule.DimensionType, dimValue),
				RuleID:        rule.RuleID,
				EventDate:     entry.metricDate,
				Severity:      rule.Severity,
				MetricName:    rule.MetricName,
				ObjectType:    rule.DimensionType,
				ObjectID:      dimValue,
				CurrentValue:  currentValue,
				BaselineValue: bv,
				ChangeRate:    changeRate,
				SampleSize:    entry.sampleSize,
				Description:   rule.Description,
				OwnerRole:     rule.OwnerRole,
			})
		}
	}
	return alerts, nil
}

// enrichDimAlerts enriches alerts with affected_gmv and impact_score.
func enrichDimAlerts(ctx context.Context, tx pgx.Tx, alerts []DimensionalAlert) error {
	for i := range alerts {
		a := &alerts[i]
		var gmv float64
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(metric_value, 0)
			FROM mart.metric_dimension_daily
			WHERE metric_date = $1
			  AND dimension_type = $2
			  AND dimension_value = $3
			  AND metric_name = 'gmv'
		`, a.EventDate, a.ObjectType, a.ObjectID).Scan(&gmv)
		if err != nil && err != pgx.ErrNoRows {
			return fmt.Errorf("query gmv for %s: %w", a.AlertID, err)
		}
		if err == nil && gmv != 0 {
			a.AffectedGMV = &gmv
		}
		a.AffectedOrders = a.SampleSize
		sw := severityWeights[a.Severity]
		if sw == 0 {
			sw = 1.0
		}
		a.ImpactScore = sw * float64(a.SampleSize)
	}
	return nil
}

// SuppressResult holds suppression output.
type SuppressResult struct {
	Alerts     []DimensionalAlert
	Suppressed int
}

// SuppressAlerts sorts by severity→impact→sample_size and caps at maxAlerts.
// Per-dim-value capping (max 5) is done in ExecuteDimensionalRule.
func SuppressAlerts(alerts []DimensionalAlert, maxAlerts int) SuppressResult {
	if maxAlerts <= 0 {
		maxAlerts = DefaultMaxAlertsPerRun
	}
	if len(alerts) == 0 {
		return SuppressResult{Alerts: alerts, Suppressed: 0}
	}
	sort.Slice(alerts, func(i, j int) bool {
		si := severityOrder[alerts[i].Severity]
		sj := severityOrder[alerts[j].Severity]
		if si != sj {
			return si < sj
		}
		if alerts[i].ImpactScore != alerts[j].ImpactScore {
			return alerts[i].ImpactScore > alerts[j].ImpactScore
		}
		return alerts[i].SampleSize > alerts[j].SampleSize
	})
	if len(alerts) <= maxAlerts {
		return SuppressResult{Alerts: alerts, Suppressed: 0}
	}
	return SuppressResult{
		Alerts:     alerts[:maxAlerts],
		Suppressed: len(alerts) - maxAlerts,
	}
}

// DefaultDimensionalRules returns 6 rules from dimensional_alert_rules.yml.
func DefaultDimensionalRules() []DimensionalRuleConfig {
	return []DimensionalRuleConfig{
		{
			RuleID:        "seller_late_delivery_spike",
			DimensionType: "seller",
			MetricName:    "late_delivery_rate",
			Condition:     "value_gt: 0.25",
			MinSampleSize: 20,
			Severity:      "high",
			OwnerRole:     "seller_ops",
			TargetChannel: "feishu_cli",
			Description:   "卖家延迟配送率超过25%且样本>=20单",
		},
		{
			RuleID:        "seller_review_score_drop",
			DimensionType: "seller",
			MetricName:    "avg_review_score",
			Condition:     "value_lt: 3.5",
			MinSampleSize: 20,
			Severity:      "medium",
			OwnerRole:     "seller_ops",
			TargetChannel: "local_cli",
			Description:   "卖家评分低于3.5且样本>=20单",
		},
		{
			RuleID:         "category_gmv_drop",
			DimensionType:  "category",
			MetricName:     "gmv",
			Condition:      "change_rate_lt: -0.20",
			MinSampleSize:  30,
			Severity:       "medium",
			OwnerRole:      "category_ops",
			TargetChannel:  "feishu_cli",
			Description:    "品类GMV环比下降超过20%且样本>=30单",
			BaselineWindow: 14,
		},
		{
			RuleID:        "category_low_review_cluster",
			DimensionType: "category",
			MetricName:    "low_review_rate",
			Condition:     "value_gt: 0.15",
			MinSampleSize: 30,
			Severity:      "medium",
			OwnerRole:     "category_ops",
			TargetChannel: "local_cli",
			Description:   "品类差评率超过15%且样本>=30单",
		},
		{
			RuleID:        "region_cancel_rate_spike",
			DimensionType: "region",
			MetricName:    "cancel_rate",
			Condition:     "value_gt: 0.05",
			MinSampleSize: 30,
			Severity:      "medium",
			OwnerRole:     "logistics_ops",
			TargetChannel: "manual",
			Description:   "区域取消率超过5%且样本>=30单",
		},
		{
			RuleID:        "region_late_delivery_spike",
			DimensionType: "region",
			MetricName:    "late_delivery_rate",
			Condition:     "value_gt: 0.20",
			MinSampleSize: 30,
			Severity:      "high",
			OwnerRole:     "logistics_ops",
			TargetChannel: "feishu_cli",
			Description:   "区域延迟配送率超过20%且样本>=30单",
		},
	}
}

// EvaluateDimensionRules runs all dimensional rules, enriches, suppresses.
// tx must NOT be committed/rolled back by this function.
func EvaluateDimensionRules(ctx context.Context, tx pgx.Tx, rules []DimensionalRuleConfig, maxAlertsPerRun int) ([]DimensionalAlert, int, error) {
	if len(rules) == 0 {
		rules = DefaultDimensionalRules()
	}
	if maxAlertsPerRun <= 0 {
		maxAlertsPerRun = DefaultMaxAlertsPerRun
	}
	var allAlerts []DimensionalAlert
	for _, rule := range rules {
		alerts, err := ExecuteDimensionalRule(ctx, tx, rule)
		if err != nil {
			return nil, 0, fmt.Errorf("rule %s: %w", rule.RuleID, err)
		}
		allAlerts = append(allAlerts, alerts...)
	}
	if err := enrichDimAlerts(ctx, tx, allAlerts); err != nil {
		return nil, 0, fmt.Errorf("enrich: %w", err)
	}
	result := SuppressAlerts(allAlerts, maxAlertsPerRun)
	return result.Alerts, result.Suppressed, nil
}

// CountAlertsByDimension returns object_type → count map.
func CountAlertsByDimension(alerts []DimensionalAlert) map[string]int {
	counts := make(map[string]int)
	for _, a := range alerts {
		counts[a.ObjectType]++
	}
	return counts
}
