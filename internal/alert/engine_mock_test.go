package alert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMockTxForDate returns a mockTx that returns a fixed date for the getLatestDate query.
func newMockTxForDate(dateStr string) *mockTx {
	return &mockTx{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					if len(dest) > 0 {
						t, err := time.Parse("2006-01-02", dateStr)
						if err != nil {
							return err
						}
						if tp, ok := dest[0].(*time.Time); ok {
							*tp = t
						}
					}
					return nil
				},
			}
		},
	}
}

// newMockTxForMetricSeries returns a mockTx that returns float64 values for queryMetricSeries.
func newMockTxForMetricSeries(values []float64) *mockTx {
	return newMockTxForMultiMetric(map[string][]float64{"__default__": values})
}

// metricData holds per-metric values and order count for mock tests.
type metricData struct {
	values     []float64
	orderCount int64
}

// newMockTxForMultiMetric returns a mockTx that returns different data per metric name.
func newMockTxForMultiMetric(metrics map[string][]float64) *mockTx {
	return &mockTx{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					if tp, ok := dest[0].(*time.Time); ok {
						*tp = time.Date(2018, 10, 21, 0, 0, 0, 0, time.UTC)
					}
					return nil
				},
			}
		},
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			// Detect metric name from SQL
			metricName := "__default__"
			for _, m := range []string{"gmv", "late_delivery_rate", "cancel_rate", "order_count", "avg_review_score"} {
				if containsSubstring(sql, m) {
					metricName = m
					break
				}
			}
			vals, ok := metrics[metricName]
			if !ok {
				vals, ok = metrics["__default__"]
				if !ok {
					return &mockRows{data: [][]interface{}{}}, nil
				}
			}
			rows := &mockRows{data: make([][]interface{}, len(vals))}
			for i, v := range vals {
				rows.data[i] = []interface{}{fmt.Sprintf("%f", v)}
			}
			return rows, nil
		},
	}
}

// newMockTxForDimensionalRule returns a mockTx with configurable responses for dimensional rules.
func newMockTxForDimensionalRule(dimRows []dimRowData, gmvRows []gmvRowData) *mockTx {
	return &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if len(dimRows) > 0 && containsStr(sql, "metric_dimension_daily") && !containsStr(sql, "gmv") {
				rows := &mockRows{
					data: make([][]interface{}, len(dimRows)),
				}
				for i, dr := range dimRows {
					rows.data[i] = []interface{}{dr.date, dr.dimValue, dr.metricValue, dr.sampleSize}
				}
				return rows, nil
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			// For enrichDimAlerts GMV lookup
			if containsStr(sql, "gmv") {
				for _, g := range gmvRows {
					return &mockRow{
						scanFn: func(dest ...any) error {
							if len(dest) > 0 {
								if fp, ok := dest[0].(*float64); ok {
									*fp = g.gmv
								}
							}
							return nil
						},
					}
				}
			}
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}
}

type dimRowData struct {
	date        string
	dimValue    string
	metricValue float64
	sampleSize  int64
}

type gmvRowData struct {
	gmv float64
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- getLatestDate tests ---

func TestGetLatestDate_Success(t *testing.T) {
	tx := newMockTxForDate("2018-10-21")
	e := NewEngine()
	date, err := e.getLatestDate(context.Background(), tx)
	require.NoError(t, err)
	assert.Equal(t, "2018-10-21", date)
}

func TestGetLatestDate_ZeroDate(t *testing.T) {
	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					// Leave time.Time as zero value
					return nil
				},
			}
		},
	}
	e := NewEngine()
	_, err := e.getLatestDate(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestGetLatestDate_QueryError(t *testing.T) {
	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error {
				return fmt.Errorf("db connection lost")
			}}
		},
	}
	e := NewEngine()
	_, err := e.getLatestDate(context.Background(), tx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query max metric_date")
}

// --- queryMetricSeries tests ---

func TestQueryMetricSeries_Success(t *testing.T) {
	vals := []float64{100.0, 200.0, 300.0}
	tx := newMockTxForMetricSeries(vals)
	e := NewEngine()
	result, err := e.queryMetricSeries(context.Background(), tx, "gmv", "2018-10-21", 3)
	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.InDelta(t, 100.0, result[0], 0.01)
	assert.InDelta(t, 200.0, result[1], 0.01)
	assert.InDelta(t, 300.0, result[2], 0.01)
}

