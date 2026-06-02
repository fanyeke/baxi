package decision

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"baxi/internal/repository/common"
	"baxi/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLLMDecisionTest starts a Postgres container, applies the DDL,
// and returns a LLMDecisionRepository ready for testing.
func setupLLMDecisionTest(t *testing.T) *LLMDecisionRepository {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping database test in short mode")
	}

	ctx := context.Background()

	pg, err := testutil.StartPostgres(ctx)
	require.NoError(t, err, "StartPostgres")

	t.Cleanup(func() {
		assert.NoError(t, pg.Terminate(context.Background()), "Terminate")
	})

	pool, err := pgxpool.New(ctx, pg.ConnectionString())
	require.NoError(t, err, "pgxpool.New")

	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, decDDL)
	require.NoError(t, err, "decDDL")

	return NewLLMDecisionRepository(common.NewPoolProvider(pool))
}

func TestLLMDecisionCreateAndGet(t *testing.T) {
	repo := setupLLMDecisionTest(t)
	ctx := context.Background()

	// Foreign key: need a decision_case first
	_, err := repo.Exec(ctx, `INSERT INTO ai.decision_case (case_id, status) VALUES ('ld-case-1', 'open')`)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Microsecond)
	confidence := 0.95
	sev := "high"
	recipe := "recipe-1"
	chash := "abc123"
	decisionJSON := json.RawMessage(`{"action":"approve"}`)

	d := &LLMDecision{
		DecisionID:   "ld-1",
		CaseID:       "ld-case-1",
		RecipeID:     &recipe,
		ContextHash:  &chash,
		DecisionJSON: &decisionJSON,
		Severity:     &sev,
		Confidence:   &confidence,
		CreatedAt:    now,
	}

	err = repo.CreateDecision(ctx, d)
	require.NoError(t, err)

	got, err := repo.GetDecisionByID(ctx, "ld-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "ld-1", got.DecisionID)
	assert.Equal(t, "ld-case-1", got.CaseID)
	assert.Equal(t, "recipe-1", *got.RecipeID)
	assert.Equal(t, "abc123", *got.ContextHash)
	assert.Equal(t, "high", *got.Severity)
	assert.Equal(t, 0.95, *got.Confidence)
	assert.WithinDuration(t, now, got.CreatedAt, time.Second)
	require.NotNil(t, got.DecisionJSON)

	// Compare JSON semantically: PostgreSQL JSONB may normalize whitespace
	var expected, actual interface{}
	require.NoError(t, json.Unmarshal([]byte(`{"action":"approve"}`), &expected))
	require.NoError(t, json.Unmarshal(*got.DecisionJSON, &actual))
	assert.Equal(t, expected, actual)
}

func TestLLMDecisionGetByIDNotFound(t *testing.T) {
	repo := setupLLMDecisionTest(t)
	ctx := context.Background()

	got, err := repo.GetDecisionByID(ctx, "nonexistent")
	require.Error(t, err)
	assert.Nil(t, got)
}

func TestLLMDecisionCreateWithNullables(t *testing.T) {
	repo := setupLLMDecisionTest(t)
	ctx := context.Background()

	// Foreign key: need a decision_case first
	_, err := repo.Exec(ctx, `INSERT INTO ai.decision_case (case_id, status) VALUES ('ld-case-2', 'open')`)
	require.NoError(t, err)

	d := &LLMDecision{
		DecisionID: "ld-2",
		CaseID:     "ld-case-2",
		CreatedAt:  time.Now().UTC(),
	}

	err = repo.CreateDecision(ctx, d)
	require.NoError(t, err)

	got, err := repo.GetDecisionByID(ctx, "ld-2")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, "ld-2", got.DecisionID)
	assert.Equal(t, "ld-case-2", got.CaseID)
	assert.Nil(t, got.RecipeID)
	assert.Nil(t, got.ContextHash)
	assert.Nil(t, got.DecisionJSON)
	assert.Nil(t, got.Severity)
	assert.Nil(t, got.Confidence)
	assert.False(t, got.CreatedAt.IsZero())
}
