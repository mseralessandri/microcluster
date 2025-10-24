package cluster

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/canonical/lxd/shared/api"

	"github.com/canonical/microcluster/v3/internal/db/query"
)

var _ = api.ServerEnvironment{}

var coreTokenRecordObjects = RegisterStmt(`
SELECT core_token_records.id, core_token_records.secret, core_token_records.name, core_token_records.expiry_date
  FROM core_token_records
  ORDER BY core_token_records.secret
`)

var coreTokenRecordObjectsBySecret = RegisterStmt(`
SELECT core_token_records.id, core_token_records.secret, core_token_records.name, core_token_records.expiry_date
  FROM core_token_records
  WHERE ( core_token_records.secret = ? )
  ORDER BY core_token_records.secret
`)

var coreTokenRecordID = RegisterStmt(`
SELECT core_token_records.id FROM core_token_records
  WHERE core_token_records.secret = ?
`)

var coreTokenRecordCreate = RegisterStmt(`
INSERT INTO core_token_records (secret, name, expiry_date)
  VALUES (?, ?, ?)
`)

var coreTokenRecordDeleteByName = RegisterStmt(`
DELETE FROM core_token_records WHERE name = ?
`)

// GetCoreTokenRecordID return the ID of the core_token_record with the given key.
// generator: core_token_record ID
func GetCoreTokenRecordID(ctx context.Context, tx *sql.Tx, secret string) (int64, error) {
	stmt, err := Stmt(tx, coreTokenRecordID)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"coreTokenRecordID\" prepared statement: %w", err)
	}

	row := stmt.QueryRowContext(ctx, secret)
	var id int64
	err = row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return -1, api.StatusErrorf(http.StatusNotFound, "CoreTokenRecord not found")
	}

	if err != nil {
		return -1, fmt.Errorf("Failed to get \"core_token_records\" ID: %w", err)
	}

	return id, nil
}

// CoreTokenRecordExists checks if a core_token_record with the given key exists.
// generator: core_token_record Exists
func CoreTokenRecordExists(ctx context.Context, tx *sql.Tx, secret string) (bool, error) {
	_, err := GetCoreTokenRecordID(ctx, tx, secret)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// GetCoreTokenRecord returns the core_token_record with the given key.
// generator: core_token_record GetOne
func GetCoreTokenRecord(ctx context.Context, tx *sql.Tx, secret string) (*CoreTokenRecord, error) {
	filter := CoreTokenRecordFilter{}
	filter.Secret = &secret

	objects, err := GetCoreTokenRecords(ctx, tx, filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_token_records\" table: %w", err)
	}

	switch len(objects) {
	case 0:
		return nil, api.StatusErrorf(http.StatusNotFound, "CoreTokenRecord not found")
	case 1:
		return &objects[0], nil
	default:
		return nil, fmt.Errorf("More than one \"core_token_records\" entry matches")
	}
}

// getCoreTokenRecords can be used to run handwritten sql.Stmts to return a slice of objects.
func getCoreTokenRecords(ctx context.Context, stmt *sql.Stmt, args ...any) ([]CoreTokenRecord, error) {
	objects := make([]CoreTokenRecord, 0)

	dest := func(scan func(dest ...any) error) error {
		c := CoreTokenRecord{}
		err := scan(&c.ID, &c.Secret, &c.Name, &c.ExpiryDate)
		if err != nil {
			return err
		}

		objects = append(objects, c)

		return nil
	}

	err := query.SelectObjects(ctx, stmt, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_token_records\" table: %w", err)
	}

	return objects, nil
}

// getCoreTokenRecordsRaw can be used to run handwritten query strings to return a slice of objects.
func getCoreTokenRecordsRaw(ctx context.Context, tx *sql.Tx, sql string, args ...any) ([]CoreTokenRecord, error) {
	objects := make([]CoreTokenRecord, 0)

	dest := func(scan func(dest ...any) error) error {
		c := CoreTokenRecord{}
		err := scan(&c.ID, &c.Secret, &c.Name, &c.ExpiryDate)
		if err != nil {
			return err
		}

		objects = append(objects, c)

		return nil
	}

	err := query.Scan(ctx, tx, sql, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_token_records\" table: %w", err)
	}

	return objects, nil
}

