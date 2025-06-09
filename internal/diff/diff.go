package diff

import (
	"fmt"

	"github.com/Richd0tcom/schedrift/internal/models"
)

//TODO: concurrent comparison

type SeverityLevel string

const (
	None SeverityLevel = "none"

	Low SeverityLevel = "low"

	Medium SeverityLevel = "medium"

	High SeverityLevel = "high"
)

type ChangeType string

const (
	Added ChangeType = "added"

	Removed ChangeType = "removed"

	Modified ChangeType = "modified"

	Renamed ChangeType = "renamed"
)

type Change struct {
	Type        ChangeType
	ObjectType  string
	ObjectName  string
	ParentName  string
	Description string
	Severity    SeverityLevel

	Details map[string]any
}

type Diff struct {
	Changes []Change
	Summary map[string]int
}

func NewDiff() *Diff {
	return &Diff{
		Changes: make([]Change, 0),
		Summary: make(map[string]int),
	}
}

func (d *Diff) AddChange(c Change) {
	d.Changes = append(d.Changes, c)

	key := fmt.Sprintf("%s.%s", c.ObjectType, c.Type)
	d.Summary[key]++
}

func (d *Diff) HasChanges() bool {
	return len(d.Changes) > 0
}

func (d *Diff) HasSeverity(minSeverity SeverityLevel) bool {
	severityMap := map[SeverityLevel]int{
		None:   0,
		Low:    1,
		Medium: 2,
		High:   3,
	}
	minLevel := severityMap[minSeverity]

	for _, change := range d.Changes {
		if severityMap[change.Severity] >= minLevel {
			return true
		}
	}

	return false
}

func isBreakingTypeChange(oldType, newType string) bool

func compareColumns(diff *Diff, tableName string, sourceTable, targetTable *models.Table) {
	srcCols := make(map[string]*models.Column)
	targetCols := make(map[string]*models.Column)

	for _, t := range sourceTable.Columns {
		srcCols[t.Name] = t
	}

	for _, t := range targetTable.Columns {
		targetCols[t.Name] = t
	}

	//check for added cols
	for colName, col := range targetCols {
		if _, exists := srcCols[colName]; !exists {

			severity := Medium

			if col.IsNullable {
				//Adding NOT NULL columns are HIGH severity
				severity = High
			}
			diff.AddChange(Change{
				Type:        Added,
				ObjectType:  "column",
				ParentName:  tableName, //TODO: see if name can be gotten from table w/o consequence
				Severity:    severity,
				Description: fmt.Sprintf("Column %s.%s was added", tableName, colName),
				Details: map[string]any{
					"data_type": col.DataType,
					"nullable":  col.IsNullable,
					"default":   col.DefaultValue,
				},
			})
		}
	}

	
	for colName, col := range srcCols {
		_, exists := targetCols[colName];
		if !exists {
			//handle removed columns
			

			diff.AddChange(Change{
				Type:        Removed,
				ObjectType:  "column",
				ParentName:  tableName,
				Severity:    High, // Removing a column is high severity
				Description: fmt.Sprintf("Column %s.%s was removed", tableName, colName),
				Details: map[string]any{
					"data_type": col.DataType,
					"nullable":  col.IsNullable,
				},
			})
		}
	}

	for srcColName, srcCol := range srcCols {
		tgtCol, exists := targetCols[srcColName];
		if !exists {
			continue
		}

		if srcCol.DataType != tgtCol.DataType {
			severity:= Medium

			if isBreakingTypeChange(srcCol.DataType, tgtCol.DataType) {
				severity = High
			}

			diff.AddChange(Change{
				Type: Modified,
				ObjectType: "column",
				ObjectName: srcColName,
				ParentName: tableName,
				Severity: severity,
				Description: fmt.Sprintf("Column %s.%s data type changed from %s to %s", tableName, srcColName, srcCol.DataType, tgtCol.DataType),
				Details: map[string]any{
					"old_data_type": srcCol.DataType,
					"new_data_type": tgtCol.DataType,

				},
			})


		} 

		if srcCol.IsNullable != tgtCol.IsNullable { 
			severity := Low
			nullChange := "made nullable"

			if !tgtCol.IsNullable {
				severity = High
				nullChange = "made NOT NULL"
			}

			diff.AddChange(Change{
				Type: Modified,
				ObjectType: "column",
				ObjectName: srcColName,
				ParentName: tableName,
				Severity: severity,
				Description: fmt.Sprintf("Column %s.%s was %s", tableName, srcColName, nullChange),
				Details: map[string]any{
					"old_nullable": srcCol.IsNullable,
					"new_nullable": tgtCol.IsNullable,
				},
			})
		}

		if srcCol.DefaultValue != tgtCol.DefaultValue {

			oldDefault := "none"
			newDefault := "none"

			if srcCol.DefaultValue != "" {
				oldDefault = srcCol.DefaultValue
			}

			if tgtCol.DefaultValue != "" {
				newDefault = tgtCol.DefaultValue
			}

			diff.AddChange(Change{
				Type: Modified,
				ObjectType: "column",
				ObjectName: srcColName,
				ParentName: tableName,
				Severity: Medium,
				Description: fmt.Sprintf("Column %s.%s default value changed from %s to %s", tableName, srcColName, oldDefault, newDefault),
				Details: map[string]any{
					"old_default": oldDefault,
					"new_default": newDefault,
				},
			})
		}
	}



}

func compareTables(diff *Diff, src, target *models.Schema) {
	
}