func TestQueryMetricSeries_Empty(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
	}
	e := NewEngine()
	result, err := e.queryMetricSeries(context.Background(), tx, "gmv", "2018-10-21", 21)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestQueryMetricSeries_QueryError(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, fmt.Errorf("query failed")
		},
	}
	e := NewEngine()
	_, err := e.queryMetricSeries(context.Background(), tx, "gmv", "2018-10-21", 21)
	assert.Error(t, err)
}

func TestQueryMetricSeries_EmptyStringValues(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{{""}, {"5.5"}}}, nil
		},
	}
	e := NewEngine()
	result, err := e.queryMetricSeries(context.Background(), tx, "gmv", "2018-10-21", 2)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.InDelta(t, 0.0, result[0], 0.01) // empty string -> 0
	assert.InDelta(t, 5.5, result[1], 0.01)
}

// --- evaluateGMVDrop tests ---

func TestEvaluateGMVDrop_TriggersOnBigDrop(t *testing.T) {
	// 14 days baseline at 1000, 7 days current at 100 => 90% drop > 15%
	vals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		vals[i] = 100.0 // current 7d
	}
	for i := 7; i < 21; i++ {
		vals[i] = 1000.0 // prev 14d
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.InDelta(t, 100.0, result.CurrentValue, 1.0)
	assert.InDelta(t, 1000.0, result.BaselineValue, 1.0)
	assert.Less(t, result.DeltaPct, -0.15)
	assert.True(t, result.SampleSize > 0)
}

