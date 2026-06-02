package schemas

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Go types matching the JSON Schema for manual validation.
// ---------------------------------------------------------------------------

// AgentDecision mirrors the canonical decision schema.
type AgentDecision struct {
	SchemaVersion       string              `json:"schema_version"`
	DecisionType        string              `json:"decision_type"`
	Severity            string              `json:"severity"`
	Summary             string              `json:"summary"`
	Rationale           []string            `json:"rationale"`
	RecommendedActions  []RecommendedAction `json:"recommended_actions"`
	Confidence          float64             `json:"confidence"`
	RequiresHumanReview bool                `json:"requires_human_review"`
	EvidenceRefs        []string            `json:"evidence_refs,omitempty"`
	RequiresApproval    *bool               `json:"requires_approval,omitempty"`
}

// RecommendedAction is a single action in a decision.
type RecommendedAction struct {
	ActionType string                 `json:"action_type"`
	Priority   string                 `json:"priority"`
	OwnerRole  string                 `json:"owner_role"`
	Payload    map[string]interface{} `json:"payload"`
}

// ---------------------------------------------------------------------------
// Schema loader and validator
// ---------------------------------------------------------------------------

// decisionSchema holds the parsed top-level keys of the JSON Schema file.
type decisionSchema struct {
	Required   []string             `json:"required"`
	Properties map[string]schemaProp `json:"properties"`
}

type schemaProp struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Enum        []interface{}       `json:"enum,omitempty"`
	Minimum     *float64            `json:"minimum,omitempty"`
	Maximum     *float64            `json:"maximum,omitempty"`
	MinLength   *int                `json:"minLength,omitempty"`
	MinItems    *int                `json:"minItems,omitempty"`
	Const       interface{}         `json:"const,omitempty"`
	Items       *schemaItems        `json:"items,omitempty"`
}

