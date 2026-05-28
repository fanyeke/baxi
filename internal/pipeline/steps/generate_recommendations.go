package steps

import (
	"context"
	"fmt"

	"baxi/internal/pipeline"
	"baxi/internal/recommendation"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// GenerateRecommendationsStep reads alerts from ops.metric_alert and generates
// recommendation records in ops.recommendation using template-based generation
// (no LLM calls). All recommendations have decision_source='rule_based'.
//
// The step uses INSERT … ON CONFLICT (recommendation_id) DO NOTHING for
// idempotency — re-running produces the same recommendations without duplicates.
// Expected output: 36 recommendations (matching baseline).
type GenerateRecommendationsStep struct{}

// NewGenerateRecommendationsStep creates a new GenerateRecommendationsStep.
func NewGenerateRecommendationsStep() *GenerateRecommendationsStep {
	return &GenerateRecommendationsStep{}
}

// Name returns the step name for audit logging.
func (s *GenerateRecommendationsStep) Name() string {
	return "generate_recommendations"
}

// Run executes the recommendation generator within the given transaction.
// The tx must NOT be committed or rolled back by this step — the Runner
// handles commit/rollback based on the error return.
func (s *GenerateRecommendationsStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	// Count source alerts for input
	var inputCount int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM ops.metric_alert`).Scan(&inputCount); err != nil {
		return nil, fmt.Errorf("count ops.metric_alert: %w", err)
	}

	outputCount, err := recommendation.Generate(ctx, tx, input.Logger)
	if err != nil {
		return nil, fmt.Errorf("generate_recommendations: %w", err)
	}

	input.Logger.Info("recommendation generation complete",
		zap.Int64("input_alerts", inputCount),
		zap.Int64("output_recommendations", outputCount),
	)

	return &pipeline.StepOutput{
		InputCount:  inputCount,
		OutputCount: outputCount,
	}, nil
}
