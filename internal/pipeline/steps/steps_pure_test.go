package steps

import (
	"testing"

	"baxi/internal/alert"
)

func TestCountEnabledGlobalRules_Mixed(t *testing.T) {
	rules := []alert.AlertRule{
		{Name: "r1", Enabled: true},
		{Name: "r2", Enabled: false},
		{Name: "r3", Enabled: true},
	}
	n := countEnabledGlobalRules(rules)
	if n != 2 {
		t.Errorf("expected 2, got %d", n)
	}
}

func TestSanitizeJSON_Valid(t *testing.T) {
	got := sanitizeJSON(`{"key":"value"}`)
	if got != `{"key":"value"}` {
		t.Errorf("expected original JSON, got %q", got)
	}
}

func TestSanitizeJSON_Empty(t *testing.T) {
	got := sanitizeJSON("")
	if got != "{}" {
		t.Errorf("expected {}, got %q", got)
	}
}

func TestSanitizeJSON_Invalid(t *testing.T) {
	got := sanitizeJSON("not json")
	if got != "{}" {
		t.Errorf("expected {}, got %q", got)
	}
}

func TestSanitizeJSON_Numeric(t *testing.T) {
	got := sanitizeJSON("42")
	if got != "42" {
		t.Errorf("expected {}, got %q", got)
	}
}

func TestDeriveOwnerRole_Unknown(t *testing.T) {
	got := deriveOwnerRole("unknown_rule")
	if got != "unassigned" {
		t.Errorf("expected 'unassigned', got %q", got)
	}
}

func TestDeriveOwnerRole_Empty(t *testing.T) {
	got := deriveOwnerRole("")
	if got != "unassigned" {
		t.Errorf("expected 'unassigned', got %q", got)
	}
}

func TestIsDimensionalTask_Dimensional(t *testing.T) {
	if !IsDimensionalTask("dimtask-abc") {
		t.Error("expected dimrec- prefix to be dimensional")
	}
}

func TestIsDimensionalTask_NotDimensional(t *testing.T) {
	if IsDimensionalTask("rec-abc") {
		t.Error("expected rec- prefix to NOT be dimensional")
	}
}

func TestIsDimensionalTask_Empty(t *testing.T) {
	if IsDimensionalTask("") {
		t.Error("expected empty string to NOT be dimensional")
	}
}

func TestIsDimensionalTask_DimrecOnly(t *testing.T) {
	if !IsDimensionalTask("dimtask-") {
		t.Error("expected 'dimrec-' to be dimensional")
	}
}
