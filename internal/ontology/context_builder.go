package ontology

import (
	"context"
	"fmt"
	"sort"
)

// ContextRequest describes the input parameters for building an LLM-safe context.
type ContextRequest struct {
	ObjectType string `json:"object_type"`
	ObjectID   string `json:"object_id"`
	Purpose    string `json:"purpose"`
	Role       string `json:"role"`
}

// LLMSafeContext is the sanitized, role-aware view of an object for LLM consumption.
type LLMSafeContext struct {
	ObjectType       string                 `json:"object_type"`
	ObjectID         string                 `json:"object_id"`
	Properties       map[string]interface{} `json:"properties"`
	RedactedFields   []RedactedField        `json:"redacted_fields,omitempty"`
	AllowedActions   []string               `json:"allowed_actions,omitempty"`
	ForbiddenActions []string               `json:"forbidden_actions,omitempty"`
}

// RedactedField records a single redacted property and the reason for redaction.
type RedactedField struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

// ContextBuilder builds LLM-safe contexts using the object registry for schema
// and sensitivity information.
type ContextBuilder struct {
	registry *ObjectRegistry
}

// NewContextBuilder creates a ContextBuilder backed by the given registry.
func NewContextBuilder(registry *ObjectRegistry) *ContextBuilder {
	return &ContextBuilder{registry: registry}
}

// BuildLLMSafeContext constructs a sanitized, deterministic view of the requested
// object for the given role.  PII and sensitive fields are redacted according to
// role-based rules, and allowed/forbidden action lists are populated.
func (b *ContextBuilder) BuildLLMSafeContext(ctx context.Context, input ContextRequest) (*LLMSafeContext, error) {
	ot, err := b.registry.GetObjectType(input.ObjectType)
	if err != nil {
		return nil, fmt.Errorf("context_builder: %w", err)
	}

	props := make(map[string]interface{})
	redactedProps, redactedFields := b.redactProperties(input.ObjectType, input.Role, props)
	allowed, forbidden := buildActionLists(input.Role, ot.AllowedActions)

	return &LLMSafeContext{
		ObjectType:       input.ObjectType,
		ObjectID:         input.ObjectID,
		Properties:       redactedProps,
		RedactedFields:   redactedFields,
		AllowedActions:   allowed,
		ForbiddenActions: forbidden,
	}, nil
}

// redactProperties filters the supplied properties according to the role-based
// sensitivity rules defined in the object registry.  It returns the allowed
// properties and a sorted list of redacted fields.
func (b *ContextBuilder) redactProperties(objectType string, role string, properties map[string]interface{}) (map[string]interface{}, []RedactedField) {
	effectiveRole := role
	if effectiveRole == "" {
		effectiveRole = "agent_readonly"
	}

	if effectiveRole == "admin" {
		return copyMap(properties), nil
	}

	var schemaProps map[string]ObjectProperty
	if ot, err := b.registry.GetObjectType(objectType); err == nil {
		schemaProps = ot.Properties
	}

	allowed := make(map[string]interface{}, len(properties))
	var redacted []RedactedField

	for name, value := range properties {
		sensitivity := ""
		if p, ok := schemaProps[name]; ok {
			sensitivity = p.Sensitivity
		}

		if shouldRedact(effectiveRole, sensitivity) {
			redacted = append(redacted, RedactedField{
				Field:  name,
				Reason: fmt.Sprintf("sensitivity=%q exceeds role %q threshold", sensitivity, effectiveRole),
			})
			continue
		}

		allowed[name] = value
	}

	sort.Slice(redacted, func(i, j int) bool {
		return redacted[i].Field < redacted[j].Field
	})

	return allowed, redacted
}

// sensitivityLevel extracts the numeric level from an L0-L4 code.
// Returns -1 if the input is not a valid level code.
func sensitivityLevel(s string) int {
	if len(s) == 2 && s[0] == 'L' && s[1] >= '0' && s[1] <= '4' {
		return int(s[1] - '0')
	}
	return -1
}

// shouldRedact returns true if the given sensitivity classification must be
// redacted for the specified role.
func shouldRedact(role, sensitivity string) bool {
	level := sensitivityLevel(sensitivity)
	if level < 0 {
		return false
	}
	switch role {
	case "admin":
		return false
	case "analyst":
		return level >= 3
	case "viewer", "agent_readonly", "":
		return level >= 2
	default:
		return level >= 2
	}
}

// buildActionLists computes the allowed and forbidden action names for a role.
// The registry's allowed actions are intersected with the role's permitted set.
func buildActionLists(role string, registryActions []string) (allowed []string, forbidden []string) {
	switch role {
	case "agent_readonly", "", "viewer":
		for _, a := range registryActions {
			switch a {
			case "create_followup_task", "notify_owner":
				allowed = append(allowed, a)
			default:
				forbidden = append(forbidden, a)
			}
		}
		canonicalForbidden := []string{"execute_dispatch", "modify_business_policy", "modify_raw_data", "trigger_pipeline"}
		for _, cf := range canonicalForbidden {
			found := false
			for _, f := range forbidden {
				if f == cf {
					found = true
					break
				}
			}
			if !found {
				forbidden = append(forbidden, cf)
			}
		}
	case "analyst":
		for _, a := range registryActions {
			switch a {
			case "read":
				allowed = append(allowed, a)
			default:
				forbidden = append(forbidden, a)
			}
		}
	case "admin":
		allowed = append([]string(nil), registryActions...)
	default:
		for _, a := range registryActions {
			switch a {
			case "create_followup_task", "notify_owner":
				allowed = append(allowed, a)
			default:
				forbidden = append(forbidden, a)
			}
		}
		canonicalForbidden := []string{"execute_dispatch", "modify_business_policy", "modify_raw_data", "trigger_pipeline"}
		for _, cf := range canonicalForbidden {
			found := false
			for _, f := range forbidden {
				if f == cf {
					found = true
					break
				}
			}
			if !found {
				forbidden = append(forbidden, cf)
			}
		}
	}

	sort.Strings(allowed)
	sort.Strings(forbidden)
	return allowed, forbidden
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
