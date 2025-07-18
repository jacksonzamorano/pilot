// Package pilot_db provides a fluent, type-safe PostgreSQL query builder with comprehensive CRUD operations,
// transaction support, and seamless integration with pgx connection pools. This package offers an intuitive
// API for building complex SQL queries programmatically while maintaining type safety and preventing common
// SQL injection vulnerabilities through proper parameter binding.
//
// Key Features:
// - Fluent API for building SELECT, INSERT, UPDATE, and DELETE queries
// - Type-safe query construction with generics support
// - Automatic parameter binding to prevent SQL injection
// - Support for complex joins, subqueries, and aggregations
// - Transaction management with bulk operations
// - Comprehensive error handling with custom error types
// - Integration with pilot_http for automatic HTTP response generation
//
// The query builder supports all major SQL operations including:
// - Field selection with aliasing and expressions
// - WHERE clauses with multiple operators (=, <>, <, <=, >, >=, LIKE, IN, IS NULL, etc.)
// - INNER JOINs with custom aliases and conditions
// - ORDER BY clauses with multiple fields and directions
// - GROUP BY operations for aggregations
// - LIMIT clauses for result pagination
// - Bulk insert operations for improved performance
//
// Usage Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    Select("id").Select("name").Select("email").
//	    Where("active", "= $", true).
//	    SortDesc("created_at").
//	    Limit(10)
//
//	users, err := query.QueryMany(ctx, conn)
package pilot_db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	pilot_http "github.com/jacksonzamorano/pilot/pilot-http"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// NoRowError represents an error that occurs when a database operation expects to find data
// but no rows are returned. This is typically used internally by the query builder to
// distinguish between actual errors and simply empty result sets.
type NoRowError struct {
}

// Error implements the error interface for NoRowError, providing a human-readable
// error message indicating that no data was found during the database operation.
func (e NoRowError) Error() string {
	return "No data found"
}

// BeginTransaction starts a new PostgreSQL transaction using the provided connection pool.
// This function creates a transaction context that can be used for multiple related database
// operations that need to be executed atomically. If the transaction cannot be started,
// the function will panic with the underlying error.
//
// Parameters:
//   - conn: A connection from the pgx connection pool
//
// Returns:
//   - *pgx.Tx: A pointer to the transaction object that can be used for subsequent operations
//
// Usage:
//
//	tx := pilot_db.BeginTransaction(conn)
//	defer pilot_db.EndTransaction(*tx) // Remember to commit or rollback
func BeginTransaction(conn *pgxpool.Conn) *pgx.Tx {
	tx, err := conn.Begin(context.Background())
	if err != nil {
		panic(err)
	}
	return &tx
}

// EndTransaction commits a PostgreSQL transaction, making all changes within the transaction
// permanent. This function should be called after all operations within a transaction are
// complete and successful. If any operation within the transaction failed, you should call
// tx.Rollback() instead to undo all changes.
//
// Parameters:
//   - tx: The transaction to commit
//
// Returns:
//   - error: Any error that occurred during the commit operation
//
// Usage:
//
//	err := pilot_db.EndTransaction(tx)
//	if err != nil {
//	    log.Printf("Failed to commit transaction: %v", err)
//	}
func EndTransaction(tx pgx.Tx) error {
	return tx.Commit(context.Background())
}

// FromTableFn is a generic function type that defines how to convert a database row
// into a Go struct or type. This function is used by the query builder to automatically
// parse query results into the desired type T. The function should read the appropriate
// columns from the pgx.Rows object and populate a struct of type T.
//
// Type Parameters:
//   - T: The target type to convert database rows into
//
// Parameters:
//   - row: A pgx.Rows object positioned at the current row to be converted
//
// Returns:
//   - *T: A pointer to the converted object of type T
//   - error: Any error that occurred during the conversion process
//
// Example:
//
//	func userFromRow(row pgx.Rows) (*User, error) {
//	    var user User
//	    err := row.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
//	    return &user, err
//	}
type FromTableFn[T any] func(row pgx.Rows, val *T) error

type KeyValue interface {
	~string
}

type WhereComponent interface {
	Build(query *string, args *[]any)
}

type ConditionWhereComponent struct {
	value string
}

func (j ConditionWhereComponent) Build(query *string, args *[]any) {
	*query += j.value
}

type StringWhereComponent struct {
	value string
}

func (s StringWhereComponent) Build(query *string, args *[]any) {
	*query += s.value
}

