package feature

import (
	"os"
	"testing"
)

func TestLoadFlags_Defaults(t *testing.T) {
	flags := LoadFlags()
	if flags == nil {
		t.Fatal("LoadFlags returned nil")
	}
	if flags.OntologyAwareRepo {
		t.Error("OntologyAwareRepo should default to false")
	}
	if flags.MarkingService {
		t.Error("MarkingService should default to false")
	}
	if flags.DecisionLineageService {
		t.Error("DecisionLineageService should default to false")
	}
	if flags.NewContextBuilder {
		t.Error("NewContextBuilder should default to false")
	}
	if flags.DualWrite {
		t.Error("DualWrite should default to false")
	}
	if flags.GoPrimaryWrite {
		t.Error("GoPrimaryWrite should default to false")
	}
}

func TestLoadFlags_EnvVarsEnabled(t *testing.T) {
	envVars := map[string]string{
		"USE_ONTOLOGY_AWARE_REPO":      "true",
		"USE_MARKING_SERVICE":          "1",
		"USE_DECISION_LINEAGE_SERVICE": "yes",
		"USE_NEW_CONTEXT_BUILDER":      "TRUE",
		"USE_DUAL_WRITE":               "YES",
		"USE_GO_PRIMARY_WRITE":         "1",
	}
	for k, v := range envVars {
		t.Setenv(k, v)
	}

	flags := LoadFlags()
	if !flags.OntologyAwareRepo {
		t.Error("OntologyAwareRepo should be true")
	}
	if !flags.MarkingService {
		t.Error("MarkingService should be true")
	}
	if !flags.DecisionLineageService {
		t.Error("DecisionLineageService should be true")
	}
	if !flags.NewContextBuilder {
		t.Error("NewContextBuilder should be true")
	}
	if !flags.DualWrite {
		t.Error("DualWrite should be true")
	}
	if !flags.GoPrimaryWrite {
		t.Error("GoPrimaryWrite should be true")
	}
}

func TestLoadFlags_EnvVarsDisabled(t *testing.T) {
	envVars := map[string]string{
		"USE_ONTOLOGY_AWARE_REPO":      "false",
		"USE_MARKING_SERVICE":          "0",
		"USE_DECISION_LINEAGE_SERVICE": "no",
		"USE_NEW_CONTEXT_BUILDER":      "FALSE",
		"USE_DUAL_WRITE":               "NO",
		"USE_GO_PRIMARY_WRITE":         "anything-else",
	}
	for k, v := range envVars {
		t.Setenv(k, v)
	}

	flags := LoadFlags()
	if flags.OntologyAwareRepo {
		t.Error("OntologyAwareRepo should be false")
	}
	if flags.MarkingService {
		t.Error("MarkingService should be false")
	}
	if flags.DecisionLineageService {
		t.Error("DecisionLineageService should be false")
	}
	if flags.NewContextBuilder {
		t.Error("NewContextBuilder should be false")
	}
	if flags.DualWrite {
		t.Error("DualWrite should be false")
	}
	if flags.GoPrimaryWrite {
		t.Error("GoPrimaryWrite should be false")
	}
}

func TestLoadFlags_PartialEnables(t *testing.T) {
	os.Unsetenv("USE_ONTOLOGY_AWARE_REPO")
	os.Unsetenv("USE_MARKING_SERVICE")
	os.Unsetenv("USE_DECISION_LINEAGE_SERVICE")
	os.Unsetenv("USE_NEW_CONTEXT_BUILDER")
	os.Unsetenv("USE_DUAL_WRITE")
	os.Unsetenv("USE_GO_PRIMARY_WRITE")

	t.Setenv("USE_ONTOLOGY_AWARE_REPO", "true")
	t.Setenv("USE_DUAL_WRITE", "1")

	flags := LoadFlags()
	if !flags.OntologyAwareRepo {
		t.Error("OntologyAwareRepo should be true")
	}
	if flags.MarkingService {
		t.Error("MarkingService should be false")
	}
	if flags.DecisionLineageService {
		t.Error("DecisionLineageService should be false")
	}
	if flags.NewContextBuilder {
		t.Error("NewContextBuilder should be false")
	}
	if !flags.DualWrite {
		t.Error("DualWrite should be true")
	}
	if flags.GoPrimaryWrite {
		t.Error("GoPrimaryWrite should be false")
	}
}

func TestIsEnabled(t *testing.T) {
	flags := &FeatureFlags{
		OntologyAwareRepo: true,
		DualWrite:         true,
	}

	tests := []struct {
		flag Flag
		want bool
	}{
		{FlagOntologyAwareRepo, true},
		{FlagMarkingService, false},
		{FlagDecisionLineageService, false},
		{FlagNewContextBuilder, false},
		{FlagDualWrite, true},
		{FlagGoPrimaryWrite, false},
		{Flag(999), false},
	}

	for _, tt := range tests {
		if got := flags.IsEnabled(tt.flag); got != tt.want {
			t.Errorf("IsEnabled(%v) = %v, want %v", tt.flag, got, tt.want)
		}
	}
}

func TestIsEnabled_NilReceiver(t *testing.T) {
	var flags *FeatureFlags
	if flags.IsEnabled(FlagOntologyAwareRepo) {
		t.Error("IsEnabled on nil should return false")
	}
}

func TestParseBoolEnv(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{"empty", "", false},
		{"true_lowercase", "true", true},
		{"true_uppercase", "TRUE", true},
		{"true_mixed", "True", true},
		{"one", "1", true},
		{"yes_lowercase", "yes", true},
		{"yes_uppercase", "YES", true},
		{"false_string", "false", false},
		{"zero", "0", false},
		{"no_string", "no", false},
		{"random_string", "enabled", false},
		{"whitespace_true", "  true  ", true},
		{"whitespace_one", " 1 ", true},
		{"whitespace_yes", " yes ", true},
		{"whitespace_true_upper", "  TRUE  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TEST_BOOL_ENV", tt.val)
			if got := parseBoolEnv("TEST_BOOL_ENV"); got != tt.want {
				t.Errorf("parseBoolEnv(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestParseBoolEnv_EmptyKey(t *testing.T) {
	os.Unsetenv("NONEXISTENT_KEY")
	if parseBoolEnv("NONEXISTENT_KEY") {
		t.Error("parseBoolEnv for unset key should return false")
	}
}

func TestLoadFlags_WhitespaceHandling(t *testing.T) {
	t.Setenv("USE_ONTOLOGY_AWARE_REPO", "  true  ")
	t.Setenv("USE_MARKING_SERVICE", "\tyes\t")
	t.Setenv("USE_DECISION_LINEAGE_SERVICE", " TRUE ")

	flags := LoadFlags()
	if !flags.OntologyAwareRepo {
		t.Error("OntologyAwareRepo should handle whitespace")
	}
	if !flags.MarkingService {
		t.Error("MarkingService should handle tabs")
	}
	if !flags.DecisionLineageService {
		t.Error("DecisionLineageService should handle whitespace")
	}
}

func TestIsEnabled_AllFlags(t *testing.T) {
	flags := &FeatureFlags{
		OntologyAwareRepo:      true,
		MarkingService:         true,
		DecisionLineageService: true,
		NewContextBuilder:      true,
		DualWrite:              true,
		GoPrimaryWrite:         true,
	}

	for i := range flagCount {
		if !flags.IsEnabled(Flag(i)) {
			t.Errorf("IsEnabled(%v) should be true", i)
		}
	}
}
