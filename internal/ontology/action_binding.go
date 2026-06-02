package ontology

// ──── Action Binding types ────────────────────────────────────────────────────
// ActionProposal represents a proposed action on an object with full audit trail.
// ActionBindingValidator checks pre-execution constraints against the ontology
// schema and the global action registry.

// ActionProposal represents a proposed action on an object.
// It tracks the full lifecycle from creation through approval to execution.
type ActionProposal struct {
	ID               string
	ObjectType       string
	ObjectID         string
	ActionType       string
	Payload          map[string]any
	RiskLevel        string // low, medium, high
	RequiresApproval bool
	Status           string // pending, approved, rejected, executed, dry_run
	CreatedBy        string
	ValidationResult *ActionValidationResult
}

// ActionValidationResult holds the result of pre-execution checks.
type ActionValidationResult struct {
	Valid  bool
	Checks []ActionValidationCheck
}

// ActionValidationCheck records a single validation step.
type ActionValidationCheck struct {
	Check  string
	Passed bool
	Reason string
}

// NewActionProposal creates a new action proposal with initial status.
func NewActionProposal(id, objectType, objectID, actionType string, payload map[string]any) *ActionProposal {
	return &ActionProposal{
		ID:         id,
		ObjectType: objectType,
		ObjectID:   objectID,
		ActionType: actionType,
		Payload:    payload,
		Status:     "pending",
	}
}

// ActionBindingValidator checks if an action is allowed per the object schema.
type ActionBindingValidator struct {
	objectTypes   map[string]*ObjectTypeV2
	allowedBy     map[string][]string // action_type → allowed roles
	actionEnabled map[string]bool     // action_type → enabled flag
}

// NewActionBindingValidator creates a validator from v2 object types and
// action registry data (allowed roles and enabled flags).
func NewActionBindingValidator(objectTypes map[string]*ObjectTypeV2) *ActionBindingValidator {
	return &ActionBindingValidator{
		objectTypes:   objectTypes,
		allowedBy:     make(map[string][]string),
		actionEnabled: make(map[string]bool),
	}
}

// SetActionRegistry sets the action registry data for validation.
func (v *ActionBindingValidator) SetActionRegistry(allowedBy map[string][]string, actionEnabled map[string]bool) {
	v.allowedBy = allowedBy
	v.actionEnabled = actionEnabled
}

// Validate performs the full validation chain for an action on an object.
// Returns a detailed validation result with per-check status.
func (v *ActionBindingValidator) Validate(objectType, actionType string, role string) *ActionValidationResult {
	result := &ActionValidationResult{
		Valid:  true,
		Checks: make([]ActionValidationCheck, 0),
	}

	// 1. object type exists
	ot, exists := v.objectTypes[objectType]
	result.Checks = append(result.Checks, ActionValidationCheck{
		Check:  "object_exists",
		Passed: exists,
		Reason: cond(exists, "", "object type "+objectType+" not found"),
	})
	if !exists {
		result.Valid = false
		return result
	}

	// 2. action is bound to this object type
	bound := false
	for _, a := range ot.AllowedActions {
		if a == actionType {
			bound = true
			break
		}
	}
	result.Checks = append(result.Checks, ActionValidationCheck{
		Check:  "action_bound",
		Passed: bound,
		Reason: cond(bound, "", "action "+actionType+" not bound to object type "+objectType),
	})
	if !bound {
		result.Valid = false
		return result
	}

	// 3. action is enabled in registry
	enabled := v.actionEnabled[actionType]
	result.Checks = append(result.Checks, ActionValidationCheck{
		Check:  "action_enabled",
		Passed: enabled,
		Reason: cond(enabled, "", "action "+actionType+" is disabled in registry"),
	})
	if !enabled {
		result.Valid = false
		return result
	}

	// 4. role is authorized
	allowedRoles, hasRoles := v.allowedBy[actionType]
	roleAuthorized := false
	if hasRoles {
		for _, r := range allowedRoles {
			if r == role {
				roleAuthorized = true
				break
			}
		}
	}
	result.Checks = append(result.Checks, ActionValidationCheck{
		Check:  "role_authorized",
		Passed: roleAuthorized,
		Reason: cond(roleAuthorized, "", "role "+role+" not authorized for action "+actionType),
	})
	if !roleAuthorized {
		result.Valid = false
		return result
	}

	return result
}

// ValidatePayload checks that the payload contains all required fields defined
// in the schema. The schema is a map of field name → type string (e.g. "string",
// "number", "boolean"). Returns a validation result with per-field checks.
func (v *ActionBindingValidator) ValidatePayload(schema map[string]any, payload map[string]any) *ActionValidationResult {
	result := &ActionValidationResult{
		Valid:  true,
		Checks: make([]ActionValidationCheck, 0),
	}

	if len(schema) == 0 {
		result.Checks = append(result.Checks, ActionValidationCheck{
			Check:  "payload_schema",
			Passed: true,
			Reason: "",
		})
		return result
	}

	for field, expectedType := range schema {
		val, exists := payload[field]
		if !exists {
			result.Checks = append(result.Checks, ActionValidationCheck{
				Check:  "required_field_" + field,
				Passed: false,
				Reason: "required field " + field + " is missing from payload",
			})
			result.Valid = false
			continue
		}

		typeStr, ok := expectedType.(string)
		if !ok {
			// Skip type check if schema definition is not a type string
			continue
		}

		typeOK := checkPayloadType(val, typeStr)
		if !typeOK {
			result.Checks = append(result.Checks, ActionValidationCheck{
				Check:  "field_type_" + field,
				Passed: false,
				Reason: "field " + field + " expected type " + typeStr,
			})
			result.Valid = false
		}
	}

	return result
}

// ValidateApproval checks pre-execution approval constraints. If the action
// type requires_approval, isApproved must be true or the validation fails.
// Actions not in the registry are treated as not requiring approval.
func (v *ActionBindingValidator) ValidateApproval(actionType string, isApproved bool) *ActionValidationResult {
	result := &ActionValidationResult{
		Valid:  true,
		Checks: make([]ActionValidationCheck, 0),
	}

	requiresApproval := v.actionEnabled[actionType] && len(v.allowedBy[actionType]) > 0

	// High-risk actions always require approval
	if v.actionEnabled[actionType] {
		for _, role := range v.allowedBy[actionType] {
			if role == "admin" || role == "manager" {
				requiresApproval = true
				break
			}
		}
	}

	if requiresApproval && !isApproved {
		result.Checks = append(result.Checks, ActionValidationCheck{
			Check:  "approval_required",
			Passed: false,
			Reason: "action " + actionType + " requires approval but isApproved=false",
		})
		result.Valid = false
		return result
	}

	result.Checks = append(result.Checks, ActionValidationCheck{
		Check:  "approval_required",
		Passed: true,
		Reason: "",
	})
	return result
}

func cond(pred bool, t, f string) string {
	if pred {
		return t
	}
	return f
}

// checkPayloadType verifies a Go value matches an expected type string.
func checkPayloadType(val any, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := val.(string)
		return ok
	case "number", "float":
		switch val.(type) {
		case float64, float32, int, int64, int32:
			return true
		}
		return false
	case "int", "integer":
		switch val.(type) {
		case int, int64, int32, float64:
			return true
		}
		return false
	case "bool", "boolean":
		_, ok := val.(bool)
		return ok
	case "object", "map":
		_, ok := val.(map[string]any)
		return ok
	case "array", "slice":
		_, ok := val.([]any)
		return ok
	default:
		return true // unknown types pass through
	}
}
