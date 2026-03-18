package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	stdlog "log"
	"project-template/infrastructure/utils"

	"github.com/doug-martin/goqu/v9"
)

// GetDBInstance returns a singleton instance of the database implementation.
func GetDBInstance() DB {
	once.Do(func() {
		instance = &dbImpl{}
	})
	return instance
}

// GetGoquDialect provides the goqu dialect used for building SQL queries.
func (d *dbImpl) GetGoquDialect() goqu.DialectWrapper {
	return goqu.Dialect("mysql")
}

// GetDB retrieves the underlying *sql.DB connection. It panics if the
// connection was not initialized.
func (d *dbImpl) GetDB() *sql.DB {
	if d.db == nil {
		d.logFatal("Database connection is not initialized. Call Connect first.")
	}
	return d.db
}

func (d *dbImpl) logFatal(format string, args ...interface{}) {
	if d.logger != nil {
		d.logger.FatalF(format, args...)
	} else {
		stdlog.Fatalf(format, args...)
	}
}

func (d *dbImpl) logInfo(format string, args ...interface{}) {
	if d.logger != nil {
		d.logger.InfoF(format, args...)
	} else {
		stdlog.Printf(format, args...)
	}
}

func (d *dbImpl) logError(format string, args ...interface{}) {
	if d.logger != nil {
		d.logger.ErrorF(format, args...)
	} else {
		stdlog.Printf(format, args...)
	}
}

// ExecContext executes a statement within an existing transaction if present,
// otherwise against the default connection.
func (d *dbImpl) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	tx := utils.GetTxCxt(ctx)
	if tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}

	return d.GetDB().ExecContext(ctx, query, args...)
}

// WithTransaction executes fn within a DB transaction.
func (d *dbImpl) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) (txErr error) {
	txCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tx, err := d.GetDB().BeginTx(txCtx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				d.logError("rollback after panic failed: %v", rbErr)
			}
			txErr = fmt.Errorf("panic in transaction: %v", p)
		}
	}()

	if err = fn(utils.SetTxCtx(txCtx, tx)); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return errors.Join(fmt.Errorf("rollback failed: %w", rbErr), fmt.Errorf("original error: %w", err))
		}
		return err
	}

	return tx.Commit()
}

// BeginTx starts a transaction. Placeholder for advanced transaction handling.
func (d *dbImpl) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.GetDB().BeginTx(ctx, opts)
}

// Close closes the database connection if open.
func (d *dbImpl) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// Ping verifies the database connection is alive.
func (d *dbImpl) Ping() error {
	if d.db != nil {
		return d.db.Ping()
	}
	return fmt.Errorf("database connection is not initialized")
}
