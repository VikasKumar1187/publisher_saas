// Package db provides support for access the database.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/vikaskumar1187/publisher_saas/foundation/logger"
	"github.com/vikaskumar1187/publisher_saas/foundation/web"
	"go.opentelemetry.io/otel/attribute"
)

const (
	uniqueViolation = "23505"
	undefinedTable  = "42P01"
)

// Set of error variables for CRUD operations.
var (
	ErrDBNotFound        = sql.ErrNoRows
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
	ErrUndefinedTable    = errors.New("undefined table")
)

// Config is the required properties to use the database.
type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	Schema       string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

// Open knows how to open a database connection based on the configuration.
func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")
	if cfg.Schema != "" {
		q.Set("search_path", cfg.Schema)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

// StatusCheck returns nil if it can successfully talk to the database. It
// returns a non-nil error otherwise.
func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	var pingError error
	for attempts := 1; ; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Run a simple query to determine connectivity.
	// Running this query forces a round trip through the database.
	const q = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}

// ExecContext is a helper function to execute a CUD operation with
// logging and tracing.
func ExecContext(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string) error {
	return NamedExecContext(ctx, log, db, query, struct{}{})
}

// NamedExecContext is a helper function to execute a CUD operation with
// logging and tracing where field replacement is necessary.
func NamedExecContext(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any) error {
	q := queryString(query, data)

	if _, ok := data.(struct{}); ok {
		log.Infoc(ctx, 5, "database.NamedExecContext", "query", q)
	} else {
		log.Infoc(ctx, 4, "database.NamedExecContext", "query", q)
	}

	ctx, span := web.AddSpan(ctx, "business.sys.database.exec", attribute.String("query", q))
	defer span.End()

	if _, err := sqlx.NamedExecContext(ctx, db, query, data); err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok {
			switch pqerr.Code {
			case undefinedTable:
				return ErrUndefinedTable
			case uniqueViolation:
				return ErrDBDuplicatedEntry
			}
		}
		return err
	}

	return nil
}

// QuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshalled into a slice.
func QuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, struct{}{}, dest, false)
}

// NamedQuerySlice is a helper function for executing queries that return a
// collection of data to be unmarshalled into a slice where field replacement is
// necessary.
func NamedQuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, data, dest, false)
}

// NamedQuerySliceUsingIn is a helper function for executing queries that return
// a collection of data to be unmarshalled into a slice where field replacement
// is necessary. Use this if the query has an IN clause.
func NamedQuerySliceUsingIn[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T) error {
	return namedQuerySlice(ctx, log, db, query, data, dest, true)
}

func namedQuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *[]T, withIn bool) error {
	q := queryString(query, data)

	log.Infoc(ctx, 5, "database.NamedQuerySlice", "query", q)

	ctx, span := web.AddSpan(ctx, "business.sys.database.queryslice", attribute.String("query", q))
	defer span.End()

	var rows *sqlx.Rows
	var err error

	switch withIn {
	case true:
		rows, err = func() (*sqlx.Rows, error) {
			named, args, err := sqlx.Named(query, data)
			if err != nil {
				return nil, err
			}

			query, args, err := sqlx.In(named, args...)
			if err != nil {
				return nil, err
			}

			query = db.Rebind(query)
			return db.QueryxContext(ctx, query, args...)
		}()

	default:
		rows, err = sqlx.NamedQueryContext(ctx, db, query, data)
	}

	if err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok && pqerr.Code == undefinedTable {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	var slice []T
	for rows.Next() {
		v := new(T)
		if err := rows.StructScan(v); err != nil {
			return err
		}
		slice = append(slice, *v)
	}
	*dest = slice

	return nil
}

// QueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type where field replacement is necessary.
func QueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, dest any) error {
	return namedQueryStruct(ctx, log, db, query, struct{}{}, dest, false)
}

// NamedQueryStruct is a helper function for executing queries that return a
// single value to be unmarshalled into a struct type where field replacement is necessary.
func NamedQueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any) error {
	return namedQueryStruct(ctx, log, db, query, data, dest, false)
}

// NamedQueryStructUsingIn is a helper function for executing queries that return
// a single value to be unmarshalled into a struct type where field replacement
// is necessary. Use this if the query has an IN clause.
func NamedQueryStructUsingIn(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any) error {
	return namedQueryStruct(ctx, log, db, query, data, dest, true)
}

func namedQueryStruct(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest any, withIn bool) error {
	q := queryString(query, data)

	log.Infoc(ctx, 5, "database.NamedQueryStruct", "query", q)

	ctx, span := web.AddSpan(ctx, "business.sys.database.query", attribute.String("query", q))
	defer span.End()

	var rows *sqlx.Rows
	var err error

	switch withIn {
	case true:
		rows, err = func() (*sqlx.Rows, error) {
			named, args, err := sqlx.Named(query, data)
			if err != nil {
				return nil, err
			}

			query, args, err := sqlx.In(named, args...)
			if err != nil {
				return nil, err
			}

			query = db.Rebind(query)
			return db.QueryxContext(ctx, query, args...)
		}()

	default:
		rows, err = sqlx.NamedQueryContext(ctx, db, query, data)
	}

	if err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok && pqerr.Code == undefinedTable {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrDBNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return err
	}

	return nil
}

// queryString provides a pretty print version of the query and parameters.
func queryString(query string, args any) string {
	query, params, err := sqlx.Named(query, args)
	if err != nil {
		return err.Error()
	}

	for _, param := range params {
		var value string
		switch v := param.(type) {
		case string:
			value = fmt.Sprintf("'%s'", v)
		case []byte:
			value = fmt.Sprintf("'%s'", string(v))
		default:
			value = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, "?", value, 1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.Trim(query, " ")
}
