// Package dto provides data transfer objects for API responses.
package dto

// AlertItem represents a single alert event in API responses.
// Fields map from ops.metric_alert with backward-compatible naming.
type AlertItem struct {
	EventID       string   `json:"event_id"`
	RuleID        string   `json:"rule_id"`
	EventDate     string   `json:"event_date"`
	Severity      string   `json:"severity"`
	MetricName    string   `json:"metric_name"`
	ObjectType    string   `json:"object_type"`
	ObjectID      string   `json:"object_id"`
	CurrentValue  *float64 `json:"current_value"`
	BaselineValue *float64 `json:"baseline_value"`
	ChangeRate    *float64 `json:"change_rate"`
	OwnerRole     string   `json:"owner_role"`
	Status        string   `json:"status"`
	ImpactScore   *float64 `json:"impact_score"`
}

// AlertFilters represents query filter parameters for alert listing.
type AlertFilters struct {
	Severity   string
	Status     string
	ObjectType string
	RuleID     string
}

// AlertListResponse is the backward-compatible paginated response.
// Uses {"items": [...], "total": N} format matching the Phase 0 baseline.
type AlertListResponse struct {
	Items []AlertItem `json:"items"`
	Total int         `json:"total"`
}
