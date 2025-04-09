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

type NoRowError struct {
}

func (e NoRowError) Error() string {
	return "No data found"
}

func BeginTransaction(conn *pgxpool.Conn) *pgx.Tx {
	tx, err := conn.Begin(context.Background())
	if err != nil {
		panic(err)
	}
	return &tx
}
func EndTransaction(tx pgx.Tx) error {
	return tx.Commit(context.Background())
}

type FromTableFn[T any] func(row pgx.Rows) (*T, error)

type QueryBuilder[T any] struct {
	operation   string
	fields      []SelectField
	from        string
	joins       []QueryJoin
	lastJoin    *QueryJoin
	where       []QueryWhere
	set         []map[string]SetField
	joinsByName map[string]*QueryJoin
	conversion  FromTableFn[T]
	warn        bool
	sort        []QuerySort
	limit       int
	groupBy     *string
}

func Select[T any](table string, conversion FromTableFn[T]) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		operation:   "SELECT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []QueryWhere{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func Update[T any](table string, conversion FromTableFn[T]) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		operation:   "UPDATE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []QueryWhere{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func Insert[T any](table string, conversion FromTableFn[T]) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		operation:   "INSERT",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []QueryWhere{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func Delete[T any](table string, conversion FromTableFn[T]) *QueryBuilder[T] {
	return &QueryBuilder[T]{
		operation:   "DELETE",
		from:        table,
		fields:      []SelectField{},
		joins:       []QueryJoin{},
		set:         []map[string]SetField{make(map[string]SetField, 0)},
		lastJoin:    nil,
		where:       []QueryWhere{},
		conversion:  conversion,
		warn:        true,
		sort:        []QuerySort{},
		joinsByName: map[string]*QueryJoin{},
		limit:       -1,
	}
}
func (b *QueryBuilder[T]) Set(field string, value any) *QueryBuilder[T] {
	if b.operation != "UPDATE" && b.operation != "INSERT" {
		log.Fatal("Attempted to set a field on a non-update/insert query. This is probably not what you want.")
	}
	set := SetField{field, nil, value}
	lastRecord := len(b.set) - 1
	_, fieldAtLast := b.set[lastRecord][field]
	if !fieldAtLast {
		if lastRecord > 1 {
			_, inLast := b.set[lastRecord-1][field]
			if !inLast {
				log.Fatal("Attempted to set a field in a bulk insert which wasn't in the previous row. Bulk inserts require the same arguments in every row")
			}
		}
		b.set[lastRecord][field] = set
	} else {
		if b.operation != "INSERT" {
			log.Fatal("Added a duplicate key. This is used for bulk inserts but this operation is not an insert.")
		}
		newRow := map[string]SetField{}
		newRow[field] = set
		b.set = append(b.set, newRow)
	}
	return b
}
func (b *QueryBuilder[T]) SetLiteral(field string, value string) *QueryBuilder[T] {
	if b.operation != "UPDATE" {
		log.Fatal("Attempted to set a field using literal syntax on a non-update query. This is probably not what you want.")
	}
	set := SetField{field, &value, nil}
	lastRecord := len(b.set) - 1
	_, fieldAtLast := b.set[lastRecord][field]
	if !fieldAtLast {
		if lastRecord > 1 {
			_, inLast := b.set[lastRecord-1][field]
			if !inLast {
				log.Fatal("Attempted to set a field in a bulk insert which wasn't in the previous row. Bulk inserts require the same arguments in every row")
			}
		}
		b.set[lastRecord][field] = set
	} else {
		if b.operation != "INSERT" {
			log.Fatal("Added a duplicate key. This is used for bulk inserts but this operation is not an insert.")
		}
		newRow := map[string]SetField{}
		newRow[field] = set
		b.set = append(b.set, newRow)
	}
	return b
}
func (b *QueryBuilder[T]) Select(field string) *QueryBuilder[T] {
	if b.lastJoin != nil {
		b.fields = append(b.fields, SelectField{field, b.lastJoin.alias, field, nil})
	} else {
		b.fields = append(b.fields, SelectField{field, b.from, field, nil})
	}
	return b
}
func (b *QueryBuilder[T]) SelectAs(field string, as string) *QueryBuilder[T] {
	if b.lastJoin != nil {
		b.fields = append(b.fields, SelectField{field, b.lastJoin.alias, as, nil})
	} else {
		b.fields = append(b.fields, SelectField{field, b.from, as, nil})
	}
	return b
}
func (b *QueryBuilder[T]) SelectExprFromBase(field string, expr string) *QueryBuilder[T] {
	b.fields = append(b.fields, SelectField{field, b.from, "", &expr})
	return b
}
func (b *QueryBuilder[T]) SelectFromBaseAs(field string, as string) *QueryBuilder[T] {
	b.fields = append(b.fields, SelectField{field, b.from, as, nil})
	return b
}
func (b *QueryBuilder[T]) SelectFromAs(field string, from string, as string) *QueryBuilder[T] {
	join, ok := b.joinsByName[from]
	if !ok {
		log.Fatalf("Attempted to select from join %v but there isn't a join.", from)
	}
	b.fields = append(b.fields, SelectField{field, join.alias, as, nil})
	return b
}
func (b *QueryBuilder[T]) Where(field string, where string, arg any) *QueryBuilder[T] {
	var discoveredField string
	if b.lastJoin != nil {
		discoveredField = b.lastJoin.alias + "." + field
	} else {
		discoveredField = b.from + "." + field
	}
	b.where = append(b.where, QueryWhere{where: discoveredField + " " + where, arg: arg, joinWith: ""})
	return b
}
func (b *QueryBuilder[T]) WhereEq(field string, arg any) *QueryBuilder[T] {
	b.Where(field, "= $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereNe(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " <> $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereLt(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " < $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereLte(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " <= $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereGt(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " > $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereGte(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " >= $", arg)
	return b
}
func (b *QueryBuilder[T]) WhereAny(field string, arg any) *QueryBuilder[T] {
	b.Where(field, " = ANY($)", arg)
	return b
}
func (b *QueryBuilder[T]) WhereNull(field string) *QueryBuilder[T] {
	b.Where(field, " IS NULL", nil)
	return b
}
func (b *QueryBuilder[T]) WhereNotNull(field string) *QueryBuilder[T] {
	b.Where(field, " IS NOT NULL", nil)
	return b
}
func (b *QueryBuilder[T]) WhereLike(field string, values any) *QueryBuilder[T] {
	b.Where(field, " LIKE $", values)
	return b
}
func (b *QueryBuilder[T]) WhereNotLike(field string, values any) *QueryBuilder[T] {
	b.Where(field, " NOT LIKE $", values)
	return b
}
func (b *QueryBuilder[T]) WhereLikeInsensitive(field string, values any) *QueryBuilder[T] {
	b.Where(field, " ILIKE $", values)
	return b
}
func (b *QueryBuilder[T]) Or() *QueryBuilder[T] {
	if len(b.where) > 0 {
		b.where[len(b.where)-1].joinWith = "OR"
	} else {
		log.Fatalf("Or() called without any previous Where-style call")
	}
	return b
}
func (b *QueryBuilder[T]) And() *QueryBuilder[T] {
	if len(b.where) > 0 {
		b.where[len(b.where)-1].joinWith = "AND"
	} else {
		log.Fatalf("And() called without any previous Where-style call")
	}
	return b
}
func (b *QueryBuilder[T]) SortAsc(field string) *QueryBuilder[T] {
	if b.lastJoin == nil {
		b.sort = append(b.sort, QuerySort{field: b.from + "." + field, order: "ASC"})
	} else {
		b.sort = append(b.sort, QuerySort{field: b.lastJoin.alias + "." + field, order: "ASC"})
	}
	return b
}
func (b *QueryBuilder[T]) SortDesc(field string) *QueryBuilder[T] {
	if b.lastJoin == nil {
		b.sort = append(b.sort, QuerySort{field: b.from + "." + field, order: "DESC"})
	} else {
		b.sort = append(b.sort, QuerySort{field: b.lastJoin.alias + "." + field, order: "DESC"})
	}
	return b
}
func (b *QueryBuilder[T]) Limit(num int) *QueryBuilder[T] {
	b.limit = num
	return b
}
func (b *QueryBuilder[T]) InnerJoin(table string, local string, foreign string) *QueryBuilder[T] {
	if b.operation != "SELECT" && b.operation != "DELETE" {
		log.Fatal("Attempted to join on a non-select query. This is probably not what you want.")
	}
	alias := string(rune(65 + len(b.joins)))
	where := fmt.Sprintf("%v.%v = %v.%v", b.from, local, alias, foreign)
	b.joins = append(b.joins, QueryJoin{joinKind: "INNER JOIN", table: table, where: where, alias: alias})
	b.lastJoin = &b.joins[len(b.joins)-1]
	b.joinsByName[b.lastJoin.alias] = b.lastJoin
	return b
}
func (b *QueryBuilder[T]) InnerJoinAs(table string, alias string, local string, foreign string) *QueryBuilder[T] {
	if b.operation != "SELECT" && b.operation != "DELETE" {
		log.Fatal("Attempted to join on a non-select query. This is probably not what you want.")
	}
	where := fmt.Sprintf("%v.%v = %v.%v", b.from, local, alias, foreign)
	b.joins = append(b.joins, QueryJoin{joinKind: "INNER JOIN", table: table, where: where, alias: alias})
	b.lastJoin = &b.joins[len(b.joins)-1]
	b.joinsByName[alias] = b.lastJoin
	return b
}
func (b *QueryBuilder[T]) GroupBy(field string) *QueryBuilder[T] {
	b.groupBy = &field
	return b
}
func (b *QueryBuilder[T]) Context(table string) *QueryBuilder[T] {
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
func (b *QueryBuilder[T]) Base() *QueryBuilder[T] {
	b.lastJoin = nil
	return b
}
func (b *QueryBuilder[T]) Force() *QueryBuilder[T] {
	b.warn = false
	return b
}
func (b *QueryBuilder[T]) selectToString(replaceLocal string) string {
	var query string
	for _, field := range b.fields {
		if field.expr == nil {
			if replaceLocal != "" && field.table == b.from {
				query += replaceLocal + "."
			} else {
				query += field.table + "."
			}
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
func (b *QueryBuilder[T]) joinToString() string {
	var query string
	for _, join := range b.joins {
		query += join.joinKind + " " + join.table + " " + join.alias + " ON " + join.where
		query += " "
	}
	if len(query) > 1 {
		return query[:len(query)-1]
	} else {
		return query
	}
}
func (b *QueryBuilder[T]) setToInsertString() (string, string, []any) {
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
func (b *QueryBuilder[T]) setToUpdateString() (string, []any) {
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
func (b *QueryBuilder[T]) whereToString() (string, []any) {
	var args []any = []any{}
	var query string = "WHERE "
	if len(b.where) == 0 {
		return "", nil
	}
	for whereIdx := range b.where {
		query += b.where[whereIdx].where
		if b.where[whereIdx].arg != nil {
			args = append(args, b.where[whereIdx].arg)
		}
		if b.where[whereIdx].joinWith != "" {
			query += " " + b.where[whereIdx].joinWith + " "
		} else {
			query += " AND "
		}
	}
	return query[:len(query)-5], args
}
func (b *QueryBuilder[T]) sortToString() string {
	if len(b.sort) == 0 {
		return ""
	}
	query := "ORDER BY "
	for sortIdx := range b.sort {
		query += " "
		query += b.sort[sortIdx].field
		query += " "
		query += b.sort[sortIdx].order
		query += ", "
	}
	return query[:len(query)-2]
}
func (b *QueryBuilder[T]) limitString() string {
	query := ""
	if b.limit > 0 {
		query = fmt.Sprintf("LIMIT %v", b.limit)
	}
	return query
}

func (b QueryBuilder[T]) Build() (string, []any) {
	return b.BuildOffset(0, true)
}

func (b QueryBuilder[T]) BuildOffset(idx int, selectResults bool) (string, []any) {
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
		query = fmt.Sprintf("SELECT %s FROM %s %s %s %v %s %s", b.selectToString(""), b.from, b.joinToString(), whereQuery, b.sortToString(), b.limitString(), groupBy)
	case "UPDATE":
		updateQuery, updateArgs := b.setToUpdateString()
		whereQuery, whereArgs := b.whereToString()
		args = append(args, updateArgs...)
		args = append(args, whereArgs...)
		innerQuery := fmt.Sprintf("UPDATE %s SET %s %v", b.from, updateQuery, whereQuery)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res %v %v", innerQuery, b.selectToString("_res"), b.joinToString(), b.sortToString())
		}
	case "INSERT":
		insertQuery, insertArgString, insertArgs := b.setToInsertString()
		args = append(args, insertArgs...)
		innerQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", b.from, insertQuery, insertArgString)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res %v %v", innerQuery, b.selectToString("_res"), b.joinToString(), b.sortToString())
		}
	case "DELETE":
		whereQuery, whereArgs := b.whereToString()
		args = append(args, whereArgs...)
		innerQuery := fmt.Sprintf("DELETE FROM %s %v", b.from, whereQuery)
		if !selectResults {
			query = innerQuery
		} else {
			query = fmt.Sprintf("WITH _res AS (%v RETURNING *) SELECT %v FROM _res %v %v", innerQuery, b.selectToString("_res"), b.joinToString(), b.sortToString())
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
	return query_final, args
}

func (builder *QueryBuilder[T]) QueryOneExpect(ctx context.Context, conn *pgxpool.Conn) (*T, *QueryBuilderError) {
	query, args := builder.Build()
	rows, err := (*conn).Query(ctx, query, args...)
	defer rows.Close()
	if err != nil {
		return nil, PostgresError(builder.from, err)
	}
	for rows.Next() {
		value, err := builder.conversion(rows)
		if err != nil {
			return nil, PostgresError(builder.from, err)
		}
		return value, nil
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, PostgresError(builder.from, rows.Err())
	}
	return nil, NotFoundError(builder.from)
}
func (builder *QueryBuilder[T]) QueryOne(ctx context.Context, conn *pgxpool.Conn) (*T, *QueryBuilderError) {
	query, args := builder.Build()
	rows, err := (*conn).Query(ctx, query, args...)
	defer rows.Close()
	if err != nil {
		return nil, PostgresError(builder.from, err)
	}
	for rows.Next() {
		value, err := builder.conversion(rows)
		if err != nil {
			return nil, PostgresError(builder.from, err)
		}
		return value, nil
	}
	rows.Close()
	if rows.Err() != nil {
		return nil, PostgresError(builder.from, rows.Err())
	}
	return nil, nil
}
func (builder *QueryBuilder[T]) QueryMany(ctx context.Context, conn *pgxpool.Conn) (*[]T, *QueryBuilderError) {
	results := []T{}
	query, args := builder.Build()
	rows, err := (*conn).Query(ctx, query, args...)
	defer rows.Close()
	if err != nil {
		return &results, PostgresError(builder.from, err)
	}
	for rows.Next() {
		value, err := builder.conversion(rows)
		if err != nil {
			log.Println(err)
			return &results, PostgresError(builder.from, err)
		}
		results = append(results, *value)
	}
	if rows.Err() != nil {
		return &results, PostgresError(builder.from, rows.Err())
	}
	return &results, nil
}
func (builder *QueryBuilder[T]) QueryInTransaction(ctx context.Context, tx *pgx.Tx) *QueryBuilderError {
	query, args := builder.Build()
	rows, err := (*tx).Query(ctx, query, args...)
	defer rows.Close()
	if err != nil {
		return PostgresError(builder.from, err)
	}
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
	friendlyName := strings.Replace(e.table, " ", "_", -1)
	friendlyName = cases.Title(language.English).String(friendlyName)
	if e.genericError != nil {
		return friendlyName + ": " + e.genericError.Error()
	}
	if strings.HasSuffix(friendlyName, "ies") {
		friendlyName = friendlyName[:len(friendlyName)-3] + "y"
	}
	if strings.HasSuffix(friendlyName, "s") {
		friendlyName = friendlyName[:len(friendlyName)-1]
	}
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
	where    string
	arg      any
	joinWith string
}

type QuerySort struct {
	field string
	order string
}

type PostgresErrorCode string

const (
	PostgresErrorCodeUniqueViolation PostgresErrorCode = "23505"
	PostgresErrorCodeNotNullViolation PostgresErrorCode = "23502"
	PostgresErrorCodeForeignKeyViolation PostgresErrorCode = "23503"
	PostgresErrorCodeCheckViolation PostgresErrorCode = "23514"
)
