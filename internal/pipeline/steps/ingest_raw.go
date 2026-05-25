// Package steps implements individual pipeline steps that can be composed
// into a full pipeline run. Each step implements pipeline.Step.
package steps

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"baxi/internal/ingest"
	"baxi/internal/pipeline"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// IngestRawStep loads CSV files into raw PostgreSQL tables.
// For each CSV: it truncates the target table, then COPYs the CSV into it.
// Required files cause a hard error if missing; optional files are skipped
// with a warning.
type IngestRawStep struct {
	loader *ingest.CSVLoader
}

// NewIngestRawStep creates a new IngestRawStep.
func NewIngestRawStep() *IngestRawStep {
	return &IngestRawStep{
		loader: ingest.NewCSVLoader(),
	}
}

// Name returns the step name for audit logging.
func (s *IngestRawStep) Name() string {
	return "ingest_raw"
}

// Run executes the step within the given transaction.
// It iterates all table mappings, truncates each target table, loads the CSV,
// and aggregates input/output row counts.
func (s *IngestRawStep) Run(ctx context.Context, tx pgx.Tx, input pipeline.StepInput) (*pipeline.StepOutput, error) {
	mappings := ingest.AllTableMappings()

	var totalInput, totalOutput int64

	for _, m := range mappings {
		csvPath := filepath.Join(input.DataDir, m.CSVFile)

		if _, err := os.Stat(csvPath); os.IsNotExist(err) {
			if m.Required {
				return nil, fmt.Errorf("required CSV not found: %s", csvPath)
			}
			input.Logger.Warn("optional CSV not found, skipping",
				zap.String("file", m.CSVFile),
				zap.String("table", m.TableName),
			)
			continue
		}

		input.Logger.Info("ingesting",
			zap.String("table", m.TableName),
			zap.String("file", m.CSVFile),
		)

		truncSQL := fmt.Sprintf("TRUNCATE TABLE %s", m.TableName)
		if _, err := tx.Exec(ctx, truncSQL); err != nil {
			return nil, fmt.Errorf("truncate %s: %w", m.TableName, err)
		}

		count, err := copyCSV(ctx, tx, csvPath, m.TableName)
		if err != nil {
			return nil, fmt.Errorf("load %s into %s: %w", m.CSVFile, m.TableName, err)
		}

		totalInput += count
		totalOutput += count

		input.Logger.Info("ingested",
			zap.String("table", m.TableName),
			zap.Int64("rows", count),
		)
	}

	return &pipeline.StepOutput{
		InputCount:  totalInput,
		OutputCount: totalOutput,
	}, nil
}

// copyCSV opens a CSV file, reads the header to determine column names,
// then uses PostgreSQL COPY with an explicit column list so that table
// columns not present in the CSV (e.g. ingested_at, source_file, raw_hash)
// receive their default values.
func copyCSV(ctx context.Context, tx pgx.Tx, csvPath, tableName string) (int64, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, fmt.Errorf("open csv %s: %w", csvPath, err)
	}
	defer file.Close()

	// Read the header row to get column names.
	reader := bufio.NewReader(file)
	header, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("read csv header %s: %w", csvPath, err)
	}

	// Parse column names from the header.
	// CSV headers may or may not be quoted. We split on ',' then strip
	// surrounding whitespace and quotes.
	rawCols := strings.Split(strings.TrimRight(header, "\r\n"), ",")
	cols := make([]string, 0, len(rawCols))
	for _, c := range rawCols {
		c = strings.TrimSpace(c)
		c = strings.Trim(c, `"`)
		if c != "" {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return 0, fmt.Errorf("empty csv header in %s", csvPath)
	}

	// Build quoted column list for COPY.
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}

	ident := pgx.Identifier(strings.Split(tableName, "."))
	copySQL := fmt.Sprintf("COPY %s (%s) FROM STDIN (FORMAT CSV, HEADER true, NULL '')",
		ident.Sanitize(),
		strings.Join(quoted, ", "))

	// Rewind to beginning and re-read the file including the header.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek %s: %w", csvPath, err)
	}

	ct, err := tx.Conn().PgConn().CopyFrom(ctx, file, copySQL)
	if err != nil {
		return 0, fmt.Errorf("copy %s into %s: %w", csvPath, tableName, err)
	}

	return ct.RowsAffected(), nil
}