func TestEvaluateGMVDrop_NoTriggerStableData(t *testing.T) {
	// All values equal => no drop
	vals := make([]float64, 21)
	for i := range vals {
		vals[i] = 500.0
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateGMVDrop_NoTriggerSmallDrop(t *testing.T) {
	// 7d avg = 900, prev avg = 1000 => 10% drop < 15%
	vals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		vals[i] = 900.0
	}
	for i := 7; i < 21; i++ {
		vals[i] = 1000.0
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateGMVDrop_InsufficientData(t *testing.T) {
	vals := []float64{100, 200, 300} // Only 3 days
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateGMVDrop_Between7And21Days(t *testing.T) {
	// 10 days total (< 21) => returns nil
	vals := make([]float64, 10)
	for i := range vals {
		vals[i] = 100.0
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateGMVDrop_Exactly7Days(t *testing.T) {
	vals := make([]float64, 7)
	for i := range vals {
		vals[i] = 100.0
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result) // Not enough for 14-day baseline
}

func TestEvaluateGMVDrop_ZeroBaseline(t *testing.T) {
	vals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		vals[i] = 100.0
	}
	for i := 7; i < 21; i++ {
		vals[i] = 0.0 // zero baseline
	}
	tx := newMockTxForMetricSeries(vals)
	result, err := evaluateGMVDrop(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result) // prevAvg == 0 => no trigger
}

// --- evaluateLateDeliverySpike tests ---

func TestEvaluateLateDeliverySpike_Triggers(t *testing.T) {
	// Late delivery rate > 0.25 with order_count >= 20
	lateVals := make([]float64, 21)
	lateVals[0] = 0.30 // latest > 0.25
	for i := 1; i < 21; i++ {
		lateVals[i] = 0.10
	}

	tx := &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if containsSubstring(sql, "late_delivery_rate") {
				rows := &mockRows{data: make([][]interface{}, len(lateVals))}
				for i, v := range lateVals {
					rows.data[i] = []interface{}{fmt.Sprintf("%f", v)}
				}
				return rows, nil
			}
			if containsSubstring(sql, "order_count") {
				return &mockRows{data: [][]interface{}{{"50"}}}, nil
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	result, err := evaluateLateDeliverySpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.CurrentValue, 0.25)
}

func TestEvaluateLateDeliverySpike_NoTrigger(t *testing.T) {
	lateVals := make([]float64, 21)
	lateVals[0] = 0.08 // below 0.25
	for i := 1; i < 21; i++ {
		lateVals[i] = 0.08
	}

	tx := &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if containsSubstring(sql, "late_delivery_rate") {
				rows := &mockRows{data: make([][]interface{}, len(lateVals))}
				for i, v := range lateVals {
					rows.data[i] = []interface{}{fmt.Sprintf("%f", v)}
				}
				return rows, nil
			}
			if containsSubstring(sql, "order_count") {
				return &mockRows{data: [][]interface{}{{"100"}}}, nil
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	result, err := evaluateLateDeliverySpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateLateDeliverySpike_LowOrderCount(t *testing.T) {
	lateVals := make([]float64, 21)
	lateVals[0] = 0.30
	for i := 1; i < 21; i++ {
		lateVals[i] = 0.10
	}

	tx := &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if containsSubstring(sql, "late_delivery_rate") {
				rows := &mockRows{data: make([][]interface{}, len(lateVals))}
				for i, v := range lateVals {
					rows.data[i] = []interface{}{fmt.Sprintf("%f", v)}
				}
				return rows, nil
			}
			if containsSubstring(sql, "order_count") {
				return &mockRows{data: [][]interface{}{{"5"}}}, nil // < 20
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	result, err := evaluateLateDeliverySpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateLateDeliverySpike_InsufficientData(t *testing.T) {
	lateVals := []float64{0.30, 0.28, 0.26} // only 3 days < 7
	tx := &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if containsSubstring(sql, "late_delivery_rate") {
				rows := &mockRows{data: make([][]interface{}, len(lateVals))}
				for i, v := range lateVals {
					rows.data[i] = []interface{}{fmt.Sprintf("%f", v)}
				}
				return rows, nil
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}
	result, err := evaluateLateDeliverySpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

// --- evaluateCancelRateSpike tests ---

func TestEvaluateCancelRateSpike_Triggers(t *testing.T) {
	// |change_rate| > 0.5 AND value > 0.05
	// current 7d avg = 0.10, prev 14d avg = 0.02 => change_rate = 4.0
	cancelVals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		cancelVals[i] = 0.10
	}
	for i := 7; i < 21; i++ {
		cancelVals[i] = 0.02
	}

	tx := newMockTxForMetricSeries(cancelVals)
	result, err := evaluateCancelRateSpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.CurrentValue, 0.05)
}

func TestEvaluateCancelRateSpike_NoTriggerLowRate(t *testing.T) {
	cancelVals := make([]float64, 21)
	for i := range cancelVals {
		cancelVals[i] = 0.02
	}

	tx := newMockTxForMetricSeries(cancelVals)
	result, err := evaluateCancelRateSpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateCancelRateSpike_NoTriggerSmallChange(t *testing.T) {
	// change_rate = (100-90)/90 = 0.11 < 0.5
	cancelVals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		cancelVals[i] = 100.0
	}
	for i := 7; i < 21; i++ {
		cancelVals[i] = 90.0
	}

	tx := newMockTxForMetricSeries(cancelVals)
	result, err := evaluateCancelRateSpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateCancelRateSpike_InsufficientData(t *testing.T) {
	cancelVals := []float64{0.10, 0.10, 0.10}
	tx := newMockTxForMetricSeries(cancelVals)
	result, err := evaluateCancelRateSpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestEvaluateCancelRateSpike_Between7And21Days(t *testing.T) {
	cancelVals := make([]float64, 10)
	for i := 0; i < 7; i++ {
		cancelVals[i] = 0.10
	}
	for i := 7; i < 10; i++ {
		cancelVals[i] = 0.02
	}

	tx := newMockTxForMetricSeries(cancelVals)
	result, err := evaluateCancelRateSpike(context.Background(), tx, "2018-10-21")
	require.NoError(t, err)
	// Has 10 values: 7 current, 3 baseline (< 14). But len >= 21 is false, so prevAvg = latestVal = 0.10
	// change_rate = (0.10 - 0.10) / 0.10 = 0.0 < 0.5
	assert.Nil(t, result)
}

// --- EvaluateGlobalRules tests ---

func TestEvaluateGlobalRules_EmptyTable(t *testing.T) {
	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					// Return zero time => "empty" error
					return nil
				},
			}
		},
	}
	e := NewEngine()
	_, err := e.EvaluateGlobalRules(context.Background(), tx)
	assert.Error(t, err)
}

func TestEvaluateGlobalRules_WithDateNoRulesTriggered(t *testing.T) {
	// Stable data -> no rules triggered
	stableVals := make([]float64, 21)
	for i := range stableVals {
		stableVals[i] = 500.0
	}
	lateDeliveryVals := make([]float64, 21)
	for i := range lateDeliveryVals {
		lateDeliveryVals[i] = 0.08
	}
	cancelRateVals := make([]float64, 21)
	for i := range cancelRateVals {
		cancelRateVals[i] = 0.02
	}

	tx := newMockTxForMultiMetric(map[string][]float64{
		"gmv":               stableVals,
		"late_delivery_rate": lateDeliveryVals,
		"cancel_rate":       cancelRateVals,
	})

	e := NewEngine()
	results, err := e.EvaluateGlobalRules(context.Background(), tx)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestEvaluateGlobalRules_GMVDropTriggered(t *testing.T) {
	gmvVals := make([]float64, 21)
	for i := 0; i < 7; i++ {
		gmvVals[i] = 100.0
	}
	for i := 7; i < 21; i++ {
		gmvVals[i] = 1000.0
	}
	lateDeliveryVals := make([]float64, 21)
	for i := range lateDeliveryVals {
		lateDeliveryVals[i] = 0.08
	}
	cancelRateVals := make([]float64, 21)
	for i := range cancelRateVals {
		cancelRateVals[i] = 0.02
	}

	tx := newMockTxForMultiMetric(map[string][]float64{
		"gmv":               gmvVals,
		"late_delivery_rate": lateDeliveryVals,
		"cancel_rate":       cancelRateVals,
	})

	e := NewEngine()
	results, err := e.EvaluateGlobalRules(context.Background(), tx)
	require.NoError(t, err)
	// gmv_drop should trigger
	foundGMV := false
	for _, r := range results {
		if r.RuleID == "gmv_drop" {
			foundGMV = true
			assert.Equal(t, SeverityHigh, r.Severity)
			assert.Equal(t, "gmv", r.MetricName)
			assert.Equal(t, "2018-10-21", r.EventDate)
			assert.Equal(t, "gmv_drop_2018-10-21", r.AlertID)
		}
	}
	assert.True(t, foundGMV, "gmv_drop rule should have triggered")
}

// --- ExecuteDimensionalRule tests ---

func TestExecuteDimensionalRule_Triggers(t *testing.T) {
	rule := DimensionalRuleConfig{
		RuleID:        "seller_late_delivery_spike",
		DimensionType: "seller",
		MetricName:    "late_delivery_rate",
		Condition:     "value_gt: 0.25",
		MinSampleSize: 20,
		Severity:      "high",
		OwnerRole:     "seller_ops",
	}

	dimRows := []dimRowData{
		{"2018-10-21", "seller-1", 0.30, 50},
		{"2018-10-20", "seller-1", 0.20, 45},
	}
	tx := newMockTxForDimensionalRule(dimRows, nil)

	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Equal(t, "seller-1", alerts[0].ObjectID)
	assert.Equal(t, "seller", alerts[0].ObjectType)
	assert.Equal(t, 0.30, alerts[0].CurrentValue)
}

func TestExecuteDimensionalRule_NoTrigger(t *testing.T) {
	rule := DimensionalRuleConfig{
		RuleID:        "seller_late_delivery_spike",
		DimensionType: "seller",
		MetricName:    "late_delivery_rate",
		Condition:     "value_gt: 0.25",
		MinSampleSize: 20,
		Severity:      "high",
	}

	dimRows := []dimRowData{
		{"2018-10-21", "seller-1", 0.10, 50}, // 0.10 < 0.25
	}
	tx := newMockTxForDimensionalRule(dimRows, nil)

	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestExecuteDimensionalRule_BelowMinSampleSize(t *testing.T) {
	rule := DimensionalRuleConfig{
		RuleID:        "seller_late_delivery_spike",
		DimensionType: "seller",
		MetricName:    "late_delivery_rate",
		Condition:     "value_gt: 0.25",
		MinSampleSize: 20,
		Severity:      "high",
	}

	dimRows := []dimRowData{
		{"2018-10-21", "seller-1", 0.30, 5}, // sample < 20
	}
	tx := newMockTxForDimensionalRule(dimRows, nil)

	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestExecuteDimensionalRule_QueryError(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	rule := DimensionalRuleConfig{RuleID: "test", DimensionType: "seller", MetricName: "late_delivery_rate"}
	_, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	assert.Error(t, err)
}

func TestExecuteDimensionalRule_ChangeRateCondition(t *testing.T) {
	rule := DimensionalRuleConfig{
		RuleID:         "category_gmv_drop",
		DimensionType:  "category",
		MetricName:     "gmv",
		Condition:      "change_rate_lt: -0.20",
		MinSampleSize:  30,
		Severity:       "medium",
		BaselineWindow: 14,
	}

	// 21 days of data with a drop
	dimRows := make([]dimRowData, 21)
	dimRows[0] = dimRowData{"2018-10-21", "electronics", 100.0, 50}
	for i := 1; i < 21; i++ {
		dimRows[i] = dimRowData{
			fmt.Sprintf("2018-10-%02d", 21-i),
			"electronics",
			1000.0,
			50,
		}
	}
	tx := newMockTxForDimensionalRule(dimRows, nil)

	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	require.Len(t, alerts, 1)
	assert.Less(t, alerts[0].ChangeRate, -0.20)
}

func TestExecuteDimensionalRule_PerDimCap(t *testing.T) {
	rule := DimensionalRuleConfig{
		RuleID:        "test_cap",
		DimensionType: "seller",
		MetricName:    "late_delivery_rate",
		Condition:     "value_gt: 0.10",
		MinSampleSize: 1,
		Severity:      "high",
	}

	// 7 entries all triggering, capped at DefaultMaxAlertsPerDimVal=5
	dimRows := make([]dimRowData, 7)
	for i := 0; i < 7; i++ {
		dimRows[i] = dimRowData{
			fmt.Sprintf("2018-10-%02d", 21-i),
			"seller-1",
			0.50,
			10,
		}
	}
	tx := newMockTxForDimensionalRule(dimRows, nil)

	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(alerts), DefaultMaxAlertsPerDimVal)
}

func TestExecuteDimensionalRule_NoRows(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
	}
	rule := DimensionalRuleConfig{RuleID: "test", DimensionType: "seller", MetricName: "late_delivery_rate"}
	alerts, err := ExecuteDimensionalRule(context.Background(), tx, rule)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

// --- enrichDimAlerts tests ---

func TestEnrichDimAlerts_WithGMV(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high", SampleSize: 100, AffectedGMV: nil},
	}

	tx := &mockTx{
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			return &mockRow{
				scanFn: func(dest ...any) error {
					if len(dest) > 0 {
						if fp, ok := dest[0].(*float64); ok {
							*fp = 5000.0
						}
					}
					return nil
				},
			}
		},
	}

	err := enrichDimAlerts(context.Background(), tx, alerts)
	require.NoError(t, err)
	assert.NotNil(t, alerts[0].AffectedGMV)
	assert.Equal(t, 5000.0, *alerts[0].AffectedGMV)
	assert.Equal(t, int64(100), alerts[0].AffectedOrders)
	assert.InDelta(t, 300.0, alerts[0].ImpactScore, 0.01) // 3.0 * 100
}

func TestEnrichDimAlerts_NoGMV(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "medium", SampleSize: 50},
	}

	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	err := enrichDimAlerts(context.Background(), tx, alerts)
	require.NoError(t, err)
	assert.Nil(t, alerts[0].AffectedGMV)
	assert.Equal(t, int64(50), alerts[0].AffectedOrders)
	assert.InDelta(t, 100.0, alerts[0].ImpactScore, 0.01) // 2.0 * 50
}

func TestEnrichDimAlerts_QueryError(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high", SampleSize: 10},
	}

	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return fmt.Errorf("db error") }}
		},
	}

	err := enrichDimAlerts(context.Background(), tx, alerts)
	assert.Error(t, err)
}

