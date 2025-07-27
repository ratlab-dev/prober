package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// NewReadProbe creates a ReadProbe and initializes the DB connection
func NewReadProbe(host, user, password, database string) (*ReadProbe, error) {
	return &ReadProbe{
		Host:     host,
		User:     user,
		Password: password,
		Database: database,
		DB:       nil,
	}, nil
}

// NewWriteProbe creates a WriteProbe and initializes the DB connection
func NewWriteProbe(host, user, password, database string) (*WriteProbe, error) {
	return &WriteProbe{
		Host:     host,
		User:     user,
		Password: password,
		Database: database,
		DB:       nil,
	}, nil
}

type ReadProbe struct {
	Host     string
	User     string
	Password string
	Database string
	DB       *sql.DB
}

func (p *ReadProbe) Probe(ctx context.Context) error {
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
	err := p.DB.QueryRowContext(ctx, "SELECT 1").Scan(&one)
	if err != nil {
		return err
	}
	if one != 1 {
		return fmt.Errorf("unexpected result from SELECT 1: %d", one)
	}
	return nil
}

type WriteProbe struct {
	Host     string
	User     string
	Password string
	Database string
	DB       *sql.DB
}

func (p *WriteProbe) Probe(ctx context.Context) error {
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

	// Ensure probe_test table exists
	_, err := p.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS probe_test (id INT PRIMARY KEY AUTO_INCREMENT, val VARCHAR(32))`)
	if err != nil {
		return fmt.Errorf("failed to create probe_test table: %w", err)
	}

	// Insert a test row
	res, err := p.DB.ExecContext(ctx, `INSERT INTO probe_test (val) VALUES ('probe')`)
	if err != nil {
		return fmt.Errorf("failed to insert test row: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	// Delete the test row
	_, err = p.DB.ExecContext(ctx, `DELETE FROM probe_test WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete test row: %w", err)
	}
	return nil
}
