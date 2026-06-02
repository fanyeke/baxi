package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"baxi/internal/model"
)

// ──── dtoFromGovernanceStatusResponse ──────────────────────────────────────

func TestDTOFromGovernanceStatusResponse_Full(t *testing.T) {
	m := &model.GovernanceStatusResponse{
		GovernanceLayer:   "active",
		ObjectSchemaCount: 12,
		Configs:           map[string]string{"data_quality.yml": "loaded", "classification.yml": "loaded"},
	}
	d := dtoFromGovernanceStatusResponse(m)
	assert.Equal(t, "active", d.GovernanceLayer)
	assert.Equal(t, 12, d.ObjectSchemaCount)
	assert.Equal(t, "loaded", d.Configs["data_quality.yml"])
	assert.Equal(t, "loaded", d.Configs["classification.yml"])
}

func TestDTOFromGovernanceStatusResponse_Nil(t *testing.T) {
	d := dtoFromGovernanceStatusResponse(nil)
	assert.Nil(t, d)
}

func TestDTOFromGovernanceStatusResponse_Empty(t *testing.T) {
	m := &model.GovernanceStatusResponse{
		GovernanceLayer:   "unknown",
		ObjectSchemaCount: 0,
		Configs:           map[string]string{},
	}
	d := dtoFromGovernanceStatusResponse(m)
	assert.Equal(t, "unknown", d.GovernanceLayer)
	assert.Equal(t, 0, d.ObjectSchemaCount)
	assert.Empty(t, d.Configs)
}

// ──── dtoFromCatalogResponse ───────────────────────────────────────────────

func TestDTOFromCatalogResponse_Full(t *testing.T) {
	m := &model.CatalogResponse{
		Objects: []model.CatalogObject{
			{ObjectType: "order", SourceDataset: "orders", PrimaryKey: "order_id", PropertiesCount: 10, LinksCount: 3},
			{ObjectType: "customer", SourceDataset: "customers", PrimaryKey: "customer_id", PropertiesCount: 8, LinksCount: 2},
		},
		Datasets: []model.CatalogDataset{
			{Dataset: "orders", Schema: "public", Table: "orders"},
			{Dataset: "customers", Schema: "public", Table: "customers"},
		},
	}
	d := dtoFromCatalogResponse(m)
	assert.Len(t, d.Objects, 2)
	assert.Len(t, d.Datasets, 2)
	assert.Equal(t, "order", d.Objects[0].ObjectType)
	assert.Equal(t, "orders", d.Objects[0].SourceDataset)
	assert.Equal(t, "public", d.Datasets[0].Schema)
}

func TestDTOFromCatalogResponse_Empty(t *testing.T) {
	m := &model.CatalogResponse{
		Objects:  []model.CatalogObject{},
		Datasets: []model.CatalogDataset{},
	}
	d := dtoFromCatalogResponse(m)
	assert.Empty(t, d.Objects)
	assert.Empty(t, d.Datasets)
}

func TestDTOFromCatalogResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromCatalogResponse(nil))
}

// ──── dtoFromClassificationResponse ────────────────────────────────────────

func TestDTOFromClassificationResponse_Full(t *testing.T) {
	m := &model.ClassificationResponse{
		Levels: []string{"public", "internal", "confidential"},
		Resources: []model.ClassificationResource{
			{Resource: "orders.order_id", Classification: "confidential"},
			{Resource: "orders.status", Classification: "public"},
		},
	}
	d := dtoFromClassificationResponse(m)
	assert.Equal(t, []string{"public", "internal", "confidential"}, d.Levels)
	assert.Len(t, d.Resources, 2)
	assert.Equal(t, "orders.order_id", d.Resources[0].Resource)
	assert.Equal(t, "confidential", d.Resources[0].Classification)
}

func TestDTOFromClassificationResponse_Empty(t *testing.T) {
	m := &model.ClassificationResponse{
		Levels:    []string{},
		Resources: []model.ClassificationResource{},
	}
	d := dtoFromClassificationResponse(m)
	assert.Empty(t, d.Levels)
	assert.Empty(t, d.Resources)
}

func TestDTOFromClassificationResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromClassificationResponse(nil))
}

// ──── dtoFromFieldMarkingResponse ──────────────────────────────────────────

