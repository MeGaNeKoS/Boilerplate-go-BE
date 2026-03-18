package db

import (
	"context"
	"database/sql"
	"fmt"
	"project-template/infrastructure/config"
	"project-template/pkg/logger"
	"sync"
	"time"

	// MySQL driver import for side effects. It registers itself with the database/sql package
	_ "github.com/go-sql-driver/mysql"

	// Register goqu MySQL dialect for SQL building
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
)

type DB interface {
	New() error
	SetLogger(logger.Logger)
	GetDB() *sql.DB
	// GetGoquDialect returns the goqu dialect for building SQL queries
	GetGoquDialect() goqu.DialectWrapper
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	// WithTransaction executes fn within a database transaction.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	// BeginTx starts a transaction with the provided options. Placeholder for advanced use cases.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	Close() error
	Ping() error
}

type dbImpl struct {
	db     *sql.DB
	once   sync.Once
	logger logger.Logger
}

var (
	instance *dbImpl
	once     sync.Once
)

// SetLogger configures a logger for this DB instance.
func (d *dbImpl) SetLogger(l logger.Logger) {
	d.logger = l
}

// New establishes the database connection using configuration values. It is
// safe to call multiple times; initialization occurs once.
func (d *dbImpl) New() error {
	var err error
	d.once.Do(func() {
		cfg := config.Cfg

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.DBName,
		)
		d.db, err = sql.Open("mysql", dsn)
		if err != nil {
			d.logFatal("Error connecting to database: %v", err)
		}

		d.db.SetConnMaxLifetime(time.Minute * 4)
		d.db.SetMaxOpenConns(30)
		d.db.SetMaxIdleConns(30)

		if err = d.db.Ping(); err != nil {
			if closeErr := d.db.Close(); closeErr != nil {
				d.logFatal("Error closing the database connection: %v", closeErr)
			}
			d.db = nil
			d.logFatal("Lost connection to database: %v", err)
		}

		d.logInfo("Successfully connected to database")
	})

	if d.db == nil {
		return fmt.Errorf("failed to initialize the database connection")
	}
	return nil
}
