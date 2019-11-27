package sqlite3

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	migrateSqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver.

	"malta/internal/database/sqlite3/migration"
)

// ClientLifecycleHook is used during the start an stop of the client.
type ClientLifecycleHook interface {
	open() error
	close() error
}

// ClientConfig holds the information to initialize the client.
type ClientConfig struct {
	// Location of the database file.
	DatabaseFile string

	// Hooks to initialize other components during the start and stop of the client.
	ClientLifecycleHook []ClientLifecycleHook

	// If these valures are less or equal to zero, then it's unbonded.
	MaxOpenConnections int
	MaxIdleConnections int
	ConnectionLifetime time.Duration
}

// Client used to access the SQLite3.
type Client struct {
	Config   ClientConfig
	instance *sql.DB
}

// Init internal state.
func (c *Client) Init() error {
	if c.Config.DatabaseFile == "" {
		return fmt.Errorf("missing database file")
	}
	return nil
}

// Start the client.
func (c *Client) Start() (err error) {
	path := fmt.Sprintf("%s?_journal=wal", c.Config.DatabaseFile)
	c.instance, err = sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("failed to start sqlite3: %w", err)
	}
	c.instance.SetMaxOpenConns(c.Config.MaxOpenConnections)
	c.instance.SetMaxIdleConns(c.Config.MaxIdleConnections)
	c.instance.SetConnMaxLifetime(c.Config.ConnectionLifetime)

	if err := c.execMigration(); err != nil {
		return fmt.Errorf("error during migration: %w", err)
	}

	for _, fn := range c.Config.ClientLifecycleHook {
		if err := fn.open(); err != nil {
			return fmt.Errorf("open on lifecycle hook failed: %w", err)
		}
	}
	return nil
}

// Stop the client.
func (c *Client) Stop() error {
	if err := c.instance.Close(); err != nil {
		return fmt.Errorf("failed to stop the sqlite3: %w", err)
	}

	for _, fn := range c.Config.ClientLifecycleHook {
		if err := fn.close(); err != nil {
			return fmt.Errorf("close on lifecycle hook failed: %w", err)
		}
	}
	return nil
}

// Begin create a transaction.
func (c *Client) Begin(
	ctx context.Context, readOnly bool, level sql.IsolationLevel,
) (*sql.Tx, error) {
	return c.instance.BeginTx(ctx, &sql.TxOptions{
		Isolation: level,
		ReadOnly:  readOnly,
	})
}

func (c *Client) execMigration() (err error) {
	var mm migration.Manager
	mm.Init()

	driver, err := migrateSqlite3.WithInstance(c.instance, &migrateSqlite3.Config{
		MigrationsTable: "migrations",
		DatabaseName:    "malta",
	})
	if err != nil {
		return fmt.Errorf("failed to create the sqlite3 driver to migrate: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("static://malta", "malta", driver)
	if err != nil {
		return fmt.Errorf("failed to create the migration instance: %w", err)
	}

	if err := m.Up(); err != nil && (err != os.ErrNotExist && err != migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate the database: %w", err)
	}
	return nil
}
