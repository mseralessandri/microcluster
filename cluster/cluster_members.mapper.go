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

var coreClusterMemberObjects = RegisterStmt(`
SELECT core_cluster_members.id, core_cluster_members.name, core_cluster_members.address, core_cluster_members.certificate, core_cluster_members.schema_internal, core_cluster_members.schema_external, core_cluster_members.api_extensions, core_cluster_members.heartbeat, core_cluster_members.role
  FROM core_cluster_members
  ORDER BY core_cluster_members.name
`)

var coreClusterMemberObjectsByAddress = RegisterStmt(`
SELECT core_cluster_members.id, core_cluster_members.name, core_cluster_members.address, core_cluster_members.certificate, core_cluster_members.schema_internal, core_cluster_members.schema_external, core_cluster_members.api_extensions, core_cluster_members.heartbeat, core_cluster_members.role
  FROM core_cluster_members
  WHERE ( core_cluster_members.address = ? )
  ORDER BY core_cluster_members.name
`)

var coreClusterMemberObjectsByName = RegisterStmt(`
SELECT core_cluster_members.id, core_cluster_members.name, core_cluster_members.address, core_cluster_members.certificate, core_cluster_members.schema_internal, core_cluster_members.schema_external, core_cluster_members.api_extensions, core_cluster_members.heartbeat, core_cluster_members.role
  FROM core_cluster_members
  WHERE ( core_cluster_members.name = ? )
  ORDER BY core_cluster_members.name
`)

var coreClusterMemberID = RegisterStmt(`
SELECT core_cluster_members.id FROM core_cluster_members
  WHERE core_cluster_members.name = ?
`)

var coreClusterMemberCreate = RegisterStmt(`
INSERT INTO core_cluster_members (name, address, certificate, schema_internal, schema_external, api_extensions, heartbeat, role)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`)

var coreClusterMemberDeleteByAddress = RegisterStmt(`
DELETE FROM core_cluster_members WHERE address = ?
`)

var coreClusterMemberUpdate = RegisterStmt(`
UPDATE core_cluster_members
  SET name = ?, address = ?, certificate = ?, schema_internal = ?, schema_external = ?, api_extensions = ?, heartbeat = ?, role = ?
 WHERE id = ?
`)

// getCoreClusterMembers can be used to run handwritten sql.Stmts to return a slice of objects.
func getCoreClusterMembers(ctx context.Context, stmt *sql.Stmt, args ...any) ([]CoreClusterMember, error) {
	objects := make([]CoreClusterMember, 0)

	dest := func(scan func(dest ...any) error) error {
		c := CoreClusterMember{}
		err := scan(&c.ID, &c.Name, &c.Address, &c.Certificate, &c.SchemaInternal, &c.SchemaExternal, &c.APIExtensions, &c.Heartbeat, &c.Role)
		if err != nil {
			return err
		}

		objects = append(objects, c)

		return nil
	}

	err := query.SelectObjects(ctx, stmt, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_cluster_members\" table: %w", err)
	}

	return objects, nil
}

// getCoreClusterMembersRaw can be used to run handwritten query strings to return a slice of objects.
func getCoreClusterMembersRaw(ctx context.Context, tx *sql.Tx, sql string, args ...any) ([]CoreClusterMember, error) {
	objects := make([]CoreClusterMember, 0)

	dest := func(scan func(dest ...any) error) error {
		c := CoreClusterMember{}
		err := scan(&c.ID, &c.Name, &c.Address, &c.Certificate, &c.SchemaInternal, &c.SchemaExternal, &c.APIExtensions, &c.Heartbeat, &c.Role)
		if err != nil {
			return err
		}

		objects = append(objects, c)

		return nil
	}

	err := query.Scan(ctx, tx, sql, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_cluster_members\" table: %w", err)
	}

	return objects, nil
}

// GetCoreClusterMembers returns all available core_cluster_members.
// generator: core_cluster_member GetMany
func GetCoreClusterMembers(ctx context.Context, tx *sql.Tx, filters ...CoreClusterMemberFilter) ([]CoreClusterMember, error) {
	var err error

	// Result slice.
	var objects []CoreClusterMember

	// Pick the prepared statement and arguments to use based on active criteria.
	var sqlStmt *sql.Stmt
	args := []any{}
	queryParts := [2]string{}

	if len(filters) == 0 {
		sqlStmt, err = Stmt(tx, coreClusterMemberObjects)
		if err != nil {
			return nil, fmt.Errorf("Failed to get \"coreClusterMemberObjects\" prepared statement: %w", err)
		}
	}

	for i, filter := range filters {
		if filter.Name != nil && filter.Address == nil {
			args = append(args, []any{filter.Name}...)
			if len(filters) == 1 {
				sqlStmt, err = Stmt(tx, coreClusterMemberObjectsByName)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"coreClusterMemberObjectsByName\" prepared statement: %w", err)
				}

				break
			}

			query, err := StmtString(coreClusterMemberObjectsByName)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"coreClusterMemberObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.Address != nil && filter.Name == nil {
			args = append(args, []any{filter.Address}...)
			if len(filters) == 1 {
				sqlStmt, err = Stmt(tx, coreClusterMemberObjectsByAddress)
				if err != nil {
					return nil, fmt.Errorf("Failed to get \"coreClusterMemberObjectsByAddress\" prepared statement: %w", err)
				}

				break
			}

			query, err := StmtString(coreClusterMemberObjectsByAddress)
			if err != nil {
				return nil, fmt.Errorf("Failed to get \"coreClusterMemberObjects\" prepared statement: %w", err)
			}

			parts := strings.SplitN(query, "ORDER BY", 2)
			if i == 0 {
				copy(queryParts[:], parts)
				continue
			}

			_, where, _ := strings.Cut(parts[0], "WHERE")
			queryParts[0] += "OR" + where
		} else if filter.Address == nil && filter.Name == nil {
			return nil, fmt.Errorf("Cannot filter on empty CoreClusterMemberFilter")
		} else {
			return nil, fmt.Errorf("No statement exists for the given Filter")
		}
	}

	// Select.
	if sqlStmt != nil {
		objects, err = getCoreClusterMembers(ctx, sqlStmt, args...)
	} else {
		queryStr := strings.Join(queryParts[:], "ORDER BY")
		objects, err = getCoreClusterMembersRaw(ctx, tx, queryStr, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_cluster_members\" table: %w", err)
	}

	return objects, nil
}