func TestEnrichDimAlerts_UnknownSeverity(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "unknown", SampleSize: 10},
	}

	tx := &mockTx{
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	err := enrichDimAlerts(context.Background(), tx, alerts)
	require.NoError(t, err)
	assert.InDelta(t, 10.0, alerts[0].ImpactScore, 0.01) // 1.0 * 10 (default weight)
}

// --- EvaluateDimensionRules tests ---

func TestEvaluateDimensionRules_EmptyRules(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	alerts, suppressed, err := EvaluateDimensionRules(context.Background(), tx, nil, 50)
	require.NoError(t, err)
	assert.Empty(t, alerts)
	assert.Equal(t, 0, suppressed)
}

func TestEvaluateDimensionRules_ZeroMaxAlerts(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{data: [][]interface{}{}}, nil
		},
		queryRowFunc: func(_ context.Context, _ string, _ ...any) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	alerts, _, err := EvaluateDimensionRules(context.Background(), tx, []DimensionalRuleConfig{}, 0)
	require.NoError(t, err)
	assert.Empty(t, alerts)
}

func TestEvaluateDimensionRules_RuleError(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return nil, fmt.Errorf("rule query failed")
		},
	}

	_, _, err := EvaluateDimensionRules(context.Background(), tx, []DimensionalRuleConfig{
		{RuleID: "bad_rule", DimensionType: "seller", MetricName: "test"},
	}, 50)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad_rule")
}

