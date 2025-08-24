package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type ReadProbe struct {
	Region   string
	Host     string
	User     string
	Password string
	Database string
	Query    string
	DB       *sql.DB
}

// NewReadProbe creates a ReadProbe and initializes the DB connection
func NewReadProbe(host, user, password, database, query string) (*ReadProbe, error) {
	return &ReadProbe{
		Host:     host,
		User:     user,
		Password: password,
		Database: database,
		Query:    query,
		DB:       nil,
	}, nil
}

// NewWriteProbe creates a WriteProbe and initializes the DB connection
func NewWriteProbe(host, user, password, database, query string) (*WriteProbe, error) {
	return &WriteProbe{
		Host:     host,
		User:     user,
		Password: password,
		Database: database,
		Query:    query,
		DB:       nil,
	}, nil
}

func (p *ReadProbe) Probe(ctx context.Context) error {
	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][MySQL][%s] Executing query: %s", p.Host, p.Query)
	}
	if p.Host == "" || p.User == "" || p.Database == "" {
		// Noop if config is incomplete
		return nil
	}
	if p.DB == nil {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", p.User, p.Password, p.Host, p.Database)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		p.DB = db
	}
	var one int
	query := p.Query
	if query == "" {
		query = "SELECT 1"
	}
	err := p.DB.QueryRowContext(ctx, query).Scan(&one)
	if err != nil {
		return err
	}
	if one != 1 {
		return fmt.Errorf("unexpected result from query: %d", one)
	}
	return nil
}

func (p *ReadProbe) MetadataString() string {
	return fmt.Sprintf("Host: %s | Database: %s | User: %s | Region: %s", p.Host, p.Database, p.User, p.Region)
}

type WriteProbe struct {
	Region   string
	Host     string
	User     string
	Password string
	Database string
	Query    string
	DB       *sql.DB
}

func (p *WriteProbe) Probe(ctx context.Context) error {
	if os.Getenv("DEBUG") == "1" {
		log.Printf("[DEBUG][MySQL][%s] Executing query: %s", p.Host, p.Query)
	}
	if p.Host == "" || p.User == "" || p.Database == "" {
		// Noop if config is incomplete
		return nil
	}
	if p.DB == nil {
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", p.User, p.Password, p.Host, p.Database)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		p.DB = db
	}
	var one int
	query := p.Query
	if query == "" {
		query = "SELECT 1"
	}
	err := p.DB.QueryRowContext(ctx, query).Scan(&one)
	if err != nil {
		return err
	}
	if one != 1 {
		return fmt.Errorf("unexpected result from query: %d", one)
	}
	return nil
}

func (p *WriteProbe) MetadataString() string {
	return fmt.Sprintf("Host: %s | Database: %s | User: %s | Region: %s", p.Host, p.Database, p.User, p.Region)
}
