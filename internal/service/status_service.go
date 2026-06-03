package service

import (
	"context"
	"fmt"

	"baxi/internal/model"
	statusRepo "baxi/internal/repository/status"
)

const apiVersion = "0.6.0"

// StatusService aggregates table counts and pipeline run info for the status endpoint.
type StatusService struct {
	repo  *statusRepo.Repository
	dbURL string
}

// NewStatusService creates a new StatusService.
func NewStatusService(repo *statusRepo.Repository, dbURL string) *StatusService {
	return &StatusService{repo: repo, dbURL: dbURL}
}

// GetStatus assembles the full StatusResponse from repository data.
func (s *StatusService) GetStatus(ctx context.Context) (*model.StatusResponse, error) {
	tableCounts, err := s.repo.GetTableCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get table counts: %w", err)
	}

	tables := make(map[string]int, len(tableCounts))
	for _, tc := range tableCounts {
		tables[tc.TableName] = tc.RowCount
	}

	database := model.DatabaseInfo{
		Path:   s.dbURL,
		Exists: true,
		Tables: tables,
	}

	lastRun, err := s.repo.GetLastPipelineRun(ctx)
	if err != nil {
		// If no pipeline runs exist, return nil for last_pipeline_run
		lastRun = nil
	}

	var pipelineRun *model.PipelineRun
	if lastRun != nil {
		pipelineRun = &model.PipelineRun{
			RunID:        lastRun.RunID,
			RunType:      lastRun.RunType,
			Mode:         lastRun.Mode,
			Status:       lastRun.Status,
			StartedAt:    lastRun.StartedAt,
			FinishedAt:   lastRun.FinishedAt,
			InputCount:   lastRun.InputCount,
			OutputCount:  lastRun.OutputCount,
			ErrorMessage: lastRun.ErrorMessage,
		}
	}

	return &model.StatusResponse{
		Database:        database,
		LastPipelineRun: pipelineRun,
		Version:         apiVersion,
	}, nil
}
