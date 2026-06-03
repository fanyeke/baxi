package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"baxi/internal/model"
	logRepo "baxi/internal/repository/log"
)

// LogService handles business logic for log-related operations.
type LogService struct {
	repo *logRepo.Repository
}

// NewLogService creates a new LogService.
func NewLogService(repo *logRepo.Repository) *LogService {
	return &LogService{repo: repo}
}

// ListRecent retrieves a combined view of recent logs from multiple tables.
func (s *LogService) ListRecent(ctx context.Context, limit, offset int) (*model.LogListResponse, error) {
	rows, total, err := s.repo.ListRecentLogs(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list recent logs: %w", err)
	}

	items := make([]model.LogItem, len(rows))
	for i, row := range rows {
		items[i] = model.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &model.LogListResponse{Items: items, Total: total}, nil
}

// ListErrors retrieves error logs from error_log and failed pipeline step runs.
func (s *LogService) ListErrors(ctx context.Context, limit, offset int) (*model.LogListResponse, error) {
	rows, total, err := s.repo.ListErrorLogs(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list error logs: %w", err)
	}

	items := make([]model.LogItem, len(rows))
	for i, row := range rows {
		items[i] = model.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &model.LogListResponse{Items: items, Total: total}, nil
}

// ListAudit retrieves business audit trail entries.
func (s *LogService) ListAudit(ctx context.Context, limit, offset int) (*model.LogListResponse, error) {
	rows, total, err := s.repo.ListAuditLogs(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	items := make([]model.LogItem, len(rows))
	for i, row := range rows {
		items[i] = model.LogItem{
			LogType:   row.LogType,
			Level:     row.Level,
			Message:   row.Message,
			RequestID: row.RequestID,
			CreatedAt: row.CreatedAt,
		}
	}

	return &model.LogListResponse{Items: items, Total: total}, nil
}

// tailJSONL reads the last N JSON lines from a JSONL file.
// Returns entries newest first. Returns empty slice for missing or empty files.
func tailJSONL(filepath string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		return []map[string]any{}, nil
	}

	info, err := os.Stat(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	if info.Size() == 0 {
		return []map[string]any{}, nil
	}

	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const chunkSize = 4096
	size := info.Size()
	entries := make([]map[string]any, 0, limit)

	lineBuffer := make([]byte, 0, 256)

	pos := size
	for pos > 0 && len(entries) < limit {
		readSize := int64(chunkSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		_, err := f.Seek(pos, io.SeekStart)
		if err != nil {
			return nil, err
		}

		chunk := make([]byte, readSize)
		n, err := io.ReadFull(f, chunk)
		if err != nil && err != io.EOF {
			return nil, err
		}
		chunk = chunk[:n]

		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == '\n' {
				if len(lineBuffer) > 0 {
					var sb strings.Builder
					sb.Grow(len(lineBuffer))
					for j := len(lineBuffer) - 1; j >= 0; j-- {
						sb.WriteByte(lineBuffer[j])
					}
					line := strings.TrimSpace(sb.String())
					if line != "" {
						var obj map[string]any
						if err := json.Unmarshal([]byte(line), &obj); err == nil {
							entries = append(entries, obj)
							if len(entries) >= limit {
								return entries, nil
							}
						}
					}
					lineBuffer = lineBuffer[:0]
				}
			} else {
				lineBuffer = append(lineBuffer, chunk[i])
			}
		}
	}

	if len(lineBuffer) > 0 && len(entries) < limit {
		var sb strings.Builder
		sb.Grow(len(lineBuffer))
		for j := len(lineBuffer) - 1; j >= 0; j-- {
			sb.WriteByte(lineBuffer[j])
		}
		line := strings.TrimSpace(sb.String())
		if line != "" {
			var obj map[string]any
			if err := json.Unmarshal([]byte(line), &obj); err == nil {
				entries = append(entries, obj)
			}
		}
	}

	return entries, nil
}

// ReadLogErrors parses error log JSONL, filters by request_id, returns last N entries.
func (s *LogService) ReadLogErrors(errorLogPath string, requestID *string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		return []map[string]any{}, nil
	}
	if limit > 500 {
		limit = 500
	}

	entries, err := tailJSONL(errorLogPath, limit)
	if err != nil {
		return nil, err
	}

	if requestID != nil && *requestID != "" {
		filtered := make([]map[string]any, 0, len(entries))
		for _, e := range entries {
			if rid, ok := e["request_id"].(string); ok && rid == *requestID {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	return entries, nil
}

// ReadLogRecent parses API log JSONL, returns last N entries.
func (s *LogService) ReadLogRecent(apiLogPath string, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		return []map[string]any{}, nil
	}
	if limit > 500 {
		limit = 500
	}
	return tailJSONL(apiLogPath, limit)
}

// ReadAuditLogs parses CSV audit log, filters by outbox_id and status, returns sorted by timestamp desc.
func (s *LogService) ReadAuditLogs(auditCSVPath string, outboxID, status *string, limit int) ([]map[string]string, error) {
	if limit <= 0 {
		return []map[string]string{}, nil
	}
	if limit > 500 {
		limit = 500
	}

	f, err := os.Open(auditCSVPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return []map[string]string{}, nil
		}
		return nil, err
	}

	var entries []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		entry := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				entry[h] = record[i]
			}
		}
		entries = append(entries, entry)
	}

	if outboxID != nil && *outboxID != "" {
		filtered := make([]map[string]string, 0, len(entries))
		for _, e := range entries {
			if e["outbox_id"] == *outboxID {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if status != nil && *status != "" {
		filtered := make([]map[string]string, 0, len(entries))
		for _, e := range entries {
			if e["status"] == *status {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i]["timestamp"] > entries[j]["timestamp"]
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}
