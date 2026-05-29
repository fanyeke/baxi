// Package feature provides feature flag management for controlling new/old logic switching.
package feature

import (
	"os"
	"strings"
)

type Flag int

const (
	FlagOntologyAwareRepo Flag = iota
	FlagMarkingService
	FlagDecisionLineageService
	FlagNewContextBuilder
	FlagDualWrite
	FlagGoPrimaryWrite
	flagCount
)

var flagNames = map[Flag]string{
	FlagOntologyAwareRepo:      "USE_ONTOLOGY_AWARE_REPO",
	FlagMarkingService:         "USE_MARKING_SERVICE",
	FlagDecisionLineageService: "USE_DECISION_LINEAGE_SERVICE",
	FlagNewContextBuilder:      "USE_NEW_CONTEXT_BUILDER",
	FlagDualWrite:              "USE_DUAL_WRITE",
	FlagGoPrimaryWrite:         "USE_GO_PRIMARY_WRITE",
}

// FeatureFlags holds all feature flag values.
// All flags default to false (off) when their environment variables are not set.
type FeatureFlags struct {
	OntologyAwareRepo      bool
	MarkingService         bool
	DecisionLineageService bool
	NewContextBuilder      bool
	DualWrite              bool
	GoPrimaryWrite         bool
}

func LoadFlags() *FeatureFlags {
	return &FeatureFlags{
		OntologyAwareRepo:      parseBoolEnv(flagNames[FlagOntologyAwareRepo]),
		MarkingService:         parseBoolEnv(flagNames[FlagMarkingService]),
		DecisionLineageService: parseBoolEnv(flagNames[FlagDecisionLineageService]),
		NewContextBuilder:      parseBoolEnv(flagNames[FlagNewContextBuilder]),
		DualWrite:              parseBoolEnv(flagNames[FlagDualWrite]),
		GoPrimaryWrite:         parseBoolEnv(flagNames[FlagGoPrimaryWrite]),
	}
}

func (f *FeatureFlags) IsEnabled(flag Flag) bool {
	if f == nil {
		return false
	}
	switch flag {
	case FlagOntologyAwareRepo:
		return f.OntologyAwareRepo
	case FlagMarkingService:
		return f.MarkingService
	case FlagDecisionLineageService:
		return f.DecisionLineageService
	case FlagNewContextBuilder:
		return f.NewContextBuilder
	case FlagDualWrite:
		return f.DualWrite
	case FlagGoPrimaryWrite:
		return f.GoPrimaryWrite
	default:
		return false
	}
}

func parseBoolEnv(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