// GetCoreTokenRecords returns all available core_token_records.
// generator: core_token_record GetMany
func GetCoreTokenRecords(ctx context.Context, tx *sql.Tx, filters ...CoreTokenRecordFilter) ([]CoreTokenRecord, error) {
	var err error

	// Result slice.
	var objects []CoreTokenRecord

	// Pick the prepared statement and arguments to use based on active criteria.
	var sqlStmt *sql.Stmt
	args := []any{}
	queryParts := [2]string{}

	if len(filters) == 0 {
		sqlStmt, err = Stmt(tx, coreTokenRecordObjects)
		if err != nil {
			return nil, fmt.Errorf("Failed to get \"coreTokenRecordObjects\" prepared statement: %w", err)
		}
	}

	for i, filter := range filters {
		if filter.Secret != nil && filter.ID == nil && filter.Name == nil {
			args = append(args, []any{filter.Secret}...)
			if len(filters) == 1 {
				sqlStmt, err = Stmt(tx, coreTokenRecordObjectsBySecret)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"coreTokenRecordObjectsBySecret\" prepared statement: %w", err)
				}

				break
			}

			query, err := StmtString(coreTokenRecordObjectsBySecret)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"coreTokenRecordObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.ID == nil && filter.Secret == nil && filter.Name == nil {
			return nil, fmt.Errorf("Cannot filter on empty CoreTokenRecordFilter")
		} else {
			return nil, fmt.Errorf("No statement exists for the given Filter")
		}
	}

	// Select.
	if sqlStmt != nil {
		objects, err = getCoreTokenRecords(ctx, sqlStmt, args...)
	} else {
		queryStr := strings.Join(queryParts[:], "ORDER BY")
		objects, err = getCoreTokenRecordsRaw(ctx, tx, queryStr, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_token_records\" table: %w", err)
	}

	return objects, nil
}

// CreateCoreTokenRecord adds a new core_token_record to the database.
// generator: core_token_record Create
func CreateCoreTokenRecord(ctx context.Context, tx *sql.Tx, object CoreTokenRecord) (int64, error) {
	// Check if a core_token_record with the same key exists.
	exists, err := CoreTokenRecordExists(ctx, tx, object.Secret)
	if err != nil {
		return -1, fmt.Errorf("Failed to check for duplicates: %w", err)
	}

	if exists {
		return -1, api.StatusErrorf(http.StatusConflict, "This \"core_token_records\" entry already exists")
	}

	args := make([]any, 3)

	// Populate the statement arguments.
	args[0] = object.Secret
	args[1] = object.Name
	args[2] = object.ExpiryDate

	// Prepared statement to use.
	stmt, err := Stmt(tx, coreTokenRecordCreate)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"coreTokenRecordCreate\" prepared statement: %w", err)
	}

	// Execute the statement.
	result, err := stmt.Exec(args...)
	if err != nil {
		return -1, fmt.Errorf("Failed to create \"core_token_records\" entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Failed to fetch \"core_token_records\" entry ID: %w", err)
	}

	return id, nil
}

// DeleteCoreTokenRecord deletes the core_token_record matching the given key parameters.
// generator: core_token_record DeleteOne-by-Name
func DeleteCoreTokenRecord(ctx context.Context, tx *sql.Tx, name string) error {
	stmt, err := Stmt(tx, coreTokenRecordDeleteByName)
	if err != nil {
		return fmt.Errorf("Failed to get \"coreTokenRecordDeleteByName\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(name)
	if err != nil {
		return fmt.Errorf("Delete \"core_token_records\": %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n == 0 {
		return api.StatusErrorf(http.StatusNotFound, "CoreTokenRecord not found")
	} else if n > 1 {
		return fmt.Errorf("Query deleted %d CoreTokenRecord rows instead of 1", n)
	}

	return nil
}