// QueryBuilder is the core struct that represents a SQL query being constructed.
// It uses a fluent API pattern where methods can be chained together to build
// complex queries. The builder is generic over type T, which represents the target
// type that query results will be converted into.
//
// The QueryBuilder supports all major SQL operations:
// - SELECT queries with field selection, joins, filtering, sorting, and pagination
// - INSERT queries with single or bulk record insertion
// - UPDATE queries with conditional updates and field modifications
// - DELETE queries with conditional record removal
//
// Type Parameters:
//   - T: The target type for query results (e.g., User, Product, etc.)
//
// Fields:
//   - operation: The SQL operation type ("SELECT", "INSERT", "UPDATE", "DELETE")
//   - fields: List of fields to select in the query
//   - from: The primary table name for the query
//   - joins: List of JOIN clauses to include in the query
//   - lastJoin: Reference to the most recently added join for context switching
//   - where: List of WHERE conditions to filter results
//   - set: List of field assignments for INSERT/UPDATE operations
//   - joinsByName: Map of join aliases to join objects for easy reference
//   - conversion: Function to convert database rows to type T
//   - warn: Whether to show warnings for potentially dangerous operations
//   - sort: List of ORDER BY clauses for result sorting
//   - limit: Maximum number of results to return (-1 for no limit)
//   - groupBy: Optional GROUP BY clause for aggregations
//
// Example:
//
//	// Create a new query builder for User objects
//	query := pilot_db.Select("users", userFromRow).
//	    Select("id").Select("name").Select("email").
//	    Where("active", "= $", true).
//	    SortDesc("created_at").
//	    Limit(10)
type QueryBuilder[T any, ReadKeys KeyValue, WriteKeys KeyValue, SortKeys KeyValue] struct {
	ctx         context.Context
	db          *pgxpool.Conn
	operation   string
	fields      []SelectField
	from        string
	joins       []QueryJoin
	lastJoin    *QueryJoin
	where       []WhereComponent
	set         []map[string]SetField
	joinsByName map[string]*QueryJoin
	conversion  FromTableFn[T]
	warn        bool
	sort        []QuerySort
	sortArgs    []any
	limit       int
	groupBy     *string
	debug       bool
}

