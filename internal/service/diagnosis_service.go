package service

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"baxi/internal/api/dto"
)

// Diagnoser is the interface for cross-source request tracing.
type Diagnoser interface {
	DiagnoseByRequestID(requestID string) (*dto.DiagnosisResponse, error)
}

// DiagnosisService performs cross-source request tracing by reading
// error.log (JSONL), audit_dispatch.csv, and audit_feishu.csv.
type DiagnosisService struct {
	errorLogPath     string
	auditCSVPath     string
	auditFeishuPath  string
}

// NewDiagnosisService creates a new DiagnosisService with the given file paths.
func NewDiagnosisService(errorLogPath, auditCSVPath, auditFeishuPath string) *DiagnosisService {
	return &DiagnosisService{
		errorLogPath:    errorLogPath,
		auditCSVPath:    auditCSVPath,
		auditFeishuPath: auditFeishuPath,
	}
}

// DiagnoseByRequestID searches error.log, audit_dispatch.csv, and audit_feishu.csv
// for entries matching the given request_id and returns a structured diagnosis.
func (s *DiagnosisService) DiagnoseByRequestID(requestID string) (*dto.DiagnosisResponse, error) {
	errorEntries := s.searchJSONL(s.errorLogPath, requestID)
	auditEntries := s.searchCSV(s.auditCSVPath, requestID)
	feishuEntries := s.searchCSV(s.auditFeishuPath, requestID)

	var relatedLogs []dto.DiagnosisLogEntry

	for _, e := range errorEntries {
		relatedLogs = append(relatedLogs, dto.DiagnosisLogEntry{
			Source:    "error.log",
			Ts:        getString(e, "timestamp", "ts"),
			ErrorCode: getString(e, "error_code"),
			Message:   getString(e, "message"),
			Diagnosis: getString(e, "diagnosis"),
		})
	}

	for _, a := range auditEntries {
		relatedLogs = append(relatedLogs, dto.DiagnosisLogEntry{
			Source:    "audit_dispatch.csv",
			Timestamp: a["timestamp"],
			OutboxID:  a["outbox_id"],
			Status:    a["status"],
			Error:     a["error"],
		})
	}

	for _, a := range feishuEntries {
		relatedLogs = append(relatedLogs, dto.DiagnosisLogEntry{
			Source:    "audit_feishu.csv",
			Timestamp: a["timestamp"],
			Action:    a["action"],
			Status:    a["status"],
		})
	}

	if len(relatedLogs) == 0 {
		return nil, nil
	}

	primaryError := make(map[string]interface{})
	if len(errorEntries) > 0 {
		primaryError = errorEntries[0]
	}

	summary := getString(primaryError, "message")
	if summary == "" {
		summary = "No error message recorded"
	}

	return &dto.DiagnosisResponse{
		RequestID:       requestID,
		Summary:         summary,
		ErrorCode:       getString(primaryError, "error_code"),
		Diagnosis:       getString(primaryError, "diagnosis"),
		SuggestedAction: getString(primaryError, "suggested_action"),
		RelatedLogs:     relatedLogs,
	}, nil
}

func (s *DiagnosisService) searchJSONL(filepath, requestID string) []map[string]interface{} {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()

	var results []map[string]interface{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		if rid, ok := obj["request_id"].(string); ok && rid == requestID {
			results = append(results, obj)
		}
	}

	return results
}

func (s *DiagnosisService) searchCSV(filepath, requestID string) []map[string]string {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable fields
	headers, err := reader.Read()
	if err != nil {
		return nil
	}

	var results []map[string]string
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = record[i]
			}
		}

		if row["request_id"] == requestID {
			results = append(results, row)
		}
	}

	return results
}

func getString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch val := v.(type) {
			case string:
				return val
			case fmt.Stringer:
				return val.String()
			}
		}
	}
	return ""
}
