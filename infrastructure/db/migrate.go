package db

import (
	"errors"
	"fmt"

	"project-template/infrastructure/config"
	"project-template/infrastructure/enums"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies database migrations located at path using the
// application's existing database connection. The path should use the file:// scheme.
func RunMigrations(path string) error {
	driver, err := mysql.WithInstance(GetDBInstance().GetDB(), &mysql.Config{})
	if err != nil {
		return fmt.Errorf("driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(path, "mysql", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	strategy := config.Cfg.Database.DirtyStrategy
	if strategy == "" {
		strategy = enums.DirtyStrategyRetry
	}
	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			// clear dirty flag so further operations can proceed
			if forceErr := m.Force(int(dirty.Version)); forceErr != nil {
				return fmt.Errorf("force dirty: %w", forceErr)
			}
			if strategy == enums.DirtyStrategyRetry {
				if stepErr := m.Steps(-1); stepErr != nil {
					return fmt.Errorf("rollback dirty: %w", stepErr)
				}
			}
			if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return fmt.Errorf("migrate up after force: %w", err)
			}
			return nil
		}
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
