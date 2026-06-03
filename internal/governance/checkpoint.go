package governance

import (
	"context"
	"encoding/json"

	governanceRepo "baxi/internal/repository/governance"
)

// CheckpointRule represents a checkpoint rule from config_snapshot.
type CheckpointRule struct {
	Action              string `json:"action"`
	RequiresReason      bool   `json:"requires_reason"`
	RequiresHumanReview bool   `json:"requires_human_review"`
}

// checkpointRulesConfig wraps the checkpoint_rules key from config_snapshot.
type checkpointRulesConfig struct {
	Checkpoints []CheckpointRule `json:"checkpoints"`
}

type checkpointSnapshotRepo interface {
	GetConfigSnapshots(ctx context.Context) ([]governanceRepo.ConfigSnapshotRow, error)
}

type providerAdapter struct {
	provider ConfigSnapshotProvider
}

func (a *providerAdapter) GetConfigSnapshots(ctx context.Context) ([]governanceRepo.ConfigSnapshotRow, error) {
	return a.provider.GetConfigSnapshots(ctx)
}

// CheckpointService provides checkpoint evaluation for sensitive actions.
type CheckpointService struct {
	repo checkpointSnapshotRepo
}

// NewCheckpointService creates a new CheckpointService.
func NewCheckpointService(repo checkpointSnapshotRepo) *CheckpointService {
	return &CheckpointService{repo: repo}
}

// NewCheckpointServiceWithProvider creates a CheckpointService with a provider for testing.
func NewCheckpointServiceWithProvider(provider ConfigSnapshotProvider) *CheckpointService {
	return &CheckpointService{repo: &providerAdapter{provider: provider}}
}

// RequiresCheckpoint checks if an action requires a checkpoint before execution.
// Returns true for actions: "execute_dispatch", "modify_business_policy", "trigger_pipeline".
func (s *CheckpointService) RequiresCheckpoint(ctx context.Context, action string) bool {
	sensitiveActions := map[string]bool{
		"execute_dispatch":       true,
		"modify_business_policy": true,
		"trigger_pipeline":       true,
	}

	if sensitiveActions[action] {
		return true
	}

	// Also check config_snapshot for additional checkpoint rules
	snapshots, err := s.repo.GetConfigSnapshots(ctx)
	if err != nil {
		return false
	}

	for _, snap := range snapshots {
		if snap.ConfigKey == "checkpoint_rules" || snap.ConfigKey == "checkpoint_rules.yml" {
			var cfg checkpointRulesConfig
			if err := json.Unmarshal([]byte(snap.Status), &cfg); err != nil {
				continue
			}
			for _, rule := range cfg.Checkpoints {
				if rule.Action == action {
					return true
				}
			}
		}
	}

	return false
}

// GetRules returns all checkpoint rules loaded from config_snapshot.
func (s *CheckpointService) GetRules(ctx context.Context) []CheckpointRule {
	snapshots, err := s.repo.GetConfigSnapshots(ctx)
	if err != nil {
		return nil
	}

	var rules []CheckpointRule
	for _, snap := range snapshots {
		if snap.ConfigKey == "checkpoint_rules" || snap.ConfigKey == "checkpoint_rules.yml" {
			var cfg checkpointRulesConfig
			if err := json.Unmarshal([]byte(snap.Status), &cfg); err != nil {
				continue
			}
			rules = append(rules, cfg.Checkpoints...)
		}
	}

	// Always include the built-in sensitive actions
	builtIn := []CheckpointRule{
		{Action: "execute_dispatch", RequiresReason: true, RequiresHumanReview: true},
		{Action: "modify_business_policy", RequiresReason: true, RequiresHumanReview: true},
		{Action: "trigger_pipeline", RequiresReason: true, RequiresHumanReview: false},
	}

	seen := make(map[string]bool)
	for _, r := range rules {
		seen[r.Action] = true
	}
	for _, r := range builtIn {
		if !seen[r.Action] {
			rules = append(rules, r)
		}
	}

	if rules == nil {
		rules = []CheckpointRule{}
	}
	return rules
}
