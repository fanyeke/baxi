package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Unit tests (no DB required) ----

func TestAllTableMappings_HasExpectedCount(t *testing.T) {
	mappings := AllTableMappings()
	if len(mappings) != 11 {
		t.Fatalf("expected 11 table mappings, got %d", len(mappings))
	}
}

func TestAllTableMappings_CSVFilesExist(t *testing.T) {
	mappings := AllTableMappings()
	for _, m := range mappings {
		path := filepath.Join("..", "..", "data", "raw", m.CSVFile)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing CSV file: %s (%v)", m.CSVFile, err)
		}
	}
}

func TestAllTableMappings_NoDuplicates(t *testing.T) {
	mappings := AllTableMappings()
	seen := make(map[string]bool)
	for _, m := range mappings {
		if seen[m.CSVFile] {
			t.Errorf("duplicate CSV file mapping: %s", m.CSVFile)
		}
		seen[m.CSVFile] = true
	}
}

func TestAllTableMappings_SchemaPrefixed(t *testing.T) {
	mappings := AllTableMappings()
	for _, m := range mappings {
		if !strings.Contains(m.TableName, ".") {
			t.Errorf("table name %q is not schema-qualified", m.TableName)
		}
	}
}

func TestNewCSVLoader_Defaults(t *testing.T) {
	l := NewCSVLoader()
	if l == nil {
		t.Fatal("expected non-nil loader")
	}
}

func TestLoadCSV_Error_fileNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	l := NewCSVLoader()

	_, err := l.LoadCSV(context.Background(), nil, "/nonexistent/file.csv", "raw.olist_customers")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

// ---- Integration tests (require reachable PostgreSQL) ----

func mustPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("SKIP_INTEGRATION is set")
	}
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://localhost:5432/baxi_test?sslmode=disable&connect_timeout=3"
	} else if !strings.Contains(dsn, "connect_timeout") {
		dsn += "&connect_timeout=3"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("cannot create pool (dsn=%s): %v", maskDSN(dsn), err)
	}
	t.Cleanup(pool.Close)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		t.Skipf("test db not reachable: %v", err)
	}
	return pool
}

func maskDSN(dsn string) string {
	if i := strings.Index(dsn, ":"); i >= 0 {
		j := strings.Index(dsn[i+1:], "@")
		if j >= 0 {
			return dsn[:i+1] + dsn[i+1+j+1:]
		}
	}
	return dsn
}

func setupTestTable(ctx context.Context, t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	schema := fmt.Sprintf("tmp_ingest_%d", os.Getpid())
	table := fmt.Sprintf("%s.test_csv", schema)

	exec := func(sql string) {
		_, err := pool.Exec(ctx, sql)
		if err != nil {
			t.Fatalf("setup sql: %s\n  error: %v", sql, err)
		}
	}

	exec(fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, pgx.Identifier{schema}.Sanitize()))
	exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id    BIGSERIAL PRIMARY KEY,
		a     TEXT,
		b     BIGINT,
		c     NUMERIC(18,2)
	)`, pgx.Identifier{schema, "test_csv"}.Sanitize()))

	t.Cleanup(func() {
		pool.Exec(context.Background(),
			fmt.Sprintf("DROP TABLE IF EXISTS %s", pgx.Identifier{schema, "test_csv"}.Sanitize()))
		pool.Exec(context.Background(),
			fmt.Sprintf("DROP SCHEMA IF EXISTS %s", pgx.Identifier{schema}.Sanitize()))
	})

	return table
}

func writeTempCSV(t *testing.T, lines ...string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	header := "a,b,c\n"
	data := header + strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

func TestLoadCSV_HappyPath(t *testing.T) {
	ctx := context.Background()
	pool := mustPool(ctx, t)
	loader := NewCSVLoader()

	table := setupTestTable(ctx, t, pool)
	csvPath := writeTempCSV(t,
		"hello,42,3.14",
		"world,100,200.50",
	)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	n, err := loader.LoadCSV(ctx, tx, csvPath, table)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 rows loaded, got %d", n)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}
}

func TestLoadCSV_NULLHandling(t *testing.T) {
	ctx := context.Background()
	pool := mustPool(ctx, t)
	loader := NewCSVLoader()

	table := setupTestTable(ctx, t, pool)
	// Row 1: ,,  → empty CSV fields → PostgreSQL NULL (matching safe_int/safe_float)
	// Row 2: "",, → quoted empty string '' is not NULL per CSV spec; '' becomes ''
	csvPath := writeTempCSV(t, ",,", `"",,`)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	n, err := loader.LoadCSV(ctx, tx, csvPath, table)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 rows loaded, got %d", n)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	row1 := pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT id, a, b, c FROM %s ORDER BY id LIMIT 1", tableIdent(table)))
	var id int64
	var a, b, c interface{}
	if err := row1.Scan(&id, &a, &b, &c); err != nil {
		t.Fatalf("scan row 1: %v", err)
	}
	if a != nil {
		t.Errorf("expected NULL for empty text field a, got %#v", a)
	}
	if b != nil {
		t.Errorf("expected NULL for empty bigint field b, got %#v", b)
	}
	if c != nil {
		t.Errorf("expected NULL for empty numeric field c, got %#v", c)
	}

	row2 := pool.QueryRow(ctx, fmt.Sprintf(
		"SELECT a FROM %s ORDER BY id OFFSET 1 LIMIT 1", tableIdent(table)))
	var a2 string
	if err := row2.Scan(&a2); err != nil {
		t.Fatalf("scan row 2: %v", err)
	}
	if a2 != "" {
		t.Errorf("expected empty string for quoted empty field, got %q", a2)
	}
}

func tableIdent(name string) string {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return pgx.Identifier{parts[0], parts[1]}.Sanitize()
	}
	return pgx.Identifier{name}.Sanitize()
}

func TestLoadCSV_Error_wrongTable(t *testing.T) {
	ctx := context.Background()
	pool := mustPool(ctx, t)
	loader := NewCSVLoader()

	csvPath := writeTempCSV(t, "x,1,2.0")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = loader.LoadCSV(ctx, tx, csvPath, "raw.nonexistent_table_xyz")
	if err == nil {
		t.Fatal("expected error for non-existent table")
	}
}

func TestLoadCSV_LargeFile(t *testing.T) {
	ctx := context.Background()
	pool := mustPool(ctx, t)
	loader := NewCSVLoader()

	table := setupTestTable(ctx, t, pool)

	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		lines[i] = fmt.Sprintf("val_%d,%d,%.2f", i, i*10, float64(i)*1.5)
	}
	csvPath := writeTempCSV(t, lines...)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	defer tx.Rollback(ctx)

	n, err := loader.LoadCSV(ctx, tx, csvPath, table)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}
	if n != 1000 {
		t.Fatalf("expected 1000 rows, got %d", n)
	}
}
