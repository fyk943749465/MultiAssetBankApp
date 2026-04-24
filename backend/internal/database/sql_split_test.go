package database

import "testing"

func TestSplitSQLStatements_commentOnly(t *testing.T) {
	out := splitSQLStatements("-- a\n-- b\n")
	if len(out) != 0 {
		t.Fatalf("expected 0 stmts, got %d: %v", len(out), out)
	}
}

func TestSplitSQLStatements_simple(t *testing.T) {
	sql := "SELECT 1;\nSELECT 2;"
	out := splitSQLStatements(sql)
	if len(out) != 2 {
		t.Fatalf("got %d: %v", len(out), out)
	}
}

func TestSplitSQLStatements_stringWithSemicolon(t *testing.T) {
	sql := `SELECT ';';`
	out := splitSQLStatements(sql)
	if len(out) != 1 {
		t.Fatalf("got %d: %v", len(out), out)
	}
}