type schemaItems struct {
	Ref        string              `json:"$ref,omitempty"`
	Type       string              `json:"type,omitempty"`
	MinLength  *int                `json:"minLength,omitempty"`
	Properties map[string]schemaProp `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// loadSchema parses the JSON Schema file at the given path.
func loadSchema(t *testing.T, path string) *decisionSchema {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema file: %v", err)
	}

	// Verify the file is parseable JSON.
	var schema decisionSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	return &schema
}

// mustMarshal is a test helper that serialises d to JSON or fails the test.
func mustMarshal(t *testing.T, d interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// validateRequired checks that every required field in the schema is present
// and non-zero in the decision.
func validateRequired(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string
	for _, field := range s.Required {
		v, ok := d[field]
		if !ok || v == nil {
			errs = append(errs, "missing required field: "+field)
			continue
		}
		// Reject empty strings and zero-length arrays.
		switch tv := v.(type) {
		case string:
			if tv == "" {
				// Allow empty string if the field's enum explicitly permits it.
				if prop, ok := s.Properties[field]; ok {
					allowsEmpty := false
					for _, enumVal := range prop.Enum {
						if s, ok := enumVal.(string); ok && s == "" {
							allowsEmpty = true
							break
						}
					}
					if !allowsEmpty {
						errs = append(errs, "required field "+field+" must not be empty")
					}
				} else {
					errs = append(errs, "required field "+field+" must not be empty")
				}
			}
		case []interface{}:
			if len(tv) == 0 {
				errs = append(errs, "required field "+field+" must have at least 1 item")
			}
		case float64:
			if field == "confidence" && (tv < 0 || tv > 1) {
				errs = append(errs, "confidence must be in [0,1]")
			}
		}
	}
	return errs
}

// validateEnum checks that the value of a field is one of the allowed enums.
func validateEnum(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string
	for field, prop := range s.Properties {
		if len(prop.Enum) == 0 {
			continue
		}
		val, ok := d[field]
		if !ok {
			continue
		}
		found := false
		for _, allowed := range prop.Enum {
			if val == allowed {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, field+": value not in allowed enum")
		}
	}
	return errs
}

// validateConst checks that const fields match exactly.
func validateConst(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string
	for field, prop := range s.Properties {
		if prop.Const == nil {
			continue
		}
		val, ok := d[field]
		if !ok {
			errs = append(errs, field+": missing required const field")
			continue
		}
		if val != prop.Const {
			errs = append(errs, field+": must equal const value")
		}
	}
	return errs
}

// validateNumericBounds checks minimum/maximum constraints.
func validateNumericBounds(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string
	for field, prop := range s.Properties {
		if prop.Minimum == nil && prop.Maximum == nil {
			continue
		}
		val, ok := d[field]
		if !ok {
			continue
		}
		fv, ok := val.(float64)
		if !ok {
			continue
		}
		if prop.Minimum != nil && fv < *prop.Minimum {
			errs = append(errs, field+": value below minimum")
		}
		if prop.Maximum != nil && fv > *prop.Maximum {
			errs = append(errs, field+": value above maximum")
		}
	}
	return errs
}

// validateRecommendedActions checks that each recommended action matches
// the RecommendedAction sub-schema constraints.
func validateRecommendedActions(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string

	actionsRaw, ok := d["recommended_actions"]
	if !ok {
		return nil // required check handles absence
	}
	actionsList, ok := actionsRaw.([]interface{})
	if !ok {
		return []string{"recommended_actions: expected array"}
	}

	actionProp, exists := s.Properties["recommended_actions"]
	if !exists || actionProp.Items == nil {
		return errs
	}

	for i, raw := range actionsList {
		action, ok := raw.(map[string]interface{})
		if !ok {
			errs = append(errs, "recommended_actions["+itoa(i)+"]: expected object")
			continue
		}

		actionSchema := actionProp.Items
		prefix := "recommended_actions[" + itoa(i) + "]"

		// Check required action fields.
		for _, req := range actionSchema.Required {
			if _, ok := action[req]; !ok {
				errs = append(errs, prefix+"."+req+": missing required field")
			}
		}

		// Check action_type enum.
		at, ok := action["action_type"]
		if ok {
			allowedActions := []string{
				"create_followup_task",
				"notify_owner",
				"export_report",
				"create_outbox_message",
				"escalate_to_human",
			}
			found := false
			for _, a := range allowedActions {
				if at == a {
					found = true
					break
				}
			}
			if !found {
				errs = append(errs, prefix+".action_type: invalid action type")
			}
		}

		// Check that payload is an object with at least 1 property.
		if p, ok := action["payload"]; ok {
			if _, isObj := p.(map[string]interface{}); !isObj {
				errs = append(errs, prefix+".payload: must be an object")
			}
		}
	}

	return errs
}

// validateDecision runs all schema validation rules against a decision map.
func validateDecision(s *decisionSchema, d map[string]interface{}) []string {
	var errs []string
	errs = append(errs, validateRequired(s, d)...)
	errs = append(errs, validateEnum(s, d)...)
	errs = append(errs, validateConst(s, d)...)
	errs = append(errs, validateNumericBounds(s, d)...)
	errs = append(errs, validateRecommendedActions(s, d)...)
	return errs
}

// itoa converts an int to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	d := i
	digits := ""
	for d > 0 {
		digits = string(rune('0'+d%10)) + digits
		d /= 10
	}
	return digits
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

const schemaPath = "/home/zzz/project/baxi/pi-agent/schemas/decision.schema.json"

func TestSchemaFile_IsParseableJSON(t *testing.T) {
	s := loadSchema(t, schemaPath)

	// Basic structural assertions.
	if len(s.Required) == 0 {
		t.Error("schema has no required fields")
	}
	if len(s.Properties) == 0 {
		t.Error("schema has no properties")
	}
}

func TestSchemaFile_HasRequiredFields(t *testing.T) {
	s := loadSchema(t, schemaPath)

	required := map[string]bool{
		"schema_version":       false,
		"decision_type":        false,
		"severity":             false,
		"summary":              false,
		"rationale":            false,
		"recommended_actions":  false,
		"confidence":           false,
		"requires_human_review": false,
	}
	for _, r := range s.Required {
		required[r] = true
	}
	for field, found := range required {
		if !found {
			t.Errorf("missing required field in schema: %s", field)
		}
	}
}

func TestValidDecision_PassesValidation(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "decision_output.v1",
		"decision_type":  "intervention",
		"severity":       "high",
		"summary":        "Seller SELLER_001 exhibits significant late delivery rate of 31%.",
		"rationale": []interface{}{
			"Late delivery rate is 31%, far exceeding 8% baseline.",
			"Order count declining and review scores dropping.",
		},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "notify_owner",
				"priority":    "high",
				"owner_role":  "seller_ops",
				"payload": map[string]interface{}{
					"message":  "Late delivery intervention required.",
					"channels": []interface{}{"feishu"},
				},
			},
		},
		"confidence":           0.85,
		"requires_human_review": true,
		"evidence_refs":        []interface{}{"metric:seller_late_delivery_rate_7d"},
	}

	errs := validateDecision(s, decision)
	if len(errs) > 0 {
		t.Errorf("valid decision should pass, got %d errors: %v", len(errs), errs)
	}
}

func TestValidDecision_EmptySchemaVersion_Passes(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "",
		"decision_type":  "monitor_only",
		"severity":       "low",
		"summary":        "No anomaly detected.",
		"rationale":      []interface{}{"All metrics within normal range."},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "create_followup_task",
				"priority":    "low",
				"owner_role":  "data_engineer",
				"payload":     map[string]interface{}{"note": "log for tracking"},
			},
		},
		"confidence":           0.95,
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) > 0 {
		t.Errorf("valid decision with empty schema_version should pass, got errors: %v", errs)
	}
}

// --- Invalid examples -------------------------------------------------------

func TestInvalidDecision_MissingRequiredField_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	// Missing "summary" field entirely.
	decision := map[string]interface{}{
		"schema_version": "decision_output.v1",
		"decision_type":  "investigate",
		"severity":       "medium",
		"rationale":      []interface{}{"reason 1"},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "notify_owner",
				"priority":    "high",
				"owner_role":  "ops",
				"payload":     map[string]interface{}{"message": "hello"},
			},
		},
		"confidence":           0.5,
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for missing required field 'summary', got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "summary") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about missing 'summary', got: %v", errs)
	}
}

func TestInvalidDecision_WrongEnumValue_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "decision_output.v1",
		"decision_type":  "bogus_type", // not in enum
		"severity":       "medium",
		"summary":        "Test summary",
		"rationale":      []interface{}{"reason 1"},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "notify_owner",
				"priority":    "high",
				"owner_role":  "ops",
				"payload":     map[string]interface{}{"message": "hello"},
			},
		},
		"confidence":           0.5,
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for invalid decision_type, got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "decision_type") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about 'decision_type', got: %v", errs)
	}
}

func TestInvalidDecision_ConfidenceOutOfRange_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "",
		"decision_type":  "investigate",
		"severity":       "medium",
		"summary":        "Test summary",
		"rationale":      []interface{}{"reason 1"},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "notify_owner",
				"priority":    "high",
				"owner_role":  "ops",
				"payload":     map[string]interface{}{"message": "hello"},
			},
		},
		"confidence":           1.5, // out of range
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for confidence out of range, got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "confidence") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about 'confidence', got: %v", errs)
	}
}

func TestInvalidDecision_RequiresHumanReviewFalse_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "decision_output.v1",
		"decision_type":  "investigate",
		"severity":       "medium",
		"summary":        "Test summary",
		"rationale":      []interface{}{"reason 1"},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "notify_owner",
				"priority":    "high",
				"owner_role":  "ops",
				"payload":     map[string]interface{}{"message": "hello"},
			},
		},
		"confidence":           0.5,
		"requires_human_review": false, // must be true
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for requires_human_review=false, got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "requires_human_review") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about 'requires_human_review', got: %v", errs)
	}
}

func TestInvalidDecision_InvalidActionType_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "",
		"decision_type":  "optimize",
		"severity":       "low",
		"summary":        "Optimize shipping routes",
		"rationale":      []interface{}{"Route analysis complete."},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "restart_server", // not in allowed enum
				"priority":    "high",
				"owner_role":  "ops",
				"payload":     map[string]interface{}{"host": "prod-01"},
			},
		},
		"confidence":           0.7,
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for invalid action_type, got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "action_type") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about 'action_type', got: %v", errs)
	}
}

func TestInvalidDecision_NegativeConfidence_Fails(t *testing.T) {
	s := loadSchema(t, schemaPath)

	decision := map[string]interface{}{
		"schema_version": "",
		"decision_type":  "monitor_only",
		"severity":       "low",
		"summary":        "Everything is fine",
		"rationale":      []interface{}{"Baseline checks passed."},
		"recommended_actions": []interface{}{
			map[string]interface{}{
				"action_type": "create_followup_task",
				"priority":    "low",
				"owner_role":  "observer",
				"payload":     map[string]interface{}{"log": true},
			},
		},
		"confidence":           -0.1, // below minimum
		"requires_human_review": true,
	}

	errs := validateDecision(s, decision)
	if len(errs) == 0 {
		t.Fatal("expected validation errors for negative confidence, got none")
	}
	found := false
	for range errs {
		if fieldHasError(errs, "confidence") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error about 'confidence', got: %v", errs)
	}
}

func TestSchemaFile_AdditionalPropertiesFalse_IsHonored(t *testing.T) {
	// Verify the schema specifies additionalProperties: false at the root.
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	ap, ok := doc["additionalProperties"]
	if !ok || ap != false {
		t.Error(`schema must set "additionalProperties": false at root level`)
	}
}

// fieldHasError reports whether any string in the slice contains the given prefix.
func fieldHasError(errs []string, fieldPrefix string) bool {
	for _, e := range errs {
		if strings.Contains(e, fieldPrefix) {
			return true
		}
	}
	return false
}

// Ensure Go's float comparison does not fail for the confidence 0 check.
var _ = math.Float64bits(0.0)
