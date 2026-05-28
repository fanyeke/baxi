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
// then loads data using a two-step approach:
//  1. COPY CSV into a temporary staging table (no PK constraints).
//  2. INSERT from staging INTO target with ON CONFLICT DO NOTHING.
//
// The staging step handles duplicate PK rows in the source CSV (which exist
// in the real Olist data — e.g. 789 duplicate review_id values in
// olist_order_reviews_dataset.csv) while preserving COPY's bulk-load speed.
//
// Columns not present in the CSV (e.g. ingested_at, source_file, raw_hash)
// receive their default values via LIKE INCLUDING DEFAULTS.
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
	// Also strip any UTF-8 BOM from the first column (common in Kaggle CSVs).
	rawCols := strings.Split(strings.TrimRight(header, "\r\n"), ",")
	cols := make([]string, 0, len(rawCols))
	for i, c := range rawCols {
		c = strings.TrimSpace(c)
		c = strings.Trim(c, `"`)
		// Strip BOM from the first column name only.
		if i == 0 {
			c = strings.TrimPrefix(c, "\ufeff")
			c = strings.TrimPrefix(c, "\xef\xbb\xbf")
		}
		if c != "" {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return 0, fmt.Errorf("empty csv header in %s", csvPath)
	}

	parts := strings.Split(tableName, ".")
	targetIdent := pgx.Identifier(parts).Sanitize()

	stagingName := "_staging_" + strings.Join(parts, "_")
	stagingIdent := pgx.Identifier{stagingName}.Sanitize()

	if _, err := tx.Exec(ctx, fmt.Sprintf(
		"CREATE TEMP TABLE %s (LIKE %s INCLUDING DEFAULTS) ON COMMIT DROP",
		stagingIdent, targetIdent,
	)); err != nil {
		return 0, fmt.Errorf("create staging table for %s: %w", tableName, err)
	}

	// Build quoted column list for COPY.
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = pgx.Identifier{c}.Sanitize()
	}

	copySQL := fmt.Sprintf("COPY %s (%s) FROM STDIN (FORMAT CSV, HEADER true, NULL '')",
		stagingIdent,
		strings.Join(quoted, ", "))

	// Rewind to beginning and re-read the file including the header.
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek %s: %w", csvPath, err)
	}

	if _, err := tx.Conn().PgConn().CopyFrom(ctx, file, copySQL); err != nil {
		return 0, fmt.Errorf("copy into staging %s: %w", csvPath, err)
	}

	result, err := tx.Exec(ctx, fmt.Sprintf(
		"INSERT INTO %s SELECT * FROM %s ON CONFLICT DO NOTHING",
		targetIdent, stagingIdent,
	))
	if err != nil {
		return 0, fmt.Errorf("insert from staging into %s: %w", tableName, err)
	}

	return result.RowsAffected(), nil
}
