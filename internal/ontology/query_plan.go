package ontology

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type QueryCompiler struct {
	objects  map[string]*ObjectTypeV2
	maxLimit int
}

func NewQueryCompiler(objects map[string]*ObjectTypeV2, maxLimit int) *QueryCompiler {
	if maxLimit <= 0 {
		maxLimit = 10000
	}
	return &QueryCompiler{objects: objects, maxLimit: maxLimit}
}

type ObjectFilters struct {
	Filters map[string]interface{}
	Limit   int
	Offset  int
	Sort    string
	Order   string
}

func (qc *QueryCompiler) GetObjectType(name string) (*ObjectTypeV2, bool) {
	ot, ok := qc.objects[name]
	return ot, ok
}

func (qc *QueryCompiler) ObjectTypes() map[string]*ObjectTypeV2 {
	return qc.objects
}

func (qc *QueryCompiler) MaxLimit() int {
	return qc.maxLimit
}

func (qc *QueryCompiler) CompileObjectQuery(objectType, objectID string) (CompiledQuery, error) {
	ot, ok := qc.objects[objectType]
	if !ok {
		return CompiledQuery{}, fmt.Errorf("unknown object type: %s", objectType)
	}

	cols := make([]string, 0, len(ot.Properties))
	colNames := make([]string, 0, len(ot.Properties))
	for name, prop := range ot.Properties {
		if prop.Expression != "" {
			// Skip cross-table expressions (contain ".")
			// because the source table is not joined to the referenced table.
			if strings.Contains(prop.Expression, ".") {
				continue
			}
			cols = append(cols, prop.Expression+" AS "+name)
		} else if prop.SourceField != "" {
			cols = append(cols, prop.SourceField)
		} else {
			cols = append(cols, name)
		}
		colNames = append(colNames, name)
	}

	pk := ot.Source.PrimaryKey
	if pk == "" {
		for name, prop := range ot.Properties {
			if prop.IsPK {
				pk = name
				break
			}
		}
	}

	tableName := ot.Source.Schema + "." + ot.Source.Table
	namedArgs := pgx.NamedArgs{}
	namedArgs["pk"] = objectID

	sql := fmt.Sprintf("SELECT %s FROM %s WHERE %s = @pk LIMIT 1",
		strings.Join(cols, ", "), tableName, pk)

	return CompiledQuery{
		SQL:        sql,
		Args:       namedArgs,
		Columns:    colNames,
		ObjectType: objectType,
		PrimaryKey: pk,
		Schema:     ot.Source.Schema,
		Table:      ot.Source.Table,
	}, nil
}
