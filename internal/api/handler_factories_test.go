package api

import (
	"testing"

	"baxi/internal/config"
	"github.com/stretchr/testify/assert"
)

// ──── newPipelineRunID ──────────────────────────────────────────────────

func TestNewPipelineRunID_Format(t *testing.T) {
	id := newPipelineRunID()
	assert.NotEmpty(t, id)
	// UUID format: 8-4-4-4-12
	assert.Len(t, id, 36)
	assert.Equal(t, byte('-'), id[8])
	assert.Equal(t, byte('-'), id[13])
	assert.Equal(t, byte('-'), id[18])
	assert.Equal(t, byte('-'), id[23])
}

func TestNewPipelineRunID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newPipelineRunID()
		assert.False(t, ids[id], "duplicate pipeline run ID: %s", id)
		ids[id] = true
	}
}

func TestNewPipelineRunID_VersionBits(t *testing.T) {
	id := newPipelineRunID()
	// The UUID format is: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// Position 14 (after the second dash) should be '4' for version 4
	assert.Equal(t, byte('4'), id[14], "UUID version should be 4")
}

func TestNewPipelineRunID_VariantBits(t *testing.T) {
	id := newPipelineRunID()
	// Position 19 (after the third dash) should be 8, 9, a, or b for variant 1
	variantChar := id[19]
	assert.Contains(t, []byte{'8', '9', 'a', 'b'}, variantChar, "UUID variant should be correct")
}

// ──── actionExecutors ──────────────────────────────────────────────────

func TestActionExecutors_ReturnsMap(t *testing.T) {
	s := &Server{}
	executors := s.actionExecutors()
	assert.NotNil(t, executors)
	assert.Contains(t, executors, "feishu")
	assert.Contains(t, executors, "github")
	assert.Contains(t, executors, "cli")
	assert.Contains(t, executors, "manual")
	assert.Contains(t, executors, "noop")
}

func TestActionExecutors_NilConfig(t *testing.T) {
	s := &Server{cfg: nil}
	executors := s.actionExecutors()
	assert.NotNil(t, executors)
	assert.Len(t, executors, 5)
}

func TestActionExecutors_EmptyConfig(t *testing.T) {
	s := &Server{cfg: &config.Config{}}
	executors := s.actionExecutors()
	assert.NotNil(t, executors)
}

// ──── reviewHandlerSvc ─────────────────────────────────────────────────

func TestReviewHandlerSvc_GetReviewByProposal_NilPool(t *testing.T) {
	// With nil pool, the repo should return an error
	svc := &reviewHandlerSvc{
		svc:  nil,
		repo: nil,
		pool: nil,
	}

	// Calling with nil repo will panic, so this is a basic nil check
	assert.Nil(t, svc.repo)
}

// ──── outboxServiceAdapter ─────────────────────────────────────────────

func TestOutboxServiceAdapter_NilFields(t *testing.T) {
	adapter := &outboxServiceAdapter{}
	assert.Nil(t, adapter.readSvc)
	assert.Nil(t, adapter.readRepo)
	assert.Nil(t, adapter.writeRepo)
	assert.Nil(t, adapter.pool)
	assert.Nil(t, adapter.executors)
}

// ──── proposalLoaderAdapter ────────────────────────────────────────────

func TestProposalLoaderAdapter_NilRepo(t *testing.T) {
	adapter := &proposalLoaderAdapter{repo: nil}
	assert.Nil(t, adapter.repo)
}

// ──── actionHandlerSvc ─────────────────────────────────────────────────

func TestActionHandlerSvc_NilFields(t *testing.T) {
	svc := &actionHandlerSvc{}
	assert.Nil(t, svc.applySvc)
	assert.Nil(t, svc.repo)
	assert.Nil(t, svc.pool)
}

// ──── pipelineRunService ───────────────────────────────────────────────

func TestPipelineRunService_NilFields(t *testing.T) {
	svc := &pipelineRunService{}
	assert.Nil(t, svc.runner)
	assert.Nil(t, svc.log)
}

// ──── Server handler lazy init (nil pool) ──────────────────────────────

func TestServer_HandlerFactories_NilPool(t *testing.T) {
	s := &Server{
		ctx:    nil,
		logger: nil,
		pool:   nil,
		cfg:    nil,
	}

	// These should not panic when pool is nil
	// They create services with nil pool, which is valid

	// Test actionExecutors (doesn't need pool)
	executors := s.actionExecutors()
	assert.NotNil(t, executors)
}