// Select creates a new QueryBuilder configured for SELECT operations on the specified table.
// This is the primary constructor for building queries that retrieve data from the database.
// The returned QueryBuilder can be further configured with field selections, joins, where
// clauses, sorting, and pagination using the fluent API.
//
// Type Parameters:
//   - T: The target type that query results will be converted into
//
// Parameters:
//   - table: The name of the primary table to select from
//   - conversion: A function that converts database rows into objects of type T
//
// Returns:
//   - *QueryBuilder[T]: A new QueryBuilder configured for SELECT operations
//
// Example:
//
//	// Create a SELECT query for users
//	query := pilot_db.Select("users", userFromRow).
//	    Select("id").Select("name").Select("email").
//	    Where("active", "= $", true)
//
//	users, err := query.QueryMany(ctx, conn)
func Select[T any](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, string, string, string] {
	return &QueryBuilder[T, string, string, string]{
		ctx:         ctx,
		db:          db,
		operation:   "SELECT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}

func SelectKeyed[T any, RK KeyValue, WK KeyValue, SK KeyValue](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, RK, WK, SK] {
	return &QueryBuilder[T, RK, WK, SK]{
		ctx:         ctx,
		db:          db,
		operation:   "SELECT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}

// Update creates a new QueryBuilder configured for UPDATE operations on the specified table.
// This constructor is used to build queries that modify existing records in the database.
// You must use the Set() method to specify which fields to update before executing the query.
//
// Type Parameters:
//   - T: The target type that query results will be converted into (for RETURNING clauses)
//
// Parameters:
//   - table: The name of the table to update records in
//   - conversion: A function that converts database rows into objects of type T
//
// Returns:
//   - *QueryBuilder[T]: A new QueryBuilder configured for UPDATE operations
//
// Example:
//
//	// Create an UPDATE query to modify user records
//	query := pilot_db.Update("users", userFromRow).
//	    Set("name", "John Doe").
//	    Set("updated_at", time.Now()).
//	    Where("id", "= $", 123)
//
//	err := query.QueryInTransaction(ctx, tx)
func Update[T any](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, string, string, string] {
	return &QueryBuilder[T, string, string, string]{
		ctx:         ctx,
		db:          db,
		operation:   "UPDATE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func UpdateKeyed[T any, RK KeyValue, WK KeyValue, SK KeyValue](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, RK, WK, SK] {
	return &QueryBuilder[T, RK, WK, SK]{
		ctx:         ctx,
		db:          db,
		operation:   "UPDATE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}

// Insert creates a new QueryBuilder configured for INSERT operations on the specified table.
// This constructor is used to build queries that add new records to the database. You can
// insert single records using Set() calls, or perform bulk inserts by calling Set() multiple
// times with the same field names to create multiple rows.
//
// Type Parameters:
//   - T: The target type that query results will be converted into (for RETURNING clauses)
//
// Parameters:
//   - table: The name of the table to insert records into
//   - conversion: A function that converts database rows into objects of type T
//
// Returns:
//   - *QueryBuilder[T]: A new QueryBuilder configured for INSERT operations
//
// Example:
//
//	// Create an INSERT query for a new user
//	query := pilot_db.Insert("users", userFromRow).
//	    Set("name", "Jane Doe").
//	    Set("email", "jane@example.com").
//	    Set("created_at", time.Now())
//
//	newUser, err := query.QueryOneExpect(ctx, conn)
//
//	// Bulk insert example
//	bulkQuery := pilot_db.Insert("users", userFromRow).
//	    Set("name", "User 1").Set("email", "user1@example.com").  // First row
//	    Set("name", "User 2").Set("email", "user2@example.com")   // Second row
func Insert[T any](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, string, string, string] {
	return &QueryBuilder[T, string, string, string]{
		ctx:         ctx,
		db:          db,
		operation:   "INSERT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func InsertKeyed[T any, RK KeyValue, WK KeyValue, SK KeyValue](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, RK, WK, SK] {
	return &QueryBuilder[T, RK, WK, SK]{
		ctx:         ctx,
		db:          db,
		operation:   "INSERT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}

// Delete creates a new QueryBuilder configured for DELETE operations on the specified table.
// This constructor is used to build queries that remove records from the database. For safety,
// DELETE queries require at least one WHERE clause unless you explicitly call Force() to
// override this protection against accidental mass deletions.
//
// Type Parameters:
//   - T: The target type that query results will be converted into (for RETURNING clauses)
//
// Parameters:
//   - table: The name of the table to delete records from
//   - conversion: A function that converts database rows into objects of type T
//
// Returns:
//   - *QueryBuilder[T]: A new QueryBuilder configured for DELETE operations
//
// Example:
//
//	// Create a DELETE query to remove inactive users
//	query := pilot_db.Delete("users", userFromRow).
//	    Where("active", "= $", false).
//	    Where("last_login", "< $", cutoffDate)
//
//	err := query.QueryInTransaction(ctx, tx)
//
//	// Force delete all records (dangerous!)
//	query := pilot_db.Delete("temp_data", tempDataFromRow).
//	    Force()  // Required to delete without WHERE clause
func Delete[T any](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, string, string, string] {
	return &QueryBuilder[T, string, string, string]{
		ctx:         ctx,
		db:          db,
		operation:   "DELETE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func DeleteKeyed[T any, RK KeyValue, WK KeyValue, SK KeyValue](table string, ctx context.Context, db *pgxpool.Conn, conversion FromTableFn[T]) *QueryBuilder[T, RK, WK, SK] {
	return &QueryBuilder[T, RK, WK, SK]{
		ctx:         ctx,
		db:          db,
		operation:   "DELETE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []WhereComponent{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}

func (qb *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Debug() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	qb.debug = true
	return qb
}

// Set assigns a value to a field for INSERT or UPDATE operations. This method uses parameter
// binding to safely include values in the query, preventing SQL injection attacks. For INSERT
// operations, calling Set multiple times with the same field name creates multiple rows for
// bulk insertion. For UPDATE operations, subsequent calls to Set with the same field will
// override the previous value.
//
// Parameters:
//   - field: The name of the database column to set
//   - value: The value to assign to the field (will be parameter-bound)
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	// Single INSERT
//	query := pilot_db.Insert("users", userFromRow).
//	    Set("name", "John Doe").
//	    Set("email", "john@example.com").
//	    Set("age", 30)
//
//	// Bulk INSERT (multiple rows)
//	query := pilot_db.Insert("users", userFromRow).
//	    Set("name", "John").Set("email", "john@example.com").     // Row 1
//	    Set("name", "Jane").Set("email", "jane@example.com")      // Row 2
//
//	// UPDATE operation
//	query := pilot_db.Update("users", userFromRow).
//	    Set("name", "Updated Name").
//	    Set("updated_at", time.Now()).
//	    Where("id", "= $", userId)
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Set(field WriteKeys, value any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if b.operation != "UPDATE" && b.operation != "INSERT" {
		log.Fatal("Attempted to set a field on a non-update/insert query. This is probably not what you want.")
	}
	set := SetField{string(field), nil, value}
	lastRecord := len(b.set) - 1
	_, fieldAtLast := b.set[lastRecord][string(field)]
	if !fieldAtLast {
		if lastRecord > 1 {
			_, inLast := b.set[lastRecord-1][string(field)]
			if !inLast {
				log.Fatal("Attempted to set a field in a bulk insert which wasn't in the previous row. Bulk inserts require the same arguments in every row")
			}
		}
		b.set[lastRecord][string(field)] = set
	} else {
		if b.operation != "INSERT" {
			log.Fatal("Added a duplicate key. This is used for bulk inserts but this operation is not an insert.")
		}
		newRow := map[string]SetField{}
		newRow[string(field)] = set
		b.set = append(b.set, newRow)
	}
	return b
}

// SetLiteral assigns a literal SQL expression to a field for UPDATE operations. Unlike Set(),
// this method does not use parameter binding and inserts the value directly into the SQL.
// This is useful for database functions, calculations, or other SQL expressions that need
// to be evaluated by the database engine. Use with caution as this bypasses SQL injection
// protection - only use with trusted input or database functions.
//
// Parameters:
//   - field: The name of the database column to set
//   - value: The literal SQL expression to assign (not parameter-bound)
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	// Using database functions
//	query := pilot_db.Update("users", userFromRow).
//	    SetLiteral("updated_at", "NOW()").
//	    SetLiteral("login_count", "login_count + 1").
//	    Where("id", "= $", userId)
//
//	// Using calculations
//	query := pilot_db.Update("products", productFromRow).
//	    SetLiteral("price", "price * 1.1").  // 10% price increase
//	    Where("category", "= $", "electronics")
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SetLiteral(field WriteKeys, value string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if b.operation != "UPDATE" {
		log.Fatal("Attempted to set a field using literal syntax on a non-update query. This is probably not what you want.")
	}
	set := SetField{string(field), &value, nil}
	lastRecord := len(b.set) - 1
	_, fieldAtLast := b.set[lastRecord][string(field)]
	if !fieldAtLast {
		if lastRecord > 1 {
			_, inLast := b.set[lastRecord-1][string(field)]
			if !inLast {
				log.Fatal("Attempted to set a field in a bulk insert which wasn't in the previous row. Bulk inserts require the same arguments in every row")
			}
		}
		b.set[lastRecord][string(field)] = set
	} else {
		if b.operation != "INSERT" {
			log.Fatal("Added a duplicate key. This is used for bulk inserts but this operation is not an insert.")
		}
		newRow := map[string]SetField{}
		newRow[string(field)] = set
		b.set = append(b.set, newRow)
	}
	return b
}

// Select adds a field to the SELECT clause of the query. The field will be selected from
// the current context table (either the base table or the most recently joined table).
// This method is context-aware and will automatically use the appropriate table alias
// based on the current query context set by Context() or the most recent join operation.
//
// Parameters:
//   - field: The name of the database column to select
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    Select("id").
//	    Select("name").
//	    Select("email")
//
//	// With joins and context switching
//	query := pilot_db.Select("users", userFromRow).
//	    Select("id").Select("name").          // From users table
//	    InnerJoin("profiles", "id", "user_id").
//	    Select("bio").Select("avatar_url")    // From profiles table (current context)
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Select(field ReadKeys) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if b.lastJoin != nil {
		b.fields = append(b.fields, SelectField{string(field), b.lastJoin.alias, string(field), nil})
	} else {
		b.fields = append(b.fields, SelectField{string(field), b.from, string(field), nil})
	}
	return b
}

// SelectAs adds a field to the SELECT clause with a custom alias. This is useful when you
// need to rename columns in the result set or when dealing with name conflicts between
// joined tables. The field will be selected from the current context table.
//
// Parameters:
//   - field: The name of the database column to select
//   - as: The alias to use for this field in the result set
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    SelectAs("name", "user_name").
//	    SelectAs("created_at", "signup_date").
//	    InnerJoin("companies", "company_id", "id").
//	    SelectAs("name", "company_name")  // Avoids conflict with users.name
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SelectAs(field ReadKeys, as string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if b.lastJoin != nil {
		b.fields = append(b.fields, SelectField{string(field), b.lastJoin.alias, as, nil})
	} else {
		b.fields = append(b.fields, SelectField{string(field), b.from, as, nil})
	}
	return b
}

// SelectExprFromBase adds a SQL expression to the SELECT clause using the base table context.
// This method allows you to use database functions, calculations, or other SQL expressions
// in your query results. The expression is not parameter-bound, so use with caution.
//
// Parameters:
//   - field: The alias name for the expression result
//   - expr: The SQL expression to evaluate
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    Select("name").
//	    SelectExprFromBase("age_years", "EXTRACT(YEAR FROM AGE(birth_date))").
//	    SelectExprFromBase("full_name", "CONCAT(first_name, ' ', last_name)")
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SelectExprFromBase(field ReadKeys, expr string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.fields = append(b.fields, SelectField{string(field), b.from, "", &expr})
	return b
}

// SelectFromBaseAs adds a field from the base table with a custom alias, regardless of the
// current context. This is useful when you've joined to other tables but need to explicitly
// select a field from the primary table with a specific alias.
//
// Parameters:
//   - field: The name of the database column to select from the base table
//   - as: The alias to use for this field in the result set
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    InnerJoin("profiles", "id", "user_id").
//	    Select("bio").                              // From profiles (current context)
//	    SelectFromBaseAs("name", "user_name")       // Explicitly from users table
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SelectFromBaseAs(field ReadKeys, as string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.fields = append(b.fields, SelectField{string(field), b.from, as, nil})
	return b
}

// SelectFromAs adds a field from a specific joined table with a custom alias. This method
// allows you to explicitly specify which joined table to select from using the join's alias,
// providing full control over field selection in complex multi-table queries.
//
// Parameters:
//   - field: The name of the database column to select
//   - from: The alias of the joined table to select from
//   - as: The alias to use for this field in the result set
//
// Returns:
//   - *QueryBuilder[T]: The QueryBuilder instance for method chaining
//
// Example:
//
//	query := pilot_db.Select("users", userFromRow).
//	    InnerJoinAs("profiles", "user_profiles", "id", "user_id").
//	    InnerJoinAs("companies", "user_companies", "company_id", "id").
//	    Select("name").                                           // From users
//	    SelectFromAs("bio", "user_profiles", "profile_bio").      // From profiles
//	    SelectFromAs("name", "user_companies", "company_name")    // From companies
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SelectFromAs(field ReadKeys, from string, as string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if from == b.from {
		b.fields = append(b.fields, SelectField{string(field), b.from, as, nil})
		return b
	}
	join, ok := b.joinsByName[from]
	if !ok {
		log.Fatalf("Attempted to select from join %v but there isn't a join.", from)
	}
	b.fields = append(b.fields, SelectField{string(field), join.alias, as, nil})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Where(field ReadKeys, where string, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if len(b.where) > 0 {
		switch b.where[len(b.where)-1].(type) {
		case QueryWhere:
			b.where = append(b.where, ConditionWhereComponent{value: " AND "})
		default:
			break
		}
	}

	b.where = append(b.where, QueryWhere{where: string(field) + " " + where, arg: arg})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereEq(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "= $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereNe(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "<> $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereLt(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "< $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereLte(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "<= $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereGt(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "> $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereGte(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, ">= $", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereAny(field ReadKeys, arg any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "= ANY($)", arg)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereNull(field ReadKeys) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "IS NULL", nil)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereNotNull(field ReadKeys) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "IS NOT NULL", nil)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereLike(field ReadKeys, values any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "LIKE $", values)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereNotLike(field ReadKeys, values any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "NOT LIKE $", values)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) WhereLikeInsensitive(field ReadKeys, values any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.Where(field, "ILIKE $", values)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Or() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if len(b.where) > 0 {
		b.where = append(b.where, StringWhereComponent{value: " OR "})
	} else {
		log.Fatalf("Or() called without any previous Where-style call")
	}
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) And() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if len(b.where) > 0 {
		b.where = append(b.where, StringWhereComponent{value: " AND "})
	} else {
		log.Fatalf("And() called without any previous Where-style call")
	}
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SortAsc(field SortKeys) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.sort = append(b.sort, QuerySort{field: string(field), order: "ASC"})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SortAscExpression(expr string, args ...any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.sort = append(b.sort, QuerySort{field: expr, order: "ASC"})
	b.sortArgs = append(b.sortArgs, args...)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SortDesc(field SortKeys) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.sort = append(b.sort, QuerySort{field: string(field), order: "DESC"})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) SortDescExpression(expr string, args ...any) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.sort = append(b.sort, QuerySort{field: expr, order: "DESC"})
	b.sortArgs = append(b.sortArgs, args...)
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Limit(num int) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.limit = num
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) InnerJoin(table string, local string, foreign string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	alias := string(rune(65 + len(b.joins)))
	where := fmt.Sprintf("%v.%v = %v.%v", b.from, local, alias, foreign)
	b.joins = append(b.joins, QueryJoin{joinKind: "INNER JOIN", table: table, where: where, alias: alias})
	b.lastJoin = &b.joins[len(b.joins)-1]
	b.joinsByName[b.lastJoin.alias] = b.lastJoin
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) InnerJoinAs(table string, alias string, local string, foreign string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	loc := b.from
	if b.operation != "SELECT" {
		loc = "_res"
	}
	where := fmt.Sprintf("%v.%v = %v.%v", loc, local, alias, foreign)
	b.joins = append(b.joins, QueryJoin{joinKind: "INNER JOIN", table: table, where: where, alias: alias})
	b.lastJoin = &b.joins[len(b.joins)-1]
	b.joinsByName[alias] = b.lastJoin
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) GroupBy(field string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.groupBy = &field
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Context(table string) *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	if b.from == table {
		b.lastJoin = nil
	} else {
		for idx := range b.joins {
			if b.joins[idx].table == table {
				b.lastJoin = &b.joins[idx]
				return b
			}
		}
		log.Fatal("Attempted to context a table that was not joined on. This is probably not what you want.")
	}
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Base() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.lastJoin = nil
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Force() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.warn = false
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) selectToString() string {
	var query string
	for _, field := range b.fields {
		if field.expr == nil {
			query += field.table + "."
			query += field.name
			if field.as != "" {
				query += " AS " + field.as
			}
		} else {
			query += *field.expr + " AS " + field.name
		}
		query += ", "
	}
	return query[:len(query)-2]
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) joinToString() string {
	var query string
	for _, join := range b.joins {
		query += join.joinKind + " " + join.table + " " + join.alias + " ON " + join.where
		query += " "
	}
	if len(query) > 1 {
		return query
	} else {
		return query
	}
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) setToInsertString() (string, string, []any) {
	args := []any{}
	keys := []string{}
	var fieldString string = ""
	var argString string = ""
	for fieldName := range b.set[0] {
		fieldString += fieldName + ","
		keys = append(keys, fieldName)
	}
	for setIdx := range b.set {
		argString += "("
		for field := range keys {
			argString += "$,"
			args = append(args, b.set[setIdx][keys[field]].value)
		}
		argString = argString[:len(argString)-1]
		argString += "),"
	}
	return fieldString[:len(fieldString)-1], argString[:len(argString)-1], args
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) setToUpdateString() (string, []any) {
	args := []any{}
	var query string = ""
	for fieldKey := range b.set[0] {
		if b.set[0][fieldKey].set_literal != nil {
			query += b.set[0][fieldKey].field + " = " + *b.set[0][fieldKey].set_literal
		} else {
			query += b.set[0][fieldKey].field + " = $"
			args = append(args, b.set[0][fieldKey].value)
		}
		query += ","
	}
	return query[:len(query)-1], args
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) whereToString() (string, []any) {
	var args []any = []any{}
	var query string = "WHERE "
	if len(b.where) == 0 {
		return "", nil
	}
	for whereIdx := range b.where {
		b.where[whereIdx].Build(&query, &args)
	}
	return query, args
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) StartGroup() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.where = append(b.where, StringWhereComponent{value: "("})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) EndGroup() *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys] {
	b.where = append(b.where, StringWhereComponent{value: ")"})
	return b
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) sortToString() string {
	if len(b.sort) == 0 {
		return ""
	}
	query := " ORDER BY"
	for sortIdx := range b.sort {
		query += " "
		query += b.sort[sortIdx].field
		query += " "
		query += b.sort[sortIdx].order
		query += ", "
	}
	return query[:len(query)-2]
}
func (b *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) limitString() string {
	query := ""
	if b.limit > 0 {
		query = fmt.Sprintf(" LIMIT %v", b.limit)
	}
	return query
}

func (b QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) Build() (string, []any) {
	return b.BuildOffset(0, true)
}

func (b QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) BuildOffset(idx int, selectResults bool) (string, []any) {
	args := []any{}
	var query string
	switch b.operation {
	case "SELECT":
		whereQuery, whereArgs := b.whereToString()
		args = append(args, whereArgs...)
		groupBy := ""
		if b.groupBy != nil {
			groupBy = " GROUP BY " + *b.groupBy
		}
		query = fmt.Sprintf("SELECT %s FROM %s %s%s%v%s%s", b.selectToString(), b.from, b.joinToString(), whereQuery, b.sortToString(), b.limitString(), groupBy)
	case "UPDATE":
		updateQuery, updateArgs := b.setToUpdateString()
		whereQuery, whereArgs := b.whereToString()
		args = append(args, updateArgs...)
		args = append(args, whereArgs...)
		innerQuery := fmt.Sprintf("UPDATE %s SET %s %v", b.from, updateQuery, whereQuery)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res AS %v%v%v", innerQuery, b.selectToString(), b.from, b.joinToString(), b.sortToString())
		}
	case "INSERT":
		insertQuery, insertArgString, insertArgs := b.setToInsertString()
		args = append(args, insertArgs...)
		innerQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", b.from, insertQuery, insertArgString)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res AS %v%v%v", innerQuery, b.selectToString(), b.from, b.joinToString(), b.sortToString())
		}
	case "DELETE":
		whereQuery, whereArgs := b.whereToString()
		args = append(args, whereArgs...)
		innerQuery := fmt.Sprintf("DELETE FROM %s %v", b.from, whereQuery)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res AS %v%v%v", innerQuery, b.selectToString(), b.from, b.joinToString(), b.sortToString())
		}
	}
	query_final := ""
	arg_ct := 1 + idx
	for idx := range query {
		if query[idx] == '$' {
			query_final += fmt.Sprintf("$%v", arg_ct)
			arg_ct += 1
		} else {
			query_final += string(query[idx])
		}
	}
	if b.warn && len(b.where) == 0 && (b.operation == "UPDATE" || b.operation == "DELETE") {
		log.Fatal("Attempted to run a query with no where clause. This is probably not what you want. Override with .Force()")
	}
	args = append(args, b.sortArgs...)
	if b.debug {
		log.Printf("[QUERY]: '%s'", query_final)
	}
	return query_final, args
}

func (builder *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) QueryOneExpect() (*T, *QueryBuilderError) {
	query, args := builder.Build()
	rows, err := (*builder.db).Query(builder.ctx, query, args...)
	if err != nil {
		return nil, PostgresError(builder.from, err)
	}
	defer rows.Close()
	if rows.Next() {
		var value T
		err := builder.conversion(rows, &value)
		if err != nil {
			return nil, PostgresError(builder.from, err)
		}
		return &value, nil
	}
	if rows.Err() != nil {
		return nil, PostgresError(builder.from, rows.Err())
	}
	return nil, NotFoundError(builder.from)
}
func (builder *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) QueryOne() (*T, *QueryBuilderError) {
	query, args := builder.Build()
	rows, err := (*builder.db).Query(builder.ctx, query, args...)
	if err != nil {
		return nil, PostgresError(builder.from, err)
	}
	defer rows.Close()
	if rows.Next() {
		var value T
		err := builder.conversion(rows, &value)
		if err != nil {
			return nil, PostgresError(builder.from, err)
		}
		return &value, nil
	}
	if rows.Err() != nil {
		return nil, PostgresError(builder.from, rows.Err())
	}
	return nil, nil
}
func (builder *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) QueryUpdate(obj *T) *QueryBuilderError {
	query, args := builder.Build()
	rows, err := (*builder.db).Query(builder.ctx, query, args...)
	if err != nil {
		return PostgresError(builder.from, err)
	}
	defer rows.Close()
	if rows.Next() {
		err := builder.conversion(rows, obj)
		if err != nil {
			return PostgresError(builder.from, err)
		}
		return nil
	}
	if rows.Err() != nil {
		return PostgresError(builder.from, rows.Err())
	}
	return nil
}
func (builder *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) QueryMany() ([]T, *QueryBuilderError) {
	results := []T{}
	query, args := builder.Build()
	rows, err := (*builder.db).Query(builder.ctx, query, args...)
	if err != nil {
		return results, PostgresError(builder.from, err)
	}
	defer rows.Close()
	for rows.Next() {
		var value T
		err := builder.conversion(rows, &value)
		if err != nil {
			log.Println(err)
			return results, PostgresError(builder.from, err)
		}
		results = append(results, value)
	}
	if rows.Err() != nil {
		return results, PostgresError(builder.from, rows.Err())
	}
	return results, nil
}
func (builder *QueryBuilder[T, ReadKeys, WriteKeys, SortKeys]) QueryInTransaction(tx *pgx.Tx) *QueryBuilderError {
	query, args := builder.Build()
	rows, err := (*tx).Query(builder.ctx, query, args...)
	if err != nil {
		return PostgresError(builder.from, err)
	}
	defer rows.Close()
	return nil
}

type queryBuilderAlias interface {
	BuildOffset(int, bool) (string, []any)
}

type QueryBuilderTransaction struct {
	builders []queryBuilderAlias
}

func NewQueryBuilderTransaction() QueryBuilderTransaction {
	return QueryBuilderTransaction{
		builders: []queryBuilderAlias{},
	}
}
func (transaction *QueryBuilderTransaction) Add(qb queryBuilderAlias) {
	transaction.builders = append(transaction.builders, qb)
}
func (transaction *QueryBuilderTransaction) Exec(ctx context.Context, db *pgxpool.Conn) *QueryBuilderError {
	query := "BEGIN;\n"
	args := []any{}
	for tIdx := range transaction.builders {
		tQuery, tArgs := transaction.builders[tIdx].BuildOffset(len(args), false)
		query += tQuery + "\n"
		args = append(args, tArgs...)
	}
	query += "COMMIT;"

	_, err := db.Exec(ctx, query, args...)
	if err != nil {
		return PostgresError("Bulk", err)
	}
	return nil
}

type QueryBuilderError struct {
	table        string
	genericError error
}

func (e QueryBuilderError) Error() string {
	friendlyName := strings.ReplaceAll(e.table, " ", "_")
	friendlyName = cases.Title(language.English).String(friendlyName)
	if e.genericError != nil {
		return friendlyName + ": " + e.genericError.Error()
	}
	if strings.HasSuffix(friendlyName, "ies") {
		friendlyName = friendlyName[:len(friendlyName)-3] + "y"
	}
	friendlyName = strings.TrimSuffix(friendlyName, "s")
	return friendlyName + " not found"
}
func (e *QueryBuilderError) Violates(code PostgresErrorCode) bool {
	var pgError *pgconn.PgError
	if errors.As(e.genericError, &pgError) {
		return pgError.Code == string(code)
	}
	return false
}
func PostgresError(table string, err error) *QueryBuilderError {
	return &QueryBuilderError{
		table:        table,
		genericError: err,
	}
}
func NotFoundError(table string) *QueryBuilderError {
	return &QueryBuilderError{
		table:        table,
		genericError: nil,
	}
}
func (e *QueryBuilderError) Response() *pilot_http.HttpResponse {
	if e.genericError != nil {
		return pilot_http.ErrorResponse(e)
	} else {
		return pilot_http.NotFoundResponse(e.Error())
	}
}

type SelectField struct {
	name  string
	table string
	as    string
	expr  *string
}

type SetField struct {
	field       string
	set_literal *string
	value       any
}

type QueryJoin struct {
	joinKind string
	table    string
	where    string
	alias    string
}

type QueryWhere struct {
	where string
	arg   any
}

func (q QueryWhere) Build(query *string, args *[]any) {
	*query += q.where
	if q.arg != nil {
		*args = append(*args, q.arg)
	}
}

type QuerySort struct {
	field string
	order string
}

type PostgresErrorCode string

const (
	PostgresErrorCodeUniqueViolation     PostgresErrorCode = "23505"
	PostgresErrorCodeNotNullViolation    PostgresErrorCode = "23502"
	PostgresErrorCodeForeignKeyViolation PostgresErrorCode = "23503"
	PostgresErrorCodeCheckViolation      PostgresErrorCode = "23514"
)
