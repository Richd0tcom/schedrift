package models

import (
	"fmt"
	"strings"
	"time"
)

type ConstraintType string

const (
	PRIMARY_KEY ConstraintType = "PRIMARY KEY"
	FOREIGN_KEY ConstraintType = "FOREIGN KEY"
	CHECK       ConstraintType = "CHECK"
	UNIQUE      ConstraintType = "UNIQUE"
	EXCLUDE     ConstraintType = "EXCLUDE"
)

type Schema struct {
	Name      string
	Tables    []*Table
	Views     []*View
	Triggers  []*Trigger
	Indexes   []*Index
	Functions []*Function
	Sequences []*Sequence
}

type Table struct {
	Name        string
	Schema      string
	Columns     []*Column
	Constraints []*Constraint
	Comment     string
}

type Column struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue string
	Comment      string
}

type Constraint struct {
	Name       string
	Type       ConstraintType // PRIMARY KEY, FOREIGN KEY, CHECK, UNIQUE
	Columns    []string
	References string // Used for FOREIGN KEY: e.g., "other_table(col1, col2)"
	CheckExpr  string // Used for CHECK: e.g., "price > 0"
	RawSQL     string // Optional: if set, returned as-is
}

type View struct {
	Schema     string
	Name       string
	Definition string
	Comment    string
}

type Trigger struct {
	Name      string
	Schema    string
	Table     string
	Events    []string
	Timing    string // BEFORE, AFTER, INSTEAD OF
	Statement string
}

type Index struct {
	Name       string
	Schema     string
	Table      string
	Columns    []string
	IsUnique   bool
	Method     string // btree, hash, etc.
	Definition string
}

type Function struct {
	Name       string
	Schema     string
	Arguments  string //?
	ReturnType string
	Definition string
	Language   string
}

type Sequence struct {
	Name      string
	Schema    string
	Start     int64
	Increment int64
	Min       int64
	Max       int64
	Cache     int64
}

// DatabaseSchema represents the complete schema of a database
type DatabaseSchema struct {
	Name    string
	Schemas []*Schema
}

func (ds *DatabaseSchema) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- Schema Drift Detector - Database: %s\n", ds.Name))
	sb.WriteString("-- Generated at: " + fmt.Sprintf("%s\n\n", NowFunc()))

	for _, schema := range ds.Schemas {
		sb.WriteString(schema.ToSQL())
		sb.WriteString("\n")
	}

	return sb.String()
}