// --- SuppressAlerts edge cases ---

func TestSuppressAlerts_SortBySampleSizeWithinSameImpactScore(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "medium", ImpactScore: 100, SampleSize: 50},
		{AlertID: "a2", Severity: "medium", ImpactScore: 100, SampleSize: 200},
		{AlertID: "a3", Severity: "medium", ImpactScore: 100, SampleSize: 100},
	}
	result := SuppressAlerts(alerts, 3)
	assert.Equal(t, "a2", result.Alerts[0].AlertID) // highest sample size
	assert.Equal(t, "a3", result.Alerts[1].AlertID)
	assert.Equal(t, "a1", result.Alerts[2].AlertID)
}

func TestSuppressAlerts_ExactLimit(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high"},
		{AlertID: "a2", Severity: "low"},
	}
	result := SuppressAlerts(alerts, 2)
	assert.Len(t, result.Alerts, 2)
	assert.Equal(t, 0, result.Suppressed)
}

func TestSuppressAlerts_SingleAlert(t *testing.T) {
	alerts := []DimensionalAlert{
		{AlertID: "a1", Severity: "high", ImpactScore: 50},
	}
	result := SuppressAlerts(alerts, 10)
	assert.Len(t, result.Alerts, 1)
	assert.Equal(t, 0, result.Suppressed)
}

