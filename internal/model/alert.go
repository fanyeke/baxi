package model

// Alert represents a single alert event.
type Alert struct {
	EventID       string
	RuleID        string
	EventDate     string
	Severity      string
	MetricName    string
	ObjectType    string
	ObjectID      string
	CurrentValue  *float64
	BaselineValue *float64
	ChangeRate    *float64
	OwnerRole     string
	Status        string
	ImpactScore   *float64
}

// AlertFilters represents query filter parameters for alert listing.
type AlertFilters struct {
	Severity   string
	Status     string
	ObjectType string
	RuleID     string
}

// AlertListResponse is the paginated response for alert listing.
type AlertListResponse struct {
	Items []Alert
	Total int
}
