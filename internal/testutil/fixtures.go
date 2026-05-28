package testutil

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

// LoadFixtureCSV reads a CSV file and inserts all rows into tableName using
// the provided transaction. The first row of the CSV must be column headers
// matching the target table's column names.
//
// The CSV values are inserted as text; PostgreSQL handles type coercion.
// Returns an error if the CSV has fewer than 2 rows (header + at least one
// data row) or if any INSERT fails.
func LoadFixtureCSV(tx pgx.Tx, tableName string, csvPath string) error {
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open fixture file %s: %w", csvPath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = -1 // allow variable columns
	reader.ReuseRecord = true

	rows, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("read csv %s: %w", csvPath, err)
	}

	if len(rows) < 2 {
		return fmt.Errorf("fixture %s: need header + 1 row, got %d rows", csvPath, len(rows))
	}

	headers := rows[0]
	colNames := make([]string, len(headers))
	placeholders := make([]string, len(headers))
	for i, h := range headers {
		colNames[i] = h
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	stmt := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(colNames, ", "),
		strings.Join(placeholders, ", "),
	)

	for _, row := range rows[1:] {
		args := make([]any, len(row))
		for i, v := range row {
			args[i] = v
		}
		if _, err := tx.Exec(context.Background(), stmt, args...); err != nil {
			return fmt.Errorf("insert into %s: %w (row values: %v)", tableName, err, row)
		}
	}

	return nil
}