// --- EvaluateDimensionRules enrichment ---

func TestEvaluateDimensionRules_EnrichesAlerts(t *testing.T) {
	tx := &mockTx{
		queryHandler: func(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
			if containsSubstring(sql, "metric_dimension_daily") && !containsSubstring(sql, "gmv") {
				return &mockRows{data: [][]interface{}{
					{"2018-10-21", "seller-1", 0.30, 50},
				}}, nil
			}
			return &mockRows{}, nil
		},
		queryRowFunc: func(_ context.Context, sql string, _ ...any) pgx.Row {
			if containsSubstring(sql, "gmv") {
				return &mockRow{
					scanFn: func(dest ...any) error {
						if fp, ok := dest[0].(*float64); ok {
							*fp = 10000.0
						}
						return nil
					},
				}
			}
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	}

	rules := []DimensionalRuleConfig{{
		RuleID:        "seller_late_delivery_spike",
		DimensionType: "seller",
		MetricName:    "late_delivery_rate",
		Condition:     "value_gt: 0.25",
		MinSampleSize: 20,
		Severity:      "high",
		OwnerRole:     "seller_ops",
	}}

	alerts, _, err := EvaluateDimensionRules(context.Background(), tx, rules, 50)
	require.NoError(t, err)
	if len(alerts) > 0 {
		assert.NotNil(t, alerts[0].AffectedGMV)
	}
}

// --- DefaultDimensionalRules config ---

func TestDefaultDimensionalRules_AllFieldsPopulated(t *testing.T) {
	for _, rule := range DefaultDimensionalRules() {
		assert.NotEmpty(t, rule.RuleID)
		assert.NotEmpty(t, rule.DimensionType)
		assert.NotEmpty(t, rule.MetricName)
		assert.NotEmpty(t, rule.Condition)
		assert.NotEmpty(t, rule.Severity)
		assert.NotEmpty(t, rule.OwnerRole)
		assert.NotEmpty(t, rule.Description)
		assert.Greater(t, rule.MinSampleSize, 0)
	}
}
