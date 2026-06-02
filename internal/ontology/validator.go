package ontology

import (
	"fmt"
	"sort"
	"strings"
)

// ValidationIssue describes a single problem found during schema validation.
type ValidationIssue struct {
	ObjectType string `json:"object_type"`
	Severity   string `json:"severity"` // "error" or "warning"
	Message    string `json:"message"`
}

func (v ValidationIssue) String() string {
	return fmt.Sprintf("[%s] %s: %s", v.Severity, v.ObjectType, v.Message)
}

// ValidationResult holds the outcome of schema validation.
type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Issues  []ValidationIssue `json:"issues"`
	Summary string            `json:"summary"`
}

// Validate checks the ObjectRegistry for schema completeness and consistency.
//
// Checks performed:
//  1. All 8 expected object types are present.
//  2. Each object type has a non-empty grain, display name, and at least one
//     source table.
//  3. Each object type has at least one property.
//  4. Each object type has exactly one primary key property (is_pk = true).
//  5. All links target existing, registered object types.
//  6. All alert_fields reference properties that actually exist.
func (r *ObjectRegistry) Validate() *ValidationResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := &ValidationResult{
		Valid:  true,
		Issues: []ValidationIssue{},
	}

	// 1. All 8 expected object types are present.
	expected := AllObjectTypes()
	for _, name := range expected {
		if _, ok := r.objects[name]; !ok {
			result.Issues = append(result.Issues, ValidationIssue{
				ObjectType: name,
				Severity:   "error",
				Message:    fmt.Sprintf("missing required object type %q", name),
			})
		}
	}

	// If expected types are missing, still check what we have.
	for _, ot := range r.objects {
		result.Issues = append(result.Issues, r.validateObjectType(ot)...)
	}

	// Cross-object link validation (must happen after all objects are scanned).
	for name, ot := range r.objects {
		for _, link := range ot.Links {
			if _, exists := r.objects[link.TargetType]; !exists {
				result.Issues = append(result.Issues, ValidationIssue{
					ObjectType: name,
					Severity:   "error",
					Message:    fmt.Sprintf("link %q targets unknown object type %q", link.Name, link.TargetType),
				})
			}
		}
	}

	// V2 validation: validate registered v2 objects.
	v2Issues := ValidateV2(r.objectsV2)
	for _, iss := range v2Issues {
		result.Issues = append(result.Issues, iss)
	}

	// Build summary.
	result.Valid = true
	for _, iss := range result.Issues {
		if iss.Severity == "error" {
			result.Valid = false
			break
		}
	}

	if result.Valid {
		result.Summary = fmt.Sprintf("✓ All %d object types valid", len(r.objects))
	} else {
		errorCount := 0
		for _, iss := range result.Issues {
			if iss.Severity == "error" {
				errorCount++
			}
		}
		result.Summary = fmt.Sprintf("✗ %d validation error(s) across %d object type(s)", errorCount, len(r.objects))
	}

	// Sort issues for stable output.
	sort.Slice(result.Issues, func(i, j int) bool {
		if result.Issues[i].ObjectType != result.Issues[j].ObjectType {
			return result.Issues[i].ObjectType < result.Issues[j].ObjectType
		}
		return result.Issues[i].Message < result.Issues[j].Message
	})

	return result
}

func (r *ObjectRegistry) validateObjectType(ot *ObjectType) []ValidationIssue {
	var issues []ValidationIssue
	name := ot.Name

	// 2a. Non-empty grain.
	if strings.TrimSpace(ot.Grain) == "" {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "error",
			Message: "grain is empty",
		})
	}

	// 2b. Non-empty display name.
	if strings.TrimSpace(ot.DisplayName) == "" {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "warning",
			Message: "display_name is empty",
		})
	}

	// 2c. At least one source table.
	if len(ot.SourceTables) == 0 {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "error",
			Message: "no source_tables defined",
		})
	}

	// 3. At least one property.
	if len(ot.Properties) == 0 {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "error",
			Message: "no properties defined",
		})
	}

	// 4. Exactly one primary key property.
	pkCount := 0
	for _, prop := range ot.Properties {
		if prop.IsPK {
			pkCount++
		}
	}
	if pkCount == 0 {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "error",
			Message: "no primary key property (is_pk=true) found",
		})
	} else if pkCount > 1 {
		issues = append(issues, ValidationIssue{
			ObjectType: name, Severity: "warning",
			Message: fmt.Sprintf("multiple primary key properties (%d found)", pkCount),
		})
	}

	// 6. Alert fields must reference existing properties.
	for _, af := range ot.AlertFields {
		if _, exists := ot.Properties[af]; !exists {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: fmt.Sprintf("alert_field %q does not match any property", af),
			})
		}
	}

	return issues
}
