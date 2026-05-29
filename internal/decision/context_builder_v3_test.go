package decision

import (
	"context"
	"testing"

	"baxi/internal/ontology"
	"github.com/stretchr/testify/assert"
)

// mockObjectContextBuilder implements ObjectContextBuilder for testing.
type mockObjectContextBuilder struct {
	buildFn func(ctx context.Context, caseID string) (*DecisionContext, error)
}

func (m *mockObjectContextBuilder) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	return m.buildFn(ctx, caseID)
}

// mockQuerySvc implements objectQueryService for testing.
type mockQuerySvc struct {
	buildFn func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error)
}

func (m *mockQuerySvc) BuildObjectContext(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
	return m.buildFn(ctx, objectType, objectID)
}

// mockLinkProv implements objectLinkProvider for testing.
type mockLinkProv struct {
	getLinksFn func(objectType string) ([]ontology.ObjectLink, error)
}

func (m *mockLinkProv) GetLinks(objectType string) ([]ontology.ObjectLink, error) {
	return m.getLinksFn(objectType)
}

func TestContextBuilderV3_DelegatesToBaseBuilder(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "seller",
					ObjectID:   "seller-1",
					Properties: map[string]interface{}{
						"name": "Test Seller",
					},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{}, nil
		},
	}

	querySvc := &mockQuerySvc{}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Equal(t, "dc-1", decisionCtx.DecisionCaseID)
	assert.Equal(t, "seller", decisionCtx.ObjectContext.ObjectType)
}

func TestContextBuilderV3_NoLinks_ReturnsBaseContext(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "seller",
					ObjectID:   "seller-1",
					Properties: map[string]interface{}{"name": "Test"},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{}, nil
		},
	}

	querySvc := &mockQuerySvc{}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Empty(t, decisionCtx.EnrichedObjects)
}

func TestContextBuilderV3_TraversesLinksAtDepth1(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{
						"order_id":   "order-1",
						"seller_id":  "seller-42",
						"total_gmv":  1500.0,
					},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{
				{Name: "seller", TargetType: "seller", Via: "seller_id"},
			}, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			if objectType == "order" && objectID == "order-1" {
				return &ontology.ObjectContext{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{
						"seller_id": "seller-42",
					},
				}, nil
			}
			if objectType == "seller" && objectID == "seller-42" {
				return &ontology.ObjectContext{
					ObjectType: "seller",
					ObjectID:   "seller-42",
					Properties: map[string]interface{}{
						"name":                 "Test Seller",
						"seller_state":         "SP",
						"seller_city":          "Sao Paulo",
						"business_segment":     "electronics",
					},
				}, nil
			}
			return nil, assert.AnError
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Len(t, decisionCtx.EnrichedObjects, 1)

	obj := decisionCtx.EnrichedObjects[0]
	assert.Equal(t, "seller", obj.LinkName)
	assert.Equal(t, 1, obj.Depth)
	assert.Equal(t, "seller", obj.ObjectType)
	assert.Equal(t, "seller-42", obj.ObjectID)
	assert.Equal(t, "Test Seller", obj.Properties["name"])
}

func TestContextBuilderV3_TraversesLinksAtDepth2(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{
						"order_id":   "order-1",
						"seller_id":  "seller-42",
					},
				},
			}, nil
		},
	}

	callCount := 0
	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			callCount++
			if objectType == "order" {
				return []ontology.ObjectLink{
					{Name: "seller", TargetType: "seller", Via: "seller_id"},
				}, nil
			}
			if objectType == "seller" {
				return []ontology.ObjectLink{
					{Name: "category", TargetType: "category", Via: "seller_category_id"},
				}, nil
			}
			return nil, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			if objectType == "order" && objectID == "order-1" {
				return &ontology.ObjectContext{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{"seller_id": "seller-42"},
				}, nil
			}
			if objectType == "seller" && objectID == "seller-42" {
				return &ontology.ObjectContext{
					ObjectType: "seller",
					ObjectID:   "seller-42",
					Properties: map[string]interface{}{"seller_category_id": "cat-5", "name": "Seller X"},
				}, nil
			}
			if objectType == "category" && objectID == "cat-5" {
				return &ontology.ObjectContext{
					ObjectType: "category",
					ObjectID:   "cat-5",
					Properties: map[string]interface{}{"category_name": "Electronics"},
				}, nil
			}
			return nil, assert.AnError
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Len(t, decisionCtx.EnrichedObjects, 2)

	assert.Equal(t, "seller", decisionCtx.EnrichedObjects[0].LinkName)
	assert.Equal(t, 1, decisionCtx.EnrichedObjects[0].Depth)
	assert.Equal(t, "seller", decisionCtx.EnrichedObjects[0].ObjectType)

	assert.Equal(t, "seller.category", decisionCtx.EnrichedObjects[1].LinkName)
	assert.Equal(t, 2, decisionCtx.EnrichedObjects[1].Depth)
	assert.Equal(t, "category", decisionCtx.EnrichedObjects[1].ObjectType)
	assert.Equal(t, "Electronics", decisionCtx.EnrichedObjects[1].Properties["category_name"])
}

