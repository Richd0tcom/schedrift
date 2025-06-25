package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Richd0tcom/schedrift/internal/models"
)

type SQLParser struct {
	createTableRegex *regexp.Regexp
	columnRegex      *regexp.Regexp
	constraintRegex  *regexp.Regexp
	indexRegex       *regexp.Regexp
	sequenceRegex    *regexp.Regexp
	functionRegex    *regexp.Regexp
	alterTableRegex  *regexp.Regexp
	primaryKeyRegex  *regexp.Regexp
	foreignKeyRegex  *regexp.Regexp
	uniqueRegex      *regexp.Regexp
}

func NewSQLParser() *SQLParser {
	return &SQLParser{
		createTableRegex: regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)\s*\(`),
		columnRegex:      regexp.MustCompile(`(?i)^\s*([^\s,]+)\s+([^\s,]+(?:\([^)]+\))?)\s*(.*?)(?:,\s*$|$)`),
		constraintRegex:  regexp.MustCompile(`(?i)CONSTRAINT\s+([^\s]+)\s+(.*)`),
		indexRegex:       regexp.MustCompile(`(?i)CREATE\s+(?:(UNIQUE)\s+)?INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)\s+ON\s+([^\s(]+)\s*(?:USING\s+([^\s(]+))?\s*\(([^)]+)\)`),
		sequenceRegex:    regexp.MustCompile(`(?i)CREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s]+)(?:\s+(.*))?`),
		functionRegex:    regexp.MustCompile(`(?i)CREATE\s+(?:OR\s+REPLACE\s+)?FUNCTION\s+([^\s(]+)\s*\([^)]*\)\s+RETURNS\s+([^\s]+)(?:\s+LANGUAGE\s+([^\s]+))?`),
		alterTableRegex:  regexp.MustCompile(`(?i)ALTER\s+TABLE\s+([^\s]+)\s+(.*)`),
		primaryKeyRegex:  regexp.MustCompile(`(?i)PRIMARY\s+KEY\s*\(([^)]+)\)`),
		foreignKeyRegex:  regexp.MustCompile(`(?i)FOREIGN\s+KEY\s*\(([^)]+)\)\s+REFERENCES\s+([^\s(]+)(?:\s*\(([^)]+)\))?`),
		uniqueRegex:      regexp.MustCompile(`(?i)UNIQUE\s*\(([^)]+)\)`),
	}
}

func (p *SQLParser) removeComments(content string) string {
	//romove singleline commet

	lines := strings.Split(content, "\n")
	cleanlines := make([]string, 0)

	for _, line := range lines {
		var cleanLine strings.Builder
		var inString bool
		var strchar rune

		runes := []rune(line)
		for i, r := range runes {
			if !inString {
				if r == '\'' || r == '"' {
					inString = true
					strchar = r
				} else if r == '-' && i+1 < len(runes) && runes[i+1] == '-' {
					break
				}
			} else if r == strchar {
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}

			cleanLine.WriteRune(r)
		}
		cleanlines = append(cleanlines, cleanLine.String())
	}
	content = strings.Join(cleanlines, "\n")


	//TODO: should we parse multiline comments first and why/why not? 
	multiLineComment := regexp.MustCompile(`/\*.*?\*/`)
	content = multiLineComment.ReplaceAllString(content, "")

	return content
}

func (p *SQLParser) splitStatements(content string) []string {

	pureContent:= p.removeComments(content)

	var statements []string
	var current strings.Builder
	var inString bool
	var inFunction bool
	var stringChar rune
	var parenDepth int


	runes:= []rune(pureContent)

	for i, r := range runes {
		if !inString {
			switch r {
			case '\'' , '"':
				inString = true
				stringChar = r

			case '(':
				parenDepth++
				
			case ')':
				parenDepth--
			case ';':
				if !inFunction || parenDepth == 0 {
					statements = append(statements, current.String())
					current.Reset()
					continue
				}
			}

			if strings.HasPrefix(strings.ToUpper(string(runes[i:])), "CREATE FUNCTION") || 
			strings.HasPrefix(strings.ToUpper(string(runes[i:])), "CREATE OR REPLACE FUNCTION") {
				inFunction = true
			} else if inFunction && r==';' && parenDepth == 0 {
				inFunction = false
			}

		} else {
			if r == stringChar {
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}
			
		}

		current.WriteRune(r)
	} 

	if current.Len() > 0 {
		statements = append(statements, current.String())
	}

	return statements
} 

func (p *SQLParser) parseStatements(schema *models.Schema, stmnt string) error {
	stmnt = strings.TrimSpace(stmnt)
	stmtUpper := strings.ToUpper(stmnt)

	switch {
	case strings.HasPrefix(stmtUpper, "CREATE TABLE"):
		return p.parseCreateTable(schema, stmnt)
	case strings.HasPrefix(stmtUpper, "CREATE INDEX") || strings.HasPrefix(stmtUpper, "CREATE UNIQUE INDEX"):
		return p.parseCreateIndex(schema, stmnt)
	case strings.HasPrefix(stmtUpper, "CREATE SEQUENCE"):
		return p.parseCreateSequence(schema, stmnt)
	case strings.HasPrefix(stmtUpper, "CREATE FUNCTION") || strings.HasPrefix(stmtUpper, "CREATE OR REPLACE FUNCTION"):
		return p.parseCreateFunction(schema, stmnt)
	case strings.HasPrefix(stmtUpper, "ALTER TABLE"):
		return p.parseAlterTable(schema, stmnt)
	default:
		// Ignore other statements
		return nil
	}


}

func (p *SQLParser) parseCreateTable(schema *models.Schema, stmnt string) error {
	matches:= p.createTableRegex.FindStringSubmatch(stmnt)

	if len(matches) < 2 { 
		//this (2) is not a magic number. 
		//Go's Submatch returns a slice with at least 1 value when there is a match with index 0 being the entire string	`1q	`
		//a valid match will contain at leat 2 values
		return fmt.Errorf("invalid CREATE TABLE statement")
	}

	tableName:= p.cleanIdentifier(matches[1])

	table := &models.Table{
		Name: tableName,
		Schema: schema.Name,
		Columns: make([]*models.Column, 0),
		Constraints: make([]*models.Constraint, 0),
		
		//TODO: add indexes
	}

	// Extract table definition (content between parentheses)
	start := strings.Index(stmnt, "(")
	end := strings.LastIndex(stmnt, ")")
	if start == -1 || end == -1 || start >= end {
		return fmt.Errorf("invalid table definition")
	}

	definition:= stmnt[start+1: end]

		// Parse columns and constraints
		if err := p.parseTableDefinition(table, definition); err != nil {
			return fmt.Errorf("failed to parse table definition: %w", err)
		}

	schema.Tables = append(schema.Tables, table)
	return nil
}

func (p *SQLParser) parseTableDefinition(table *models.Table, definition string) error {

	parts:=  p.splitTableParts(definition)

	for _, part:= range parts {

		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if err:= p.parseTablePart(part,table); err!= nil {
			fmt.Printf("Warning: failed to parse table part '%s': %v\n", part, err) //should probably return this erro
		}
	}
	return nil
}

func (p *SQLParser) splitTableParts(definition string) []string {
	var parts []string
	var current strings.Builder
	var parenDepth int
	var inString bool
	var stringChar rune

	for i, r := range definition {
		if !inString {
			switch r {
			case '\'', '"':
				inString = true
				stringChar = r
			case '(':
				parenDepth++
			case ')':
				parenDepth--
			case ',':
				if parenDepth == 0 {
					parts = append(parts, current.String())
					current.Reset()
					continue
				}
			}
		} else if r == stringChar {
			// Check if it's escaped
			if i > 0 && definition[i-1] != '\\' {
				inString = false
			}
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// parseTablePart parses a single part of a table definition (column or constraint)
func (p *SQLParser) parseTablePart(part string, table *models.Table) error {
	part = strings.TrimSpace(part)
	partUpper := strings.ToUpper(part)


	if strings.HasPrefix(partUpper, "CONSTRAINT") {
		return p.parseTableConstraint(table, part)
	}


	if strings.Contains(partUpper, "PRIMARY KEY") {
		return p.parseInlinePrimaryKey(table, part)
	}

	if strings.Contains(partUpper, "FOREIGN KEY") {
		return p.parseInlineForeignKey(table, part)
	}

	if strings.Contains(partUpper, "UNIQUE") && !strings.Contains(partUpper, "NOT") {
		return p.parseInlineUnique(table, part)
	}


	return p.parseColumnDefinition(table, part)
}

func (p *SQLParser) parseColumnDefinition(table *models.Table, definition string) error {
	parts:= strings.Fields(definition)

	if len(parts) < 2 {
		return fmt.Errorf("invalid column definition")
	}
	columnName:= p.cleanIdentifier(parts[0])
	columnType:= parts[1]

	column:= &models.Column{
		Name: columnName,
		DataType: columnType,
		IsNullable: true,
		DefaultValue: "",
	}
	definitionUpper := strings.ToUpper(definition)

	if strings.Contains(definitionUpper, "NOT NULL") {
		column.IsNullable = false
	}

	if strings.Contains(definitionUpper, "DEFAULT") {
		defaultRegex := regexp.MustCompile(`(?i)DEFAULT\s+([^,\s]+(?:\([^)]*\))?)`)
		matches := defaultRegex.FindStringSubmatch(definition)
		if len(matches) > 1 {
			column.DefaultValue = strings.Trim(matches[1], "'\"")
		}
	}

	table.Columns = append(table.Columns, column)
	return nil
}

func (p *SQLParser) parseTableConstraint(table *models.Table, definition string) error {
	matches := p.constraintRegex.FindStringSubmatch(definition)
	if len(matches) < 3 {
		return fmt.Errorf("invalid constraint definition")
	}

	constraintName := p.cleanIdentifier(matches[1])
	constraintDef := strings.TrimSpace(matches[2])
	constraintDefUpper := strings.ToUpper(constraintDef)

	constraint := &models.Constraint{
		Name:       constraintName,
		RawSQL: constraintDef,
	}

	// Determine constraint type and extract columns
	switch {
	case strings.HasPrefix(constraintDefUpper, "PRIMARY KEY"):
		constraint.Type = "PRIMARY KEY"
		if matches := p.primaryKeyRegex.FindStringSubmatch(constraintDef); len(matches) > 1 {
			constraint.Columns = p.parseColumnList(matches[1])
		}
	case strings.HasPrefix(constraintDefUpper, "FOREIGN KEY"):
		constraint.Type = "FOREIGN KEY"
		if matches := p.foreignKeyRegex.FindStringSubmatch(constraintDef); len(matches) > 1 {
			constraint.Columns = p.parseColumnList(matches[1])
			if len(matches) > 2 {
				constraint.References = p.cleanIdentifier(matches[2])
			}
			if len(matches) > 3 && matches[3] != "" {
				constraint.References= matches[3]
				// p.parseColumnList(matches[3])
			}
		}
	case strings.HasPrefix(constraintDefUpper, "UNIQUE"):
		constraint.Type = "UNIQUE"
		if matches := p.uniqueRegex.FindStringSubmatch(constraintDef); len(matches) > 1 {
			constraint.Columns = p.parseColumnList(matches[1])
		}
	case strings.HasPrefix(constraintDefUpper, "CHECK"):
		constraint.Type = "CHECK"
		
	default:
		constraint.Type = "OTHER"
	}

	table.Constraints = append(table.Constraints, constraint)
	return nil
}

func (p *SQLParser) parseInlinePrimaryKey(table *models.Table, definition string) error {
	constraintName := fmt.Sprintf("%s_pkey", table.Name)
	
	// Extract column name (first word)
	parts := strings.Fields(definition)
	if len(parts) == 0 {
		return fmt.Errorf("invalid primary key definition")
	}
	
	columnName := p.cleanIdentifier(parts[0])
	
	constraint := &models.Constraint{
		Name:       constraintName,
		Type:       "PRIMARY KEY",
		Columns:    []string{columnName},
		RawSQL: "PRIMARY KEY",
	}


	table.Constraints = append(table.Constraints, constraint)
	return nil
}

// parseInlineForeignKey parses an inline FOREIGN KEY constraint
func (p *SQLParser) parseInlineForeignKey(table *models.Table, definition string) error {
	parts := strings.Fields(definition)
	if len(parts) == 0 {
		return fmt.Errorf("invalid foreign key definition")
	}
	
	columnName := p.cleanIdentifier(parts[0])
	constraintName := fmt.Sprintf("%s_%s_fkey", table.Name, columnName)
	
	matches := p.foreignKeyRegex.FindStringSubmatch(definition)
	constraint := &models.Constraint{
		Name:       constraintName,
		Type:       "FOREIGN KEY",
		Columns:    []string{columnName},
		RawSQL: definition,
	}
	
	if len(matches) > 2 {
		// constraint.References = p.cleanIdentifier(matches[2])
		constraint.References = matches[2]
		if len(matches) > 3 && matches[3] != "" {
			constraint.Columns = p.parseColumnList(matches[3])
		}
	}

	
	table.Constraints = append(table.Constraints, constraint)
	return nil
}

// parseInlineUnique parses an inline UNIQUE constraint
func (p *SQLParser) parseInlineUnique(table *models.Table, definition string) error {
	parts := strings.Fields(definition)
	if len(parts) == 0 {
		return fmt.Errorf("invalid unique constraint definition")
	}
	
	columnName := p.cleanIdentifier(parts[0])
	constraintName := fmt.Sprintf("%s_%s_key", table.Name, columnName)
	
	constraint := &models.Constraint{
		Name:       constraintName,
		Type:       "UNIQUE",
		Columns:    []string{columnName},
		RawSQL: "UNIQUE",
	}

	table.Constraints = append(table.Constraints, constraint)
	return nil
}

// parseCreateIndex parses a CREATE INDEX statement
func (p *SQLParser) parseCreateIndex(schema *models.Schema, stmt string) error {
	matches := p.indexRegex.FindStringSubmatch(stmt)
	if len(matches) < 6 {
		return fmt.Errorf("invalid CREATE INDEX statement")
	}

	isUnique := matches[1] != ""
	indexName := p.cleanIdentifier(matches[2])
	tableName := p.cleanIdentifier(matches[3])
	method := "btree" // default
	if matches[4] != "" {
		method = strings.ToLower(matches[4])
	}
	columns := p.parseColumnList(matches[5])

	index := &models.Index{
		Name:     indexName,
		Table:    tableName,
		Columns:  columns,
		IsUnique: isUnique,
		Method:   method,
	}

	schema.Indexes = append(schema.Indexes, index)

	return nil
}

// parseCreateSequence parses a CREATE SEQUENCE statement
func (p *SQLParser) parseCreateSequence(schema *models.Schema, stmt string) error {
	matches := p.sequenceRegex.FindStringSubmatch(stmt)
	if len(matches) < 2 {
		return fmt.Errorf("invalid CREATE SEQUENCE statement")
	}

	sequenceName := p.cleanIdentifier(matches[1])
	sequence := &models.Sequence{
		Name:       sequenceName,
		Start: 1,
		Increment:  1,
	}

	// Parse sequence options if present
	if len(matches) > 2 && matches[2] != "" {
		options := strings.ToUpper(matches[2])
		
		startRegex := regexp.MustCompile(`START\s+WITH\s+(\d+)`)
		if startMatches := startRegex.FindStringSubmatch(options); len(startMatches) > 1 {
			if val, err := strconv.ParseInt(startMatches[1], 10, 64); err == nil {
				sequence.Start = val
			}
		}
		
		incRegex := regexp.MustCompile(`INCREMENT\s+BY\s+(\d+)`)
		if incMatches := incRegex.FindStringSubmatch(options); len(incMatches) > 1 {
			if val, err := strconv.ParseInt(incMatches[1], 10, 64); err == nil {
				sequence.Increment = val
			}
		}
	}

	schema.Sequences = append(schema.Sequences, sequence)
	return nil
}

// parseCreateFunction parses a CREATE FUNCTION statement
func (p *SQLParser) parseCreateFunction(schema *models.Schema, stmt string) error {
	matches := p.functionRegex.FindStringSubmatch(stmt)
	if len(matches) < 3 {
		return fmt.Errorf("invalid CREATE FUNCTION statement")
	}

	functionName := p.cleanIdentifier(matches[1])
	returnType := matches[2]
	language := "sql" // default
	if len(matches) > 3 && matches[3] != "" {
		language = strings.ToLower(matches[3])
	}

	function := &models.Function{
		Name:       functionName,
		ReturnType: returnType,
		Language:   language,
		Definition: stmt, // Store the full definition for comparison
	}

	schema.Functions = append(schema.Functions, function)
	return nil
}

// parseAlterTable parses an ALTER TABLE statement
func (p *SQLParser) parseAlterTable(schema *models.Schema, stmt string) error {
	matches := p.alterTableRegex.FindStringSubmatch(stmt)
	if len(matches) < 3 {
		return fmt.Errorf("invalid ALTER TABLE statement")
	}

	tableName := p.cleanIdentifier(matches[1])
	alterDef := strings.TrimSpace(matches[2])
	alterDefUpper := strings.ToUpper(alterDef)

	var exists bool = false
	var table *models.Table
	for i, t:= range schema.Tables {
		if t.Name == tableName {
			exists = true
			table = schema.Tables[i]
		}
	}
	
	if !exists {
		table = &models.Table{
			Name:        tableName,
			Columns:     make([]*models.Column, 0),
			Constraints: make([]*models.Constraint, 0),
		}
		schema.Tables = append(schema.Tables, table)
	}

	// Parse different ALTER TABLE operations
	switch {
	case strings.HasPrefix(alterDefUpper, "ADD COLUMN"):
		return p.parseAlterAddColumn(table, alterDef)
	case strings.HasPrefix(alterDefUpper, "DROP COLUMN"):
		return p.parseAlterDropColumn(table, alterDef)
	case strings.HasPrefix(alterDefUpper, "ALTER COLUMN"):
		return p.parseAlterColumn(table, alterDef)
	case strings.HasPrefix(alterDefUpper, "ADD CONSTRAINT"):
		return p.parseAlterAddConstraint(table, alterDef)
	case strings.HasPrefix(alterDefUpper, "DROP CONSTRAINT"):
		return p.parseAlterDropConstraint(table, alterDef)
	default:
		// Log unsupported ALTER TABLE operation
		fmt.Printf("Warning: unsupported ALTER TABLE operation: %s\n", alterDef)
	}

	return nil
}

// parseAlterAddColumn parses ADD COLUMN in ALTER TABLE
func (p *SQLParser) parseAlterAddColumn(table *models.Table, alterDef string) error {
	
	columnDef := strings.TrimSpace(alterDef[10:]) // len("ADD COLUMN") = 10
	return p.parseColumnDefinition(table, columnDef)
}

// parseAlterDropColumn parses DROP COLUMN in ALTER TABLE
func (p *SQLParser) parseAlterDropColumn(table *models.Table, alterDef string) error {
	columnName := strings.TrimSpace(alterDef[11:]) // len("DROP COLUMN") = 11
	columnName = p.cleanIdentifier(columnName)
	
	var colIndex int
	for i, c := range table.Columns {
		if c.Name == columnName {
			colIndex = i
			break
		}
	}
	table.Columns = append(table.Columns[:colIndex], table.Columns[colIndex+1:]... )
	return nil
}

// parseAlterColumn parses ALTER COLUMN in ALTER TABLE
func (p *SQLParser) parseAlterColumn(table *models.Table, alterDef string) error {
	//TODO: This would handle operations like ALTER COLUMN SET NOT NULL, SET DEFAULT, etc.
	fmt.Printf("Warning: ALTER COLUMN not fully implemented: %s\n", alterDef)
	return nil
}

// parseAlterAddConstraint parses ADD CONSTRAINT in ALTER TABLE
func (p *SQLParser) parseAlterAddConstraint(table *models.Table, alterDef string) error {
	constraintDef := strings.TrimSpace(alterDef[4:]) // len("ADD ") = 4
	return p.parseTableConstraint(table, constraintDef)
}

// parseAlterDropConstraint parses DROP CONSTRAINT in ALTER TABLE
func (p *SQLParser) parseAlterDropConstraint(table *models.Table, alterDef string) error {
	constraintName := strings.TrimSpace(alterDef[15:]) // len("DROP CONSTRAINT") = 15
	constraintName = p.cleanIdentifier(constraintName)
	
	
	for i, c := range table.Constraints {
		if c.Name == constraintName {
			table.Constraints = append(table.Constraints[:i], table.Constraints[i+1:]...)
			break
		}
	}
	
	return nil
}

// parseColumnList parses a comma-separated list of column names
func (p *SQLParser) parseColumnList(columnList string) []string {
	columns := strings.Split(columnList, ",")
	for i, col := range columns {
		columns[i] = p.cleanIdentifier(strings.TrimSpace(col))
	}
	return columns
}

// cleanIdentifier removes quotes and trims whitespace from identifiers
func (p *SQLParser) cleanIdentifier(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	// Remove double quotes if present
	if strings.HasPrefix(identifier, "\"") && strings.HasSuffix(identifier, "\"") {
		identifier = identifier[1 : len(identifier)-1]
	}
	// Remove backticks if present
	if strings.HasPrefix(identifier, "`") && strings.HasSuffix(identifier, "`") {
		identifier = identifier[1 : len(identifier)-1]
	}
	return identifier
}

// Parse parses SQL DDL content and returns a schema model
func (p *SQLParser) Parse(content string) (*models.Schema, error) {
	schema := &models.Schema{
		Tables:    make([]*models.Table, 0),
		Views:     make([]*models.View, 0),
		Triggers:  make([]*models.Trigger, 0),
		Indexes:   make([]*models.Index, 0),
		Functions: make([]*models.Function, 0),
		Sequences: make([]*models.Sequence, 0),
	}

	//TODO: complete
	statements := p.splitStatements(content)

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if err := p.parseStatements(schema, stmt); err != nil {
			// Log error but continue parsing other statements
			fmt.Printf("Warning: failed to parse statement: %v\n", err)
		}
	}

	return schema, nil

}
