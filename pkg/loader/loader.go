package loader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Richd0tcom/schedrift/internal/models"
	"github.com/Richd0tcom/schedrift/pkg/parser"
)

//TODO: Load from Git

type LoaderConfig struct {
	FilePath string
	TempDir string
	CleanupTemp bool
}

type SchemaLoader struct {
	config *LoaderConfig
	parser *parser.SQLParser
}

func NewSchemaLoader(config *LoaderConfig) *SchemaLoader {
	return &SchemaLoader{
		config: config,
		parser: parser.NewSQLParser(),
	}
}

func (ld *SchemaLoader) LoadFromFile(filePath string) (*models.Schema, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ld.parser.Parse(string(content))
}

func (ld *SchemaLoader) LoadFromDir(dir string) (*models.Schema, error) {
	file, err:= ld.findSchemaFile(dir)

	if err != nil {
		return nil, fmt.Errorf("failed to find schema file: %w", err)
	}
	return ld.parser.Parse(file)
}

func (ld *SchemaLoader) findSchemaFile(repoDir string) (string, error){
	if ld.config.FilePath != "" {
		fullPath := filepath.Join(repoDir, ld.config.FilePath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
		return "", fmt.Errorf("specified schema file not found: %s", ld.config.FilePath)
	}

	// Common schema file patterns
	//TODO:move to different file and add more patterns 
	patterns := []string{
		"**/schema.sql",
		"**/database.sql",
		"**/init.sql",
		"**/migrations/*.sql",
		"**/*schema*.sql",
		"db/schema.sql",
		"database/schema.sql",
		"sql/schema.sql",
		"migrations/schema.sql",

	}


	var candidates []string


	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(repoDir, pattern))
		if err != nil {
			continue
		}
		candidates = append(candidates, matches...)
	}

	if len(candidates) == 0 {
		err:= filepath.Walk(repoDir, func (path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".sql") {
				
				if ld.isLikelySchemaFile(path) {
					candidates = append(candidates, path)
				}
			}

			return  nil
		})

		if err != nil {
			return "", fmt.Errorf("failed to walk repository directory: %w", err)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no schema files found in repository")
	}

	// Rank candidates by likelihood of being the main schema file
	//TODO: change implentation to parse all candidates not just best
	bestCandidate := ld.rankSchemaFiles(candidates)
	return bestCandidate, nil
}

func (ld *SchemaLoader) rankSchemaFiles(candidates []string) string {
	if len(candidates) == 1 {
		return candidates[0]
	}

	type scoredFile struct {
		path  string
		score int
	}

	var scored []scoredFile

	for _, candidate := range candidates {
		score := 0
		filename := strings.ToLower(filepath.Base(candidate))
		dir := strings.ToLower(filepath.Dir(candidate))

		
		if strings.Contains(filename, "schema") {
			score += 10
		}

		if filename == "schema.sql" {
			score += 20
		}

		// 
		if strings.Contains(dir, "db") || strings.Contains(dir, "database") {
			score += 5
		}

		if strings.Contains(dir, "migration") {
			score -= 5
		}

		if info, err := os.Stat(candidate); err == nil {
			size := info.Size()
			if size > 10000 { // Files larger than 10KB
				score += 3
			} else if size > 1000 { // Files larger than 1KB
				score += 1
			}
		}

		scored = append(scored, scoredFile{path: candidate, score: score})
	}

	best := scored[0]
	for _, sf := range scored[1:] {
		if sf.score > best.score {
			best = sf
		}
	}

	return best.path
}


func (ld *SchemaLoader) isLikelySchemaFile(path string) bool {
	file, err:= os.Open(path)

	if err != nil {
		return false
	}

	defer file.Close()

	scanner:= bufio.NewScanner(file)
	
	lineCount := 0
	schemaKeywords := 0

	for scanner.Scan() && lineCount <100 {
		line := strings.ToLower(scanner.Text())
		lineCount++


		if strings.Contains(line, "create table") ||
		   strings.Contains(line, "alter table") ||
		   strings.Contains(line, "create index") ||
		   strings.Contains(line, "create sequence") ||
		   strings.Contains(line, "create function") ||
		   strings.Contains(line, "add constraint") ||
		   strings.Contains(line, "primary key") ||
		   strings.Contains(line, "foreign key") {
			schemaKeywords++
		}
	}

	return schemaKeywords >= 3
}


func (sl *SchemaLoader) ValidateSchema(schema *models.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema is nil")
	}

	if len(schema.Tables) == 0 {
		return fmt.Errorf("schema contains no tables")
	}


	// Check for orphaned indexes
	for _, index := range schema.Indexes {
		found := false
		for _, table := range schema.Tables {
			
			if table.Name == index.Table {
				found = true
				break
			}
			
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("orphaned index found: %s", index.Name)
		}
	}

	return nil
}

func (ld *SchemaLoader) NormalizeSchema(schema *models.Schema) *models.Schema {
	normalized := &models.Schema{
		Tables:      make([]*models.Table,0),
		Indexes:     make([]*models.Index, 0 ),
		Sequences:   make([]*models.Sequence,0),
		Functions:   make([]*models.Function, 0),

		//TODO: add others
	}

	// Normalize table names and copy tables
	for _, table := range schema.Tables {
		normalizedTable := ld.normalizeTable(table)
		normalized.Tables = append(normalized.Tables, normalizedTable)
	}

	// Copy and normalize other objects
	for _, index := range schema.Indexes {
		index.Name = strings.ToLower(index.Name)
		normalized.Indexes = append(normalized.Indexes, index)
	}



	for _, sequence := range schema.Sequences {
		sequence.Name = strings.ToLower(sequence.Name)	
		normalized.Sequences = append(normalized.Sequences, sequence)
	}

	for _, function := range schema.Functions {
		function.Name = strings.ToLower(function.Name)
		normalized.Functions = append(normalized.Functions, function)
	}

	return normalized
}

// normalizeTable normalizes a table for consistent comparison
func (ld *SchemaLoader) normalizeTable(table *models.Table) *models.Table {
	normalized := &models.Table{
		Name:        strings.ToLower(table.Name),
		Columns:     make([]*models.Column, 0),
		Constraints: make([]*models.Constraint, 0),
	}

	// Normalize column names and types
	for _ , column := range table.Columns {
		normalizedCol := &models.Column{
			Name:         strings.ToLower(column.Name),
			DataType:         ld.normalizeType(column.DataType),
			IsNullable:     column.IsNullable,
			DefaultValue: column.DefaultValue,
		}
		normalized.Columns = append(normalized.Columns, normalizedCol)
	}

	for _, constraint := range table.Constraints {
		constraint.Name = strings.ToLower(constraint.Name)
		normalized.Constraints = append(normalized.Constraints, constraint)
	}

	return normalized
}

// normalizeType normalizes a PostgreSQL data type for consistent comparison
func (ld *SchemaLoader) normalizeType(dataType string) string {
	// Remove extra whitespace
	normalized := strings.TrimSpace(dataType)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	
	// Convert to lowercase for comparison
	normalized = strings.ToLower(normalized)
	
	// Normalize common type aliases
	typeAliases := map[string]string{
		"int":     "integer",
		"int4":    "integer",
		"int8":    "bigint",
		"float4":  "real",
		"float8":  "double precision",
		"bool":    "boolean",
		"varchar": "character varying",
	}
	
	for alias, canonical := range typeAliases {
		if strings.HasPrefix(normalized, alias) {
			normalized = strings.Replace(normalized, alias, canonical, 1)
		}
	}
	
	return normalized
}