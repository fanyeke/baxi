package governance

import (
	"context"
	"fmt"

	governanceRepo "baxi/internal/repository/governance"
)

// ClassificationService provides data classification lookup and level mapping.
// Classification levels: pii→L3, sensitive→L3, internal→L2, public_internal→L1, derived_sensitive→L2
type ClassificationService struct {
	repo *governanceRepo.Repository
}

// NewClassificationService creates a new ClassificationService.
func NewClassificationService(repo *governanceRepo.Repository) *ClassificationService {
	return &ClassificationService{repo: repo}
}

// ResolveLevel maps a classification level name to its canonical label.
// Unknown levels default to "L2" (internal).
func ResolveLevel(level string) string {
	switch level {
	case "pii", "sensitive":
		return "L3"
	case "internal", "derived_sensitive":
		return "L2"
	case "public_internal":
		return "L1"
	default:
		return "L2"
	}
}

// GetClassification looks up the classification level for a field path.
// Returns the default classification "internal" (L2) for unknown fields.
func (s *ClassificationService) GetClassification(ctx context.Context, fieldPath string) (string, error) {
	row, err := s.repo.GetByFieldPath(ctx, fieldPath)
	if err != nil {
		// Return default classification for unknown fields
		return ResolveLevel("internal"), nil
	}
	return ResolveLevel(row.ClassificationLevel), nil
}

// GetFieldMarking returns classification details for a specific object_type and property.
// Loads all classifications and filters locally.
func (s *ClassificationService) GetFieldMarking(ctx context.Context, objectType, property string) (string, bool, bool, error) {
	classifications, err := s.repo.GetDataClassifications(ctx)
	if err != nil {
		return ResolveLevel("internal"), false, false, fmt.Errorf("load classifications: %w", err)
	}

	prefix := objectType + "." + property
	for _, c := range classifications {
		if c.FieldPath == prefix {
			level := ResolveLevel(c.ClassificationLevel)
			isPII := c.ClassificationLevel == "pii"
			llmAllowed := level != "L3" // L3 (pii/sensitive) not allowed for LLM
			return level, isPII, llmAllowed, nil
		}
	}

	return ResolveLevel("internal"), false, true, nil
}

// GetAll returns all data classifications from the database.
func (s *ClassificationService) GetAll(ctx context.Context) ([]governanceRepo.DataClassificationRow, error) {
	return s.repo.GetDataClassifications(ctx)
}
