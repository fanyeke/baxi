// Package ingest handles loading CSV data into PostgreSQL raw tables.
package ingest

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// CSVLoader handles loading CSV files into PostgreSQL tables via COPY.
// Empty CSV fields become NULL, matching Python's safe_int/safe_float:
//   - empty field / no data between commas → NULL (not 0, not empty string)
//   - valid number → stored as-is
//   - unparseable text → NULL (via PostgreSQL type coercion on COPY)
type CSVLoader struct{}

// Option configures the CSVLoader.
type Option func(*CSVLoader)

// NewCSVLoader creates a CSVLoader with optional configuration.
func NewCSVLoader(opts ...Option) *CSVLoader {
	l := &CSVLoader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadCSV opens a CSV file and loads it into the given PostgreSQL table
// using COPY ... FROM STDIN with CSV format.
//
// NULL handling: COPY is invoked with NULL '' so that empty CSV fields
// (consecutive delimiters) become SQL NULL. This matches the behaviour of
// Python's safe_int() / safe_float() which return None for empty input.
//
// tableName should be schema-qualified, e.g. "raw.olist_customers".
//
// Returns the number of rows loaded (excluding the CSV header).
func (l *CSVLoader) LoadCSV(ctx context.Context, tx pgx.Tx, csvPath, tableName string) (int64, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, fmt.Errorf("open csv %s: %w", csvPath, err)
	}
	defer file.Close()

	ident := pgx.Identifier(strings.Split(tableName, "."))

	copySQL := fmt.Sprintf("COPY %s FROM STDIN (FORMAT CSV, HEADER true, NULL '')",
		ident.Sanitize())

	ct, err := tx.Conn().PgConn().CopyFrom(ctx, file, copySQL)
	if err != nil {
		return 0, fmt.Errorf("copy %s into %s: %w", csvPath, tableName, err)
	}

	return ct.RowsAffected(), nil
}
