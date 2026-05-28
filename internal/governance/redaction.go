package governance

import "sort"

// RedactionPolicy defines the context for redaction decisions.
type RedactionPolicy struct {
	Role string
}

// RedactionResult holds the filtered properties and a log of redacted fields.
type RedactionResult struct {
	Properties     map[string]interface{} `json:"properties"`
	RedactedFields []RedactedFieldEntry   `json:"redacted_fields"`
}

// RedactedFieldEntry records why a field was redacted.
type RedactedFieldEntry struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
	Rule   string `json:"rule"`
}

// RedactObjectContext strips fields from properties based on classification
// and marking rules for the given policy role. Marking checks take priority
// over classification. Redacted fields are sorted by name for determinism.
func RedactObjectContext(
	properties map[string]interface{},
	classifications map[string]string,
	markings map[string]string,
	policy RedactionPolicy,
) RedactionResult {
	filtered := make(map[string]interface{}, len(properties))
	var redacted []RedactedFieldEntry

	for field, value := range properties {
		if entry, ok := checkMarking(field, markings, policy.Role); ok {
			redacted = append(redacted, entry)
			continue
		}
		if entry, ok := checkClassification(field, classifications, policy.Role); ok {
			redacted = append(redacted, entry)
			continue
		}
		filtered[field] = value
	}

	sort.Slice(redacted, func(i, j int) bool {
		return redacted[i].Field < redacted[j].Field
	})

	return RedactionResult{
		Properties:     filtered,
		RedactedFields: redacted,
	}
}

func checkMarking(field string, markings map[string]string, role string) (RedactedFieldEntry, bool) {
	marking, ok := markings[field]
	if !ok {
		return RedactedFieldEntry{}, false
	}

	var redact bool
	switch marking {
	case "PII", "FINANCIAL_INTERNAL", "RAW_DATA":
		redact = role != "admin"
	case "OPERATIONAL_INTERNAL":
		redact = role == "viewer"
	}

	if !redact {
		return RedactedFieldEntry{}, false
	}

	return RedactedFieldEntry{
		Field:  field,
		Reason: "marking: " + marking,
		Rule:   marking,
	}, true
}

func checkClassification(field string, classifications map[string]string, role string) (RedactedFieldEntry, bool) {
	classification, ok := classifications[field]
	if !ok {
		return RedactedFieldEntry{}, false
	}

	var redact bool
	switch classification {
	case "pii":
		redact = true
	case "sensitive", "derived_sensitive":
		redact = role == "viewer" || role == "agent_readonly"
	case "internal":
		redact = role == "viewer"
	case "public_internal":
		redact = false
	}

	if !redact {
		return RedactedFieldEntry{}, false
	}

	return RedactedFieldEntry{
		Field:  field,
		Reason: "classification: " + classification,
		Rule:   classification,
	}, true
}
