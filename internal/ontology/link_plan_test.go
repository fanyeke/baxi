package ontology

import (
	"strings"
	"testing"
)

func TestCompileQueryRef_ParameterizedQuery(t *testing.T) {
	// Regression test for SQL injection vulnerability.
	// compileQueryRef must use $1 as a pgx parameterized placeholder
	// rather than string-interpolating the ObjectID into the SQL.

	template := "SELECT * FROM target WHERE id = $1"
	link := &ObjectLinkV2{
		Name: "test_link",
		Target: LinkTarget{
			Key: template,
		},
		TargetType:  "target_type",
		Cardinality: "one_to_many",
	}

	resolver := NewLinkResolver(nil)

	// Test with a normal object ID
	source := ObjectRef{ObjectType: "test", ObjectID: "obj-123"}
	plan, err := resolver.compileQueryRef(source, link, 20, 0)
	if err != nil {
		t.Fatalf("compileQueryRef failed: %v", err)
	}

	// Verify the SQL still contains '$1' as a placeholder (not replaced)
	if !strings.Contains(plan.SQL, "$1") {
		t.Errorf("SQL should contain $1 placeholder but got: %s", plan.SQL)
	}

	// Verify the ObjectID is in Args (parameterized), not in the SQL
	if len(plan.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(plan.Args))
	}
	if plan.Args[0] != "obj-123" {
		t.Errorf("expected arg 'obj-123', got %v", plan.Args[0])
	}
}

func TestCompileQueryRef_SQLInjectionPrevented(t *testing.T) {
	// Regression test: malicious ObjectID values should NOT be executed as SQL.
	template := "SELECT * FROM target WHERE id = $1"
	link := &ObjectLinkV2{
		Name: "test_link",
		Target: LinkTarget{
			Key: template,
		},
		TargetType:  "target_type",
		Cardinality: "one_to_many",
	}

	resolver := NewLinkResolver(nil)

	// SQL injection attempt via ObjectID
	maliciousID := "1'; DROP TABLE target; --"
	source := ObjectRef{ObjectType: "test", ObjectID: maliciousID}

	plan, err := resolver.compileQueryRef(source, link, 20, 0)
	if err != nil {
		t.Fatalf("compileQueryRef failed: %v", err)
	}

	// The SQL must still be the unchanged template with $1 placeholder
	if plan.SQL != template+" LIMIT 20 OFFSET 0" {
		t.Errorf("SQL should be template + LIMIT/OFFSET, got: %s", plan.SQL)
	}

	// The malicious value should be in Args, not in the SQL
	if len(plan.Args) != 1 || plan.Args[0] != maliciousID {
		t.Errorf("malicious ObjectID should be in Args as parameterized value")
	}

	// Verify the SQL does NOT contain the malicious string
	if strings.Contains(plan.SQL, maliciousID) {
		t.Errorf("SQL must not contain the raw ObjectID value (SQL injection)")
	}
}

func TestCompileQueryRef_TemplateWithLimit(t *testing.T) {
	// When the template already contains LIMIT, no additional LIMIT should be appended.
	template := "SELECT * FROM target WHERE status = $1 AND id = $2 LIMIT 10"
	link := &ObjectLinkV2{
		Name: "test_link",
		Target: LinkTarget{
			Key: template,
		},
		TargetType:  "target_type",
		Cardinality: "one_to_many",
	}

	resolver := NewLinkResolver(nil)

	source := ObjectRef{ObjectType: "test", ObjectID: "obj-456"}
	plan, err := resolver.compileQueryRef(source, link, 20, 0)
	if err != nil {
		t.Fatalf("compileQueryRef failed: %v", err)
	}

	// Should NOT add another LIMIT clause since template already has one
	if strings.Count(plan.SQL, "LIMIT") != 1 {
		t.Errorf("expected 1 LIMIT clause, got SQL: %s", plan.SQL)
	}

	// Both $1 and $2 should remain as pgx placeholders
	if !strings.Contains(plan.SQL, "$1") || !strings.Contains(plan.SQL, "$2") {
		t.Errorf("SQL should contain $1 and $2 placeholders, got: %s", plan.SQL)
	}

	// Args should contain only the ObjectID (for $1)
	if len(plan.Args) != 1 || plan.Args[0] != "obj-456" {
		t.Errorf("expected 1 arg 'obj-456', got %v", plan.Args)
	}
}

func TestResolveSort_ValidSort(t *testing.T) {
	link := &ObjectLinkV2{
		Name: "test_link",
		Sort: "order_purchase_timestamp DESC",
	}
	resolver := NewLinkResolver(nil)
	result := resolver.resolveSort(link)
	expected := " ORDER BY order_purchase_timestamp DESC"
	if result != expected {
		t.Errorf("resolveSort = %q, want %q", result, expected)
	}
}

func TestResolveSort_EmptySort(t *testing.T) {
	link := &ObjectLinkV2{
		Name: "test_link",
		Sort: "",
	}
	resolver := NewLinkResolver(nil)
	result := resolver.resolveSort(link)
	if result != "" {
		t.Errorf("resolveSort = %q, want empty string", result)
	}
}

func TestResolveSort_SQLInjectionPrevented(t *testing.T) {
	link := &ObjectLinkV2{
		Name: "test_link",
		Sort: "1; DROP TABLE users",
	}
	resolver := NewLinkResolver(nil)
	result := resolver.resolveSort(link)
	// Must return empty string to prevent SQL injection
	if result != "" {
		t.Errorf("resolveSort with malicious sort = %q, want empty string (SQL injection prevented)", result)
	}
}

func TestResolveSort_MultiColumnWithDirections(t *testing.T) {
	link := &ObjectLinkV2{
		Name: "test_link",
		Sort: "col1 ASC, col2 DESC, col3",
	}
	resolver := NewLinkResolver(nil)
	result := resolver.resolveSort(link)
	expected := " ORDER BY col1 ASC, col2 DESC, col3"
	if result != expected {
		t.Errorf("resolveSort = %q, want %q", result, expected)
	}
}

func TestResolveSort_SubqueryRejected(t *testing.T) {
	link := &ObjectLinkV2{
		Name: "test_link",
		Sort: "(SELECT 1)",
	}
	resolver := NewLinkResolver(nil)
	result := resolver.resolveSort(link)
	if result != "" {
		t.Errorf("resolveSort with subquery sort = %q, want empty string", result)
	}
}