// GetCoreClusterMember returns the core_cluster_member with the given key.
// generator: core_cluster_member GetOne
func GetCoreClusterMember(ctx context.Context, tx *sql.Tx, name string) (*CoreClusterMember, error) {
	filter := CoreClusterMemberFilter{}
	filter.Name = &name

	objects, err := GetCoreClusterMembers(ctx, tx, filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"core_cluster_members\" table: %w", err)
	}

	switch len(objects) {
	case 0:
		return nil, api.StatusErrorf(http.StatusNotFound, "CoreClusterMember not found")
	case 1:
		return &objects[0], nil
	default:
		return nil, fmt.Errorf("More than one \"core_cluster_members\" entry matches")
	}
}

// GetCoreClusterMemberID return the ID of the core_cluster_member with the given key.
// generator: core_cluster_member ID
func GetCoreClusterMemberID(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	stmt, err := Stmt(tx, coreClusterMemberID)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"coreClusterMemberID\" prepared statement: %w", err)
	}

	row := stmt.QueryRowContext(ctx, name)
	var id int64
	err = row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return -1, api.StatusErrorf(http.StatusNotFound, "CoreClusterMember not found")
	}

	if err != nil {
		return -1, fmt.Errorf("Failed to get \"core_cluster_members\" ID: %w", err)
	}

	return id, nil
}

// CoreClusterMemberExists checks if a core_cluster_member with the given key exists.
// generator: core_cluster_member Exists
func CoreClusterMemberExists(ctx context.Context, tx *sql.Tx, name string) (bool, error) {
	_, err := GetCoreClusterMemberID(ctx, tx, name)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// CreateCoreClusterMember adds a new core_cluster_member to the database.
// generator: core_cluster_member Create
func CreateCoreClusterMember(ctx context.Context, tx *sql.Tx, object CoreClusterMember) (int64, error) {
	// Check if a core_cluster_member with the same key exists.
	exists, err := CoreClusterMemberExists(ctx, tx, object.Name)
	if err != nil {
		return -1, fmt.Errorf("Failed to check for duplicates: %w", err)
	}

	if exists {
		return -1, api.StatusErrorf(http.StatusConflict, "This \"core_cluster_members\" entry already exists")
	}

	args := make([]any, 8)

	// Populate the statement arguments.
	args[0] = object.Name
	args[1] = object.Address
	args[2] = object.Certificate
	args[3] = object.SchemaInternal
	args[4] = object.SchemaExternal
	args[5] = object.APIExtensions
	args[6] = object.Heartbeat
	args[7] = object.Role

	// Prepared statement to use.
	stmt, err := Stmt(tx, coreClusterMemberCreate)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"coreClusterMemberCreate\" prepared statement: %w", err)
	}

	// Execute the statement.
	result, err := stmt.Exec(args...)
	if err != nil {
		return -1, fmt.Errorf("Failed to create \"core_cluster_members\" entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Failed to fetch \"core_cluster_members\" entry ID: %w", err)
	}

	return id, nil
}

// DeleteCoreClusterMember deletes the core_cluster_member matching the given key parameters.
// generator: core_cluster_member DeleteOne-by-Address
func DeleteCoreClusterMember(ctx context.Context, tx *sql.Tx, address string) error {
	stmt, err := Stmt(tx, coreClusterMemberDeleteByAddress)
	if err != nil {
		return fmt.Errorf("Failed to get \"coreClusterMemberDeleteByAddress\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(address)
	if err != nil {
		return fmt.Errorf("Delete \"core_cluster_members\": %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n == 0 {
		return api.StatusErrorf(http.StatusNotFound, "CoreClusterMember not found")
	} else if n > 1 {
		return fmt.Errorf("Query deleted %d CoreClusterMember rows instead of 1", n)
	}

	return nil
}

// UpdateCoreClusterMember updates the core_cluster_member matching the given key parameters.
// generator: core_cluster_member Update
func UpdateCoreClusterMember(ctx context.Context, tx *sql.Tx, name string, object CoreClusterMember) error {
	id, err := GetCoreClusterMemberID(ctx, tx, name)
	if err != nil {
		return err
	}

	stmt, err := Stmt(tx, coreClusterMemberUpdate)
	if err != nil {
		return fmt.Errorf("Failed to get \"coreClusterMemberUpdate\" prepared statement: %w", err)
	}

	result, err := stmt.Exec(object.Name, object.Address, object.Certificate, object.SchemaInternal, object.SchemaExternal, object.APIExtensions, object.Heartbeat, object.Role, id)
	if err != nil {
		return fmt.Errorf("Update \"core_cluster_members\" entry failed: %w", err)
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
