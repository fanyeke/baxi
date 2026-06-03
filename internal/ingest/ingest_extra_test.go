package ingest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllTableMappings_RequiredFlags(t *testing.T) {
	mappings := AllTableMappings()
	required := 0
	optional := 0
	for _, m := range mappings {
		if m.Required {
			required++
		} else {
			optional++
		}
	}
	assert.Equal(t, 9, required, "expected 9 required mappings")
	assert.Equal(t, 2, optional, "expected 2 optional mappings")
}

func TestAllTableMappings_CSVFileNames(t *testing.T) {
	mappings := AllTableMappings()
	for _, m := range mappings {
		assert.Contains(t, m.CSVFile, ".csv", "CSV file should end with .csv")
		assert.NotEmpty(t, m.CSVFile)
	}
}

func TestAllTableMappings_TableNamesHaveSchema(t *testing.T) {
	mappings := AllTableMappings()
	for _, m := range mappings {
		assert.Contains(t, m.TableName, ".", "table name should be schema-qualified")
		assert.True(t, len(m.TableName) > 4 && m.TableName[:4] == "raw.")
	}
}

func TestCSVLoader_Option(t *testing.T) {
	called := false
	l := NewCSVLoader(func(c *CSVLoader) {
		called = true
	})
	assert.NotNil(t, l)
	assert.True(t, called)
}

func TestCSVLoader_NoOptions(t *testing.T) {
	l := NewCSVLoader()
	assert.NotNil(t, l)
}

func TestCSVFileMapping_Fields(t *testing.T) {
	m := CSVFileMapping{
		CSVFile:   "test.csv",
		TableName: "raw.test",
		Required:  true,
	}
	assert.Equal(t, "test.csv", m.CSVFile)
	assert.Equal(t, "raw.test", m.TableName)
	assert.True(t, m.Required)
}

func TestCSVLoader_LoadCSV_NilTx(t *testing.T) {
	l := NewCSVLoader()
	_, err := l.LoadCSV(context.Background(), nil, "/nonexistent.csv", "raw.test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open csv")
}

func TestCSVLoader_LoadCSV_NonexistentPath(t *testing.T) {
	l := NewCSVLoader()
	_, err := l.LoadCSV(context.Background(), nil, "/absolutely/nonexistent/path/file.csv", "raw.test")
	assert.Error(t, err)
}
