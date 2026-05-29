package decision

import (
	"context"
	"fmt"

	"baxi/internal/ontology"
)

// ObjectContextBuilder is the interface for building decision contexts.
// Both ContextBuilder and ContextBuilderV2 implement this interface.
type ObjectContextBuilder interface {
	BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error)
}

// objectQueryService defines the query operations needed by ContextBuilderV3.
type objectQueryService interface {
	BuildObjectContext(ctx context.Context, objectType, objectID string) (*ontology.ObjectContext, error)
}

// objectLinkProvider defines the link lookup operations needed by ContextBuilderV3.
type objectLinkProvider interface {
	GetLinks(objectType string) ([]ontology.ObjectLink, error)
}

// ContextBuilderV3 implements ObjectContextBuilder with OAG (Object-Action-Governance)
// link traversal. It wraps a delegate builder (typically ContextBuilderV2) for the base
// context and enriches it by traversing ontology links to fetch related objects.
//
// Link traversal is capped at a maximum depth of 2 to prevent infinite loops.
// Single link failures are logged and skipped gracefully.
type ContextBuilderV3 struct {
	delegate ObjectContextBuilder
	linkProv objectLinkProvider
	querySvc objectQueryService
}

// NewContextBuilderV3 creates a ContextBuilderV3 that wraps the given delegate
// builder and enriches decision contexts via ontology link traversal.
func NewContextBuilderV3(
	delegate ObjectContextBuilder,
	linkProv objectLinkProvider,
	querySvc objectQueryService,
) *ContextBuilderV3 {
	return &ContextBuilderV3{
		delegate: delegate,
		linkProv: linkProv,
		querySvc: querySvc,
	}
}

// BuildDecisionContext constructs a DecisionContext by first building the base
// context via the delegate, then enriching it with linked objects discovered
// through ontology link traversal at depths 1 and 2.
func (b *ContextBuilderV3) BuildDecisionContext(ctx context.Context, caseID string) (*DecisionContext, error) {
	baseCtx, err := b.delegate.BuildDecisionContext(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("v3: build base context: %w", err)
	}

	objectType := baseCtx.ObjectContext.ObjectType
	objectID := baseCtx.ObjectContext.ObjectID

	if objectType == "" || objectID == "" {
		return baseCtx, nil
	}

	enriched, err := b.traverseAtDepth(ctx, objectType, objectID, "", 1, 2)
	if err != nil {
		return baseCtx, nil
	}

	if len(enriched) > 0 {
		baseCtx.EnrichedObjects = enriched
	}

	return baseCtx, nil
}

// traverseAtDepth recursively traverses ontology links starting from the given
// object type and ID. currentDepth starts at 1 and maxDepth caps the recursion.
// linkNamePrefix carries the parent link path for nested traversal labeling.
func (b *ContextBuilderV3) traverseAtDepth(
	ctx context.Context,
	objectType, objectID, linkNamePrefix string,
	currentDepth, maxDepth int,
) ([]EnrichedObjectData, error) {
	if currentDepth > maxDepth {
		return nil, nil
	}

	links, err := b.linkProv.GetLinks(objectType)
	if err != nil {
		return nil, nil
	}

	if len(links) == 0 {
		return nil, nil
	}

	sourceObj, err := b.querySvc.BuildObjectContext(ctx, objectType, objectID)
	if err != nil {
		return nil, nil
	}

	var results []EnrichedObjectData

	for _, link := range links {
		viaVal, ok := sourceObj.Properties[link.Via]
		if !ok || viaVal == nil {
			continue
		}

		viaStr := fmt.Sprintf("%v", viaVal)
		if viaStr == "" {
			continue
		}

		linkedObj, err := b.querySvc.BuildObjectContext(ctx, link.TargetType, viaStr)
		if err != nil {
			continue
		}

		linkName := link.Name
		if linkNamePrefix != "" {
			linkName = linkNamePrefix + "." + link.Name
		}

		results = append(results, EnrichedObjectData{
			LinkName:   linkName,
			Depth:      currentDepth,
			ObjectType: linkedObj.ObjectType,
			ObjectID:   linkedObj.ObjectID,
			Properties: linkedObj.Properties,
		})

		if currentDepth < maxDepth {
			childPrefix := link.Name
			if linkNamePrefix != "" {
				childPrefix = linkNamePrefix + "." + link.Name
			}
			childResults, err := b.traverseAtDepth(
				ctx, linkedObj.ObjectType, linkedObj.ObjectID,
				childPrefix, currentDepth+1, maxDepth,
			)
			if err == nil {
				results = append(results, childResults...)
			}
		}
	}

	return results, nil
}

// compile-time interface check
var _ ObjectContextBuilder = (*ContextBuilderV3)(nil)
