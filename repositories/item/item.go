package item

import (
	"context"
	"database/sql"
	"errors"

	"github.com/doug-martin/goqu/v9"

	"project-template/infrastructure/db"
	models "project-template/infrastructure/dto/item"
	"project-template/pkg/code"
	"project-template/pkg/logger"
)

// Repository defines CRUD operations for Items.
type Repository interface {
	Create(ctx context.Context, item *models.Item) error
	List(ctx context.Context) ([]models.Item, error)
	Get(ctx context.Context, id int) (*models.Item, error)
	Update(ctx context.Context, item *models.Item) error
	Delete(ctx context.Context, id int) error
}

type itemRepository struct {
	db  db.DB
	log logger.Logger
}

// NewItemRepository returns an Item repository backed by a SQL database.
func NewItemRepository(log logger.Logger, db db.DB) Repository {
	return &itemRepository{db: db, log: log}
}

// Create inserts a new item into the database and sets the generated ID on the struct.
func (r *itemRepository) Create(ctx context.Context, item *models.Item) error {
	dialect := r.db.GetGoquDialect()
	insert := dialect.Insert("items").Rows(goqu.Record{"name": item.Name})
	sqlStr, args, err := insert.ToSQL()
	if err != nil {
		r.log.ErrorF("Failed to build insert SQL: %v", err)
		return err
	}
	// Run the insert inside its own transaction for consistency.
	var id int64
	err = r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		res, execErr := r.db.ExecContext(txCtx, sqlStr, args...)
		if execErr != nil {
			r.log.ErrorF("Failed to execute insert: %v", execErr)
			return execErr
		}
		lastID, idErr := res.LastInsertId()
		if idErr != nil {
			r.log.ErrorF("Failed to get last insert ID: %v", idErr)
			return idErr
		}
		id = lastID
		return nil
	})
	if err != nil {
		return err
	}
	item.ID = int(id)
	return nil
}

// List returns all items from the database.
func (r *itemRepository) List(ctx context.Context) ([]models.Item, error) {
	dialect := r.db.GetGoquDialect()
	sqlStr, args, err := dialect.From("items").Select("id", "name").ToSQL()
	if err != nil {
		r.log.ErrorF("Failed to build select SQL: %v", err)
		return nil, err
	}
	rows, err := r.db.GetDB().QueryContext(ctx, sqlStr, args...)
	if err != nil {
		r.log.ErrorF("Failed to query rows: %v", err)
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			r.log.WarnF("Failed to close rows: %v", err)
		}
	}(rows)

	var items []models.Item
	for rows.Next() {
		var itm models.Item
		if err := rows.Scan(&itm.ID, &itm.Name); err != nil {
			r.log.ErrorF("Failed to scan row: %v", err)
			return nil, err
		}
		items = append(items, itm)
	}
	if err := rows.Err(); err != nil {
		r.log.ErrorF("Row iteration error: %v", err)
		return nil, err
	}
	return items, nil
}

// Get retrieves an item by its ID. It returns an error when not found.
func (r *itemRepository) Get(ctx context.Context, id int) (*models.Item, error) {
	dialect := r.db.GetGoquDialect()
	sqlStr, args, err := dialect.From("items").Select("id", "name").Where(goqu.C("id").Eq(id)).ToSQL()
	if err != nil {
		r.log.ErrorF("Failed to build select by ID SQL: %v", err)
		return nil, err
	}
	row := r.db.GetDB().QueryRowContext(ctx, sqlStr, args...)
	var item models.Item
	if err := row.Scan(&item.ID, &item.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.log.ErrorF("Item not found: %v", err)
			return nil, code.ErrItemNotFound
		}
		r.log.ErrorF("Failed to scan row: %v", err)
		return nil, err
	}
	return &item, nil
}

// Update modifies an existing item identified by its ID.
func (r *itemRepository) Update(ctx context.Context, item *models.Item) error {
	dialect := r.db.GetGoquDialect()
	update := dialect.Update("items").Set(goqu.Record{"name": item.Name}).Where(goqu.C("id").Eq(item.ID))
	sqlStr, args, err := update.ToSQL()
	if err != nil {
		r.log.ErrorF("Failed to build update SQL: %v", err)
		return err
	}
	// Ensure the update is executed within a transaction.
	err = r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, execErr := r.db.ExecContext(txCtx, sqlStr, args...)
		if execErr != nil {
			r.log.ErrorF("Failed to execute update: %v", execErr)
		}
		return execErr
	})
	return err
}

// Delete removes an item by ID from the database.
func (r *itemRepository) Delete(ctx context.Context, id int) error {
	dialect := r.db.GetGoquDialect()
	del := dialect.Delete("items").Where(goqu.C("id").Eq(id))
	sqlStr, args, err := del.ToSQL()
	if err != nil {
		r.log.ErrorF("Failed to build delete SQL: %v", err)
		return err
	}
	// Execute the delete statement inside a transaction.
	err = r.db.WithTransaction(ctx, func(txCtx context.Context) error {
		_, execErr := r.db.ExecContext(txCtx, sqlStr, args...)
		if execErr != nil {
			r.log.ErrorF("Failed to execute delete: %v", execErr)
		}
		return execErr
	})
	return err
}
