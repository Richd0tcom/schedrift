package main

import (
	"fmt"
	"os"

	"github.com/Richd0tcom/schedrift/internal/db"
	"github.com/spf13/cobra"
)

func main() {
	// Initialize the root command
	rootCmd := &cobra.Command{
		Use:   "schemarift",
		Short: "A tool to detect schema drift between database and code",
		Long: `Schema Drift Detector (schemarift) is a CLI tool that helps you detect
differences between your production database schema and your reference schema.
It allows you to prevent unexpected schema changes and maintain consistency.`,
	}

	// Add commands
	rootCmd.AddCommand(createDumpCommand())
	rootCmd.AddCommand(createCheckCommand())
	rootCmd.AddCommand(createDiffCommand())
	rootCmd.AddCommand(createInitCommand())
	rootCmd.AddCommand(createVersionCommand())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Create the dump command
func createDumpCommand() *cobra.Command {
	dumpCmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump current database schema to a file",
		Long: `Connect to the specified database and dump its schema to a file
or stdout. The schema includes tables, columns, indices, constraints,
and other database objects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse flags
			url, _ := cmd.Flags().GetString("url")
			schemaName, _ := cmd.Flags().GetString("schema")
			output, _ := cmd.Flags().GetString("output")
			
			// Create connection
			conn, err := db.NewConnection(url)
			if err != nil {
				return fmt.Errorf("failed to create database connector: %w", err)
			}
			
			// Extract schema
			schema, err := conn.ExtractSchema(schemaName)
			if err != nil {
				return fmt.Errorf("failed to extract schema: %w", err)
			}
			
			// Write to output
			if output == "" {
				// Write to stdout
				fmt.Println(schema.ToSQL())
			} else {
				// Write to file
				err = os.WriteFile(output, []byte(schema.ToSQL()), 0644)
				if err != nil {
					return fmt.Errorf("failed to write schema to file: %w", err)
				}
				fmt.Printf("Schema dumped to %s\n", output)
			}
			
			return nil
		},
	}
	
	// Add flags
	dumpCmd.Flags().String("url", "", "Database connection URL (required)")
	dumpCmd.Flags().String("schema", "public", "Schema name to extract")
	dumpCmd.Flags().String("output", "", "Output file path (stdout if not specified)")
	
	// Mark required flags
	dumpCmd.MarkFlagRequired("url")
	
	return dumpCmd
}

// Create the check command (placeholder for now)
func createCheckCommand() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Compare database schema with reference",
		Long: `Connect to the database and compare its current schema with a reference
schema file. Report differences and optionally fail if significant differences
are found.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// This will be implemented later
			fmt.Println("Check command not yet implemented")
			return nil
		},
	}
	
	// Add flags (we'll implement this later)
	
	return checkCmd
}

// Create the diff command (placeholder for now)
func createDiffCommand() *cobra.Command {
	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between two schema files",
		Long: `Compare two schema files and show the differences between them.
This command does not connect to a database but works with schema files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// This will be implemented later
			fmt.Println("Diff command not yet implemented")
			return nil
		},
	}
	
	// Add flags (we'll implement this later)
	
	return diffCmd
}

// Create the init command (placeholder for now)
func createInitCommand() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long: `Create a default configuration file in the current directory
or at the specified path. The configuration file contains settings for
database connections, schema comparison, and notifications.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// This will be implemented later
			fmt.Println("Init command not yet implemented")
			return nil
		},
	}
	
	// Add flags (we'll implement this later)
	
	return initCmd
}

// Create the version command
func createVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Schema Drift Detector v0.1.0")
		},
	}
}