func (sc *Schema) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("-- Schema: %s\n", sc.Name))
	if sc.Name != "public" {
		sb.WriteString(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", sc.Name))
	}

	if len(sc.Sequences) > 0 {
		sb.WriteString(fmt.Sprintf("-- Sequences: %s\n", sc.Name))
		for _, seq := range sc.Sequences {
			sb.WriteString(seq.ToSQL())
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(sc.Tables) > 0 {
		sb.WriteString(fmt.Sprintf("-- Tables: %s\n", sc.Name))
		for _, table := range sc.Tables {
			sb.WriteString(table.ToSQL())
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(sc.Views) > 0 {
		sb.WriteString(fmt.Sprintf("-- Views: %s\n", sc.Name))
		for _, view := range sc.Views {
			sb.WriteString(view.ToSQL())
			sb.WriteString("\n")
		}
	}

	if len(sc.Triggers) > 0 {
		sb.WriteString(fmt.Sprintf("-- Triggers: %s\n", sc.Name))
		for _, trigger := range sc.Triggers {
			sb.WriteString(trigger.ToSQL())
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(sc.Indexes) > 0 {
		sb.WriteString(fmt.Sprintf("-- Indexes: %s\n", sc.Name))
		for _, index := range sc.Indexes {
			sb.WriteString(index.ToSQL())
			sb.WriteString(";\n")
		}

		sb.WriteString("\n")
	}
	if len(sc.Functions) > 0 {
		sb.WriteString(fmt.Sprintf("-- Functions: %s\n", sc.Name))
		for _, function := range sc.Functions {
			sb.WriteString(function.ToSQL())
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	if len(sc.Sequences) > 0 {
		sb.WriteString(fmt.Sprintf("-- Sequences: %s\n", sc.Name))
		for _, seq := range sc.Sequences {
			sb.WriteString(seq.ToSQL())
			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	// if sc.Name != "public" {
	// 	sb.WriteString(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;\n", sc.Name))
	// }

	return sb.String()
}

func (col *Column) ToSQL() string {
	var parts []string

	parts = append(parts, col.Name)
	parts = append(parts, col.DataType)

	if !col.IsNullable {
		parts = append(parts, "NOT NULL")
	}

	parts = append(parts, col.DefaultValue)
	parts = append(parts, col.Comment)

	return strings.Join(parts, " ")
}

func (t *Table) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE %s.%s (\n", t.Schema, t.Name))

	for i, col := range t.Columns {
		sb.WriteString("  " + col.ToSQL())
		if i < len(t.Columns)-1 || len(t.Constraints) > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	for i, constraint := range t.Constraints {
		sb.WriteString(fmt.Sprintf("  %s", constraint.ToSQL()))
		if i > len(t.Columns)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(");\n")

	if t.Comment != "" {
		sb.WriteString(fmt.Sprintf("COMMENT ON TABLE %s.%s IS '%s';\n", t.Schema, t.Name, t.Comment))
	}

	for _, c := range t.Columns {
		if c.Comment != "" {
			sb.WriteString(fmt.Sprintf("COMMENT ON COLUMN %s.%s.%s IS '%s';\n", t.Schema, t.Name, c.Name, c.Comment))
		}
	}

	return sb.String()
}

func (con *Constraint) ToSQL() string {
	if con.RawSQL != "" {
		return con.RawSQL
	}

	var sb strings.Builder

	if con.Name != "" {
		sb.WriteString(fmt.Sprintf("CONSTRAINT %s", con.Name))
	}

	switch con.Type {
	case PRIMARY_KEY:
		sb.WriteString(fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(con.Columns, ", "))) //strings.Join(con.Columns, ", "))) accounts for composite keys
	case FOREIGN_KEY:
		sb.WriteString(fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s", strings.Join(con.Columns, ", "), con.References))
	case CHECK:
		sb.WriteString(fmt.Sprintf("CHECK (%s)", con.CheckExpr))
	case UNIQUE:
		sb.WriteString(fmt.Sprintf("UNIQUE (%s)", strings.Join(con.Columns, ", ")))
	default:
		sb.WriteString("-- Unknown constraint type")

	}

	return sb.String()
}

func (i *Index) ToSQL() string {
	if i.Definition != "" {
		return i.Definition + ";\n"

	}

	var sb strings.Builder

	sb.WriteString("CREATE")

	if i.IsUnique {
		sb.WriteString("UNIQUE")

	}

	sb.WriteString(fmt.Sprintf("INDEX %s ON %s.%s USING %s (%s)",
		i.Name, i.Schema, i.Table, i.Method, strings.Join(i.Columns, ", ")))

	return sb.String()
}

func (tr *Trigger) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TRIGGER %s\n%s %s ON %s.%s",
		tr.Name, tr.Timing, strings.Join(tr.Events, " OR "), tr.Schema, tr.Table))

	sb.WriteString(fmt.Sprintf("EXECUTE PROCEDURE %s;\n", tr.Statement))

	return sb.String()
}

func (fn *Function) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE OR REPLACE FUNCTION %s.%s(%s)", fn.Schema, fn.Name, fn.Arguments))
	sb.WriteString(fmt.Sprintf("RETURNS %s AS $$\n", fn.ReturnType))
	sb.WriteString(fn.Definition)
	sb.WriteString(fmt.Sprintf("\n$$ LANGUAGE %s;\n", fn.Language))

	return sb.String()
}

func (seq *Sequence) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE SEQUENCE %s.%s\n", seq.Schema, seq.Name))
	sb.WriteString(fmt.Sprintf("    START WITH %d\n", seq.Start))
	sb.WriteString(fmt.Sprintf("    INCREMENT BY %d\n", seq.Increment))
	sb.WriteString(fmt.Sprintf("    MINVALUE %d\n", seq.Min))
	sb.WriteString(fmt.Sprintf("    MAXVALUE %d\n", seq.Max))
	sb.WriteString(fmt.Sprintf("    CACHE %d;\n", seq.Cache))

	return sb.String()
}

func (v *View) ToSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE OR REPLACE VIEW %s.%s AS\n%s;\n",
		v.Schema, v.Name, v.Definition))

	if v.Comment != "" {
		sb.WriteString(fmt.Sprintf("COMMENT ON VIEW %s.%s IS '%s';\n",
			v.Schema, v.Name, escapeString(v.Comment)))
	}

	return sb.String()
}

var NowFunc = func() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
