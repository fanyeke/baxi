package ingest

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──── NewCSVLoader option tests ────────────────────────────────────────────

func TestNewCSVLoader_Default(t *testing.T) {
	l := NewCSVLoader()
	require.NotNil(t, l)
}

func TestNewCSVLoader_WithOptions(t *testing.T) {
	called := false
	l := NewCSVLoader(func(loader *CSVLoader) {
		called = true
	})
	require.NotNil(t, l)
	assert.True(t, called)
	_ = l // use l
}

func TestNewCSVLoader_MultipleOptions(t *testing.T) {
	order := []int{}
	l := NewCSVLoader(
		func(loader *CSVLoader) { order = append(order, 1) },
		func(loader *CSVLoader) { order = append(order, 2) },
	)
	require.NotNil(t, l)
	assert.Equal(t, []int{1, 2}, order)
	_ = l
}

// ──── LoadCSV error paths ──────────────────────────────────────────────────

func TestLoadCSV_FileNotFound(t *testing.T) {
	l := NewCSVLoader()
	_, err := l.LoadCSV(context.Background(), nil, "/nonexistent/path.csv", "raw.test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open csv")
}


// ──── TableMapping tests ───────────────────────────────────────────────────

func TestAllTableMappings_Count(t *testing.T) {
	m := AllTableMappings()
	assert.Len(t, m, 11)
}

func TestAllTableMappings_RequiredFields(t *testing.T) {
	for _, m := range AllTableMappings() {
		assert.NotEmpty(t, m.CSVFile)
		assert.NotEmpty(t, m.TableName)
	}
}

func TestAllTableMappings_NoNilInterface(t *testing.T) {
	// Verify pgx.Identifier behavior with AllTableMappings table names
	for _, m := range AllTableMappings() {
		ident := pgx.Identifier{"raw", m.CSVFile}
		sanitized := ident.Sanitize()
		assert.NotEmpty(t, sanitized)
	}
}