func TestDTOFromFieldMarkingResponse_Full(t *testing.T) {
	m := &model.FieldMarkingResponse{
		Markings: []model.FieldMarking{
			{ObjectType: "order", Field: "order_id", Classification: "confidential", PII: true, LLMAllowed: false},
			{ObjectType: "order", Field: "status", Classification: "public", PII: false, LLMAllowed: true},
		},
	}
	d := dtoFromFieldMarkingResponse(m)
	assert.Len(t, d.Markings, 2)
	assert.Equal(t, "order", d.Markings[0].ObjectType)
	assert.Equal(t, "order_id", d.Markings[0].Field)
	assert.True(t, d.Markings[0].PII)
	assert.False(t, d.Markings[0].LLMAllowed)
	assert.False(t, d.Markings[1].PII)
	assert.True(t, d.Markings[1].LLMAllowed)
}

func TestDTOFromFieldMarkingResponse_Empty(t *testing.T) {
	m := &model.FieldMarkingResponse{Markings: []model.FieldMarking{}}
	d := dtoFromFieldMarkingResponse(m)
	assert.Empty(t, d.Markings)
}

func TestDTOFromFieldMarkingResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromFieldMarkingResponse(nil))
}

// ──── dtoFromLineageResponse ───────────────────────────────────────────────

func TestDTOFromLineageResponse_Full(t *testing.T) {
	m := &model.LineageResponse{
		Resource:   "orders.order_id",
		Upstream:   []string{"raw_orders.order_id", "customers.customer_id"},
		Downstream: []string{"report_monthly_revenue.order_id"},
	}
	d := dtoFromLineageResponse(m)
	assert.Equal(t, "orders.order_id", d.Resource)
	assert.Equal(t, []string{"raw_orders.order_id", "customers.customer_id"}, d.Upstream)
	assert.Equal(t, []string{"report_monthly_revenue.order_id"}, d.Downstream)
}

func TestDTOFromLineageResponse_EmptyChains(t *testing.T) {
	m := &model.LineageResponse{
		Resource:   "orders.status",
		Upstream:   []string{},
		Downstream: []string{},
	}
	d := dtoFromLineageResponse(m)
	assert.Equal(t, "orders.status", d.Resource)
	assert.Empty(t, d.Upstream)
	assert.Empty(t, d.Downstream)
}

func TestDTOFromLineageResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromLineageResponse(nil))
}

// ──── dtoFromCheckpointsResponse ───────────────────────────────────────────

func TestDTOFromCheckpointsResponse_Full(t *testing.T) {
	m := &model.CheckpointsResponse{
		Checkpoints: []model.CheckpointRule{
			{Action: "export_report", RequiresReason: true, RequiresHumanReview: true},
			{Action: "notify_owner", RequiresReason: false, RequiresHumanReview: false},
		},
	}
	d := dtoFromCheckpointsResponse(m)
	assert.Len(t, d.Checkpoints, 2)
	assert.Equal(t, "export_report", d.Checkpoints[0].Action)
	assert.True(t, d.Checkpoints[0].RequiresReason)
	assert.True(t, d.Checkpoints[0].RequiresHumanReview)
	assert.False(t, d.Checkpoints[1].RequiresReason)
}

func TestDTOFromCheckpointsResponse_Empty(t *testing.T) {
	m := &model.CheckpointsResponse{Checkpoints: []model.CheckpointRule{}}
	d := dtoFromCheckpointsResponse(m)
	assert.Empty(t, d.Checkpoints)
}

func TestDTOFromCheckpointsResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromCheckpointsResponse(nil))
}

// ──── dtoFromHealthChecksResponse ──────────────────────────────────────────

func TestDTOFromHealthChecksResponse_Full(t *testing.T) {
	m := &model.HealthChecksResponse{
		Status: "healthy",
		Checks: []model.HealthCheckItem{
			{Name: "database", Status: "ok"},
			{Name: "llm", Status: "ok"},
			{Name: "cache", Status: "degraded"},
		},
	}
	d := dtoFromHealthChecksResponse(m)
	assert.Equal(t, "healthy", d.Status)
	assert.Len(t, d.Checks, 3)
	assert.Equal(t, "database", d.Checks[0].Name)
	assert.Equal(t, "ok", d.Checks[0].Status)
	assert.Equal(t, "degraded", d.Checks[2].Status)
}

func TestDTOFromHealthChecksResponse_Empty(t *testing.T) {
	m := &model.HealthChecksResponse{
		Status: "unknown",
		Checks: []model.HealthCheckItem{},
	}
	d := dtoFromHealthChecksResponse(m)
	assert.Equal(t, "unknown", d.Status)
	assert.Empty(t, d.Checks)
}

func TestDTOFromHealthChecksResponse_Nil(t *testing.T) {
	assert.Nil(t, dtoFromHealthChecksResponse(nil))
}
