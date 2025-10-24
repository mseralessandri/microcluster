package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/canonical/lxd/shared/api"

	"github.com/canonical/microcluster/v3/cluster"
	"github.com/canonical/microcluster/v3/internal/db/query"
)

var _ = api.ServerEnvironment{}

var extendedTableObjects = cluster.RegisterStmt(`
SELECT extended_table.id, extended_table.key, extended_table.value
  FROM extended_table
  ORDER BY extended_table.key
`)

var extendedTableObjectsByKey = cluster.RegisterStmt(`
SELECT extended_table.id, extended_table.key, extended_table.value
  FROM extended_table
  WHERE ( extended_table.key = ? )
  ORDER BY extended_table.key
`)

var extendedTableID = cluster.RegisterStmt(`
SELECT extended_table.id FROM extended_table
  WHERE extended_table.key = ?
`)

var extendedTableCreate = cluster.RegisterStmt(`
INSERT INTO extended_table (key, value)
  VALUES (?, ?)
`)

var extendedTableDeleteByKey = cluster.RegisterStmt(`
DELETE FROM extended_table WHERE key = ?
`)

var extendedTableUpdate = cluster.RegisterStmt(`
UPDATE extended_table
  SET key = ?, value = ?
 WHERE id = ?
`)

// GetExtendedTables returns all available extended_tables.
// generator: extended_table GetMany
func GetExtendedTables(ctx context.Context, tx *sql.Tx, filters ...ExtendedTableFilter) ([]ExtendedTable, error) {
	var err error

	// Result slice.
	objects := make([]ExtendedTable, 0)

	// Pick the prepared statement and arguments to use based on active criteria.
	var sqlStmt *sql.Stmt
	args := []any{}
	queryParts := [2]string{}

	if len(filters) == 0 {
		sqlStmt, err = cluster.Stmt(tx, extendedTableObjects)
		if err != nil {
			return nil, fmt.Errorf("Failed to get \"extendedTableObjects\" prepared statement: %w", err)
		}
	}

	for i, filter := range filters {
		if filter.Key != nil {
			args = append(args, []any{filter.Key}...)
			if len(filters) == 1 {
				sqlStmt, err = cluster.Stmt(tx, extendedTableObjectsByKey)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"extendedTableObjectsByKey\" prepared statement: %w", err)
				}

				break
			}

			query, err := cluster.StmtString(extendedTableObjectsByKey)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"extendedTableObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.Key == nil {
			return nil, fmt.Errorf("Cannot filter on empty ExtendedTableFilter")
		} else {
			return nil, fmt.Errorf("No statement exists for the given Filter")
		}
	}

	// Dest function for scanning a row.
	dest := func(scan func(dest ...any) error) error {
		e := ExtendedTable{}
		err := scan(&e.ID, &e.Key, &e.Value)
		if err != nil {
			return err
		}

		objects = append(objects, e)

		return nil
	}

	// Select.
	if sqlStmt != nil {
		err = query.SelectObjects(ctx, sqlStmt, dest, args...)
	} else {
		queryStr := strings.Join(queryParts[:], "ORDER BY")
		err = query.Scan(ctx, tx, queryStr, dest, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"extended_table\" table: %w", err)
	}

	return objects, nil
}

// GetExtendedTable returns the extended_table with the given key.
// generator: extended_table GetOne
func GetExtendedTable(ctx context.Context, tx *sql.Tx, key string) (*ExtendedTable, error) {
	filter := ExtendedTableFilter{}
	filter.Key = &key

	objects, err := GetExtendedTables(ctx, tx, filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"extended_table\" table: %w", err)
	}

	switch len(objects) {
	case 0:
		return nil, api.StatusErrorf(http.StatusNotFound, "ExtendedTable not found")
	case 1:
		return &objects[0], nil
	default:
		return nil, fmt.Errorf("More than one \"extended_table\" entry matches")
	}
}

// GetExtendedTableID return the ID of the extended_table with the given key.
// generator: extended_table ID
func GetExtendedTableID(ctx context.Context, tx *sql.Tx, key string) (int64, error) {
	stmt, err := cluster.Stmt(tx, extendedTableID)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"extendedTableID\" prepared statement: %w", err)
	}

	row := stmt.QueryRowContext(ctx, key)
	var id int64
	err = row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return -1, api.StatusErrorf(http.StatusNotFound, "ExtendedTable not found")
	}

	if err != nil {
		return -1, fmt.Errorf("Failed to get \"extended_table\" ID: %w", err)
	}

	return id, nil
}

// ExtendedTableExists checks if a extended_table with the given key exists.
// generator: extended_table Exists
func ExtendedTableExists(ctx context.Context, tx *sql.Tx, key string) (bool, error) {
	_, err := GetExtendedTableID(ctx, tx, key)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// CreateExtendedTable adds a new extended_table to the database.
// generator: extended_table Create
func CreateExtendedTable(ctx context.Context, tx *sql.Tx, object ExtendedTable) (int64, error) {
	// Check if a extended_table with the same key exists.
	exists, err := ExtendedTableExists(ctx, tx, object.Key)
	if err != nil {
		return -1, fmt.Errorf("Failed to check for duplicates: %w", err)
	}

	if exists {
		return -1, api.StatusErrorf(http.StatusConflict, "This \"extended_table\" entry already exists")
	}

	args := make([]any, 2)

	// Populate the statement arguments.
	args[0] = object.Key
	args[1] = object.Value

	// Prepared statement to use.
	stmt, err := cluster.Stmt(tx, extendedTableCreate)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"extendedTableCreate\" prepared statement: %w", err)
	}

	// Execute the statement.
	result, err := stmt.Exec(args...)
	if err != nil {
		return -1, fmt.Errorf("Failed to create \"extended_table\" entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Failed to fetch \"extended_table\" entry ID: %w", err)
	}

	return id, nil
}

// DeleteExtendedTable deletes the extended_table matching the given key parameters.
// generator: extended_table DeleteOne-by-Key
func DeleteExtendedTable(ctx context.Context, tx *sql.Tx, key string) error {
	stmt, err := cluster.Stmt(tx, extendedTableDeleteByKey)
	if err != nil {
		return fmt.Errorf("Failed to get \"extendedTableDeleteByKey\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(key)
	if err != nil {
		return fmt.Errorf("Delete \"extended_table\": %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n == 0 {
		return api.StatusErrorf(http.StatusNotFound, "ExtendedTable not found")
	} else if n > 1 {
		return fmt.Errorf("Query deleted %d ExtendedTable rows instead of 1", n)
	}

	return nil
}

// UpdateExtendedTable updates the extended_table matching the given key parameters.
// generator: extended_table Update
func UpdateExtendedTable(ctx context.Context, tx *sql.Tx, key string, object ExtendedTable) error {
	id, err := GetExtendedTableID(ctx, tx, key)
	if err != nil {
		return err
	}

	stmt, err := cluster.Stmt(tx, extendedTableUpdate)
	if err != nil {
		return fmt.Errorf("Failed to get \"extendedTableUpdate\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(object.Key, object.Value, id)
	if err != nil {
		return fmt.Errorf("Update \"extended_table\" entry failed: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n != 1 {
		return fmt.Errorf("Query updated %d rows instead of 1", n)
	}

	return nil
}
