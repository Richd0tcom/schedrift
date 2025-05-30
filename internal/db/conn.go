package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/Richd0tcom/schedrift/internal/config"
	"github.com/Richd0tcom/schedrift/internal/db/postgres"
)

type DatabaseDriver string

const (
	PostgreSQL DatabaseDriver = "postgres"
	MySQL      DatabaseDriver = "mysql"
)

type Connection interface {
	Close() error
	DB() *sql.DB
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
	GetVersion() (string, error)
}

func NewConnection(cfg config.DatabaseConfig) (Connection, error) {
	dbType := DatabaseDriver(strings.ToLower(cfg.Driver))

	switch dbType {
		case PostgreSQL:
			conn, err:= postgres.NewConnection(cfg)

			if err != nil {
				return nil, err
			}
			

			return conn, err


		default:
			return nil, fmt.Errorf("invalid driver type")

	}
}
