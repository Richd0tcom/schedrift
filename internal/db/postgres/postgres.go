package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Richd0tcom/schedrift/internal/config"
	_ "github.com/lib/pq"
	"google.golang.org/protobuf/types/known/anypb"
)

type Connection struct {
	db *sql.DB
}

func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	connStr := fmt.Sprintf(
	"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", 
	cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DatabaseName, cfg.SSLMode)


	db, err:= sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Connection{db: db}, nil
}


func (c *Connection) Close() error {
	return c.db.Close()
}

func (c *Connection) DB() *sql.DB {
	return c.db
}

func (c *Connection) Query(query string, args ...any) (*sql.Rows, error) {
	return c.db.Query(query, args...)
}

func (c *Connection) QueryRow(query string, args ...any) *sql.Row {
	return c.db.QueryRow(query, args...)
}

// Exec executes a query without returning any rows
func (c *Connection) Exec(query string, args ...any) (sql.Result, error) {
	return c.db.Exec(query, args...)
}

func (c *Connection) GetPGVersion() (string, error) {
	var version string
	err := c.QueryRow("SELECT version()").Scan(&version)

	if err != nil {
		return "", fmt.Errorf("failed to get PostgreSQL version: %w", err)
	}
	return version, nil
}