package postgres

import (
	"fmt"

	"github.com/Richd0tcom/schedrift/internal/models"
)

type PGExtractor struct {
	conn *PGConnection
}

func NewPGExtractor(conn *PGConnection) *PGExtractor {
	return &PGExtractor{conn: conn}
}

func (e *PGExtractor) Extract(incluedSchemas,  excludedSchemas []string) (*models.DatabaseSchema, error) {

	var dbName string
	err:= e.conn.QueryRow(`SELECT current_database()`).Scan(&dbName)

	if err != nil {
		return nil, fmt.Errorf("error getting database name: %w", err)
	}

	dbSchema:= &models.DatabaseSchema{
		Name: dbName,
		Schemas: []*models.Schema{},
	}

	// Build schema filter
	// schemaFilter := e.buildSchemaFilter(includedSchemas, excludedSchemas)
	schemaFilter:= ""

	rows, err:= e.conn.Query(`SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast') 
		AND ` + schemaFilter + `
		ORDER BY schema_name
		`)

	if err != nil {
		return nil, fmt.Errorf("error querying schemas: %w", err)
	}

	defer rows.Close()

	for rows.Next() {

		var schemaName string
		if err= rows.Scan(&schemaName); err != nil {
			return nil, fmt.Errorf("error scanning schema: %w", err)
		}
		sch, err:= e.extractSchemas(schemaName)

		if err != nil {
			return nil, fmt.Errorf("error extracting schema %w", err)
		}

		dbSchema.Schemas = append(dbSchema.Schemas, sch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schemas: %w", err)
	}

	return dbSchema, nil
}

func (e *PGExtractor) extractSchemas(schemaName string) (*models.Schema, error) {
	schema:= &models.Schema{
		Name: schemaName,
		Tables: []*models.Table{},
		Views: []*models.View{},
		Triggers: []*models.Trigger{},
		Indexes: []*models.Index{},
		Functions: []*models.Function{},
		Sequences: []*models.Sequence{},
	}

	var err error

	if err= e.extractTables(schema); err != nil {
		return nil, fmt.Errorf("error extracting tables %w", err)
	}

	return schema, nil
}

func (e *PGExtractor) extractTables(schema *models.Schema) error {
	rows, err := e.conn.Query(`
		SELECT 
			c.table_name, 
			obj_description(c.oid, 'pg_class') as table_comment
		FROM 
			pg_catalog.pg_class c
		JOIN 
			pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE 
			c.relkind = 'r' 
			AND n.nspname = $1
		ORDER BY 
			c.relname
	`, schema.Name)

	if err != nil {
		return fmt.Errorf("error querying tables: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		table:= &models.Table{
			Schema: schema.Name,
			Columns: []*models.Column{},
			Constraints: []*models.Constraint{},
		}
		err= rows.Scan(&table.Name, &table.Comment); if err != nil {
			return err
		}

		err = e.extractColumns(table)
		if err != nil {
			return fmt.Errorf("error extracting columns %w",  err)
		}

		err = e.extractConstraints(table)
		if err != nil {
			return fmt.Errorf("error extracting constraints %w",  err)
		}

		schema.Tables = append(schema.Tables, table)
	}

	if rows.Err() != nil {
		return fmt.Errorf("error iterating tables: %w", err)
	}

	return nil
}

func (e *PGExtractor) extractColumns(table *models.Table) error {

	rows, err:= e.conn.Query(`SELECT 
			c.column_name, 
			c.data_type, 
			CASE WHEN c.is_nullable = 'YES' THEN true ELSE false END, 
			c.column_default,
			pgd.description as column_comment
		FROM 
			information_schema.columns c
		LEFT JOIN 
			pg_catalog.pg_description pgd ON 
				pgd.objoid = ('"' || c.table_schema || '"."' || c.table_name || '"')::regclass AND 
				pgd.objsubid = c.ordinal_position
		WHERE 
			c.table_schema = $1 AND c.table_name = $2
		ORDER BY 
			c.ordinal_position
	`, table.Schema, table.Name)

	if err != nil {
		return fmt.Errorf("error querying columns %w", err)
	}


	return nil


}

func (e *PGExtractor) extractConstraints(schema *models.Table) error

func (e *PGExtractor) extractViews(schema *models.Schema) error

func (e *PGExtractor) extractIndexes(schema *models.Schema) error

func (e *PGExtractor) extractTriggers(schema *models.Schema) error

func (e *PGExtractor) extractSequences(schema *models.Schema) error


