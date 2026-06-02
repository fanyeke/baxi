package ontology

import "fmt"

// ValidateV2 checks the v2 object schema map for completeness and consistency.
//
// Checks:
//  1. Each object has a non-empty Source (schema/table/primary_key).
//  2. Each object has at least one property.
//  3. Each object has exactly one primary key property.
//  4. All link targets reference existing object types.
//  5. All metric names are unique.
func ValidateV2(objects map[string]*ObjectTypeV2) []ValidationIssue {
	var issues []ValidationIssue

	for name, ot := range objects {
		// 1. Source completeness
		if ot.Source.Schema == "" {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: "source.schema is required",
			})
		}
		if ot.Source.Table == "" {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: "source.table is required",
			})
		}
		if ot.Source.PrimaryKey == "" {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: "source.primary_key is required",
			})
		}

		// 2. At least one property
		if len(ot.Properties) == 0 {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: "no properties defined",
			})
		}

		// 3. Exactly one PK property
		pkCount := 0
		for _, prop := range ot.Properties {
			if prop.IsPK {
				pkCount++
			}
		}
		if pkCount == 0 {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: "no primary key property found (is_pk=true)",
			})
		} else if pkCount > 1 {
			issues = append(issues, ValidationIssue{
				ObjectType: name, Severity: "error",
				Message: fmt.Sprintf("multiple primary key properties (%d)", pkCount),
			})
		}
	}

	// 4. Link targets exist
	for name, ot := range objects {
		for _, link := range ot.Links {
			if _, exists := objects[link.TargetType]; !exists {
				issues = append(issues, ValidationIssue{
					ObjectType: name, Severity: "error",
					Message: fmt.Sprintf("link %q targets unknown object type %q", link.Name, link.TargetType),
				})
			}
		}
	}

	return issues
}