func TestContextBuilderV3_SkipsMissingViaField(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{
						"order_id": "order-1",
					},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{
				{Name: "seller", TargetType: "seller", Via: "seller_id"},
			}, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			if objectType == "order" && objectID == "order-1" {
				return &ontology.ObjectContext{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{"order_id": "order-1"},
				}, nil
			}
			return nil, assert.AnError
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Empty(t, decisionCtx.EnrichedObjects)
}

func TestContextBuilderV3_SkipsLinkOnFetchError(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{"seller_id": "seller-missing"},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{
				{Name: "seller", TargetType: "seller", Via: "seller_id"},
			}, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			if objectType == "order" && objectID == "order-1" {
				return &ontology.ObjectContext{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{"seller_id": "seller-missing"},
				}, nil
			}
			return nil, assert.AnError // seller-missing returns error
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Empty(t, decisionCtx.EnrichedObjects)
}

func TestContextBuilderV3_EmptyObjectType_ReturnsBaseContext(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext:  ObjectContextData{},
			}, nil
		},
	}

	builder := NewContextBuilderV3(delegate, nil, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Empty(t, decisionCtx.EnrichedObjects)
}

func TestContextBuilderV3_DelegateError_Propagates(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return nil, assert.AnError
		},
	}

	builder := NewContextBuilderV3(delegate, nil, nil)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.Error(t, err)
	assert.Nil(t, decisionCtx)
	assert.Contains(t, err.Error(), "v3: build base context")
}

func TestContextBuilderV3_MultipleLinksOnSameObject(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{
						"seller_id":  "seller-42",
						"product_id": "prod-7",
					},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			return []ontology.ObjectLink{
				{Name: "seller", TargetType: "seller", Via: "seller_id"},
				{Name: "product", TargetType: "product", Via: "product_id"},
			}, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			if objectType == "order" {
				return &ontology.ObjectContext{
					ObjectType: "order",
					ObjectID:   objectID,
					Properties: map[string]interface{}{
						"seller_id":  "seller-42",
						"product_id": "prod-7",
					},
				}, nil
			}
			if objectType == "seller" {
				return &ontology.ObjectContext{
					ObjectType: "seller",
					ObjectID:   objectID,
					Properties: map[string]interface{}{"name": "Seller X"},
				}, nil
			}
			if objectType == "product" {
				return &ontology.ObjectContext{
					ObjectType: "product",
					ObjectID:   objectID,
					Properties: map[string]interface{}{"product_name": "Widget"},
				}, nil
			}
			return nil, assert.AnError
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	assert.Len(t, decisionCtx.EnrichedObjects, 2)

	linkNames := make([]string, len(decisionCtx.EnrichedObjects))
	for i, o := range decisionCtx.EnrichedObjects {
		linkNames[i] = o.LinkName
	}
	assert.Contains(t, linkNames, "seller")
	assert.Contains(t, linkNames, "product")
}

func TestContextBuilderV3_MaxDepthLimit(t *testing.T) {
	delegate := &mockObjectContextBuilder{
		buildFn: func(ctx context.Context, caseID string) (*DecisionContext, error) {
			return &DecisionContext{
				DecisionCaseID: caseID,
				ObjectContext: ObjectContextData{
					ObjectType: "order",
					ObjectID:   "order-1",
					Properties: map[string]interface{}{"seller_id": "seller-42"},
				},
			}, nil
		},
	}

	linkProv := &mockLinkProv{
		getLinksFn: func(objectType string) ([]ontology.ObjectLink, error) {
			// Always return one link regardless of object type
			return []ontology.ObjectLink{
				{Name: "linked", TargetType: objectType, Via: objectType + "_id"},
			}, nil
		},
	}

	querySvc := &mockQuerySvc{
		buildFn: func(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error) {
			return &ontology.ObjectContext{
				ObjectType: objectType,
				ObjectID:   objectID,
				Properties: map[string]interface{}{
					objectType + "_id": "child-" + objectID,
				},
			}, nil
		},
	}

	builder := NewContextBuilderV3(delegate, linkProv, querySvc)
	decisionCtx, err := builder.BuildDecisionContext(context.Background(), "dc-1")

	assert.NoError(t, err)
	assert.NotNil(t, decisionCtx)
	// Depth should be capped at 2, so we expect exactly 2 enriched objects
	assert.Len(t, decisionCtx.EnrichedObjects, 2)
	assert.Equal(t, 1, decisionCtx.EnrichedObjects[0].Depth)
	assert.Equal(t, 2, decisionCtx.EnrichedObjects[1].Depth)
}
