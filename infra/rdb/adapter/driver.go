package adapter

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5/stdlib"
)

// Register separate drivers so UUID conversion is limited to RDB connections.
func init() {
	// sql.Open is lazy: obtain the registered driver without opening a connection.
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		panic(err)
	}
	sqliteDriver := db.Driver()
	_ = db.Close()
	sql.Register("vine-rdb-sqlite", &_Driver{Driver: sqliteDriver})
	sql.Register("vine-rdb-pgx", &_Driver{Driver: stdlib.GetDefaultDriver()})
}

type _Driver struct{ driver.Driver }

// Open opens a connection and wraps it with UUID parameter conversion.
func (d *_Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return &_Conn{Conn: conn}, nil
}

// OpenConnector wraps the underlying connector, falling back to Open
// when the driver does not implement driver.DriverContext.
func (d *_Driver) OpenConnector(name string) (driver.Connector, error) {
	if dc, ok := d.Driver.(driver.DriverContext); ok {
		connector, err := dc.OpenConnector(name)
		if err != nil {
			return nil, err
		}
		return &_Connector{Connector: connector, driver: d}, nil
	}
	return &_Connector{driver: d, name: name}, nil
}

type _Connector struct {
	driver.Connector
	driver *_Driver
	name   string
}

// Driver returns the UUID-aware driver for this connector.
func (c *_Connector) Driver() driver.Driver { return c.driver }

// Connect opens a wrapped connection through the underlying connector
// or falls back to the driver's Open method.
func (c *_Connector) Connect(ctx context.Context) (driver.Conn, error) {
	if c.Connector == nil {
		return c.driver.Open(c.name)
	}
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &_Conn{Conn: conn}, nil
}

func convertUUIDParameter(value *driver.NamedValue) {
	switch id := value.Value.(type) {
	case uuid.UUID:
		value.Value = id.String()
	case *uuid.UUID:
		if id == nil {
			value.Value = nil
		} else {
			value.Value = id.String()
		}
	}
}

type _Conn struct{ driver.Conn }

// CheckNamedValue converts UUID parameters before delegating validation
// to the underlying connection; otherwise it returns driver.ErrSkip.
func (c *_Conn) CheckNamedValue(value *driver.NamedValue) error {
	convertUUIDParameter(value)
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return driver.ErrSkip
}

// Prepare prepares SQL and wraps the statement with UUID parameter conversion.
func (c *_Conn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &_Stmt{Stmt: stmt, conn: c}, nil
}

// PrepareContext delegates context-aware preparation when supported.
// The fallback checks cancellation before calling Prepare.
func (c *_Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if preparer, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := preparer.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &_Stmt{Stmt: stmt, conn: c}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.Prepare(query)
}

// BeginTx delegates transaction options when supported.
// The fallback rejects non-default options and checks cancellation before beginning.
func (c *_Conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.Conn.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}
	if opts.Isolation != 0 || opts.ReadOnly {
		return nil, fmt.Errorf("rdb: driver does not support transaction options")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return c.Conn.Begin()
}

// ExecContext delegates direct execution when supported, or returns driver.ErrSkip
// so database/sql can fall back to a prepared statement.
func (c *_Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if executor, ok := c.Conn.(driver.ExecerContext); ok {
		return executor.ExecContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// QueryContext delegates direct queries when supported, or returns driver.ErrSkip
// so database/sql can fall back to a prepared statement.
func (c *_Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if querier, ok := c.Conn.(driver.QueryerContext); ok {
		return querier.QueryContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

// Ping delegates connection health checks when supported; otherwise it returns
// the context error without issuing a database operation.
func (c *_Conn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return ctx.Err()
}

// ResetSession delegates session reset when supported; otherwise it returns
// the context error without changing the connection.
func (c *_Conn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return ctx.Err()
}

// IsValid delegates connection validation when supported; otherwise it returns true.
func (c *_Conn) IsValid() bool {
	if validator, ok := c.Conn.(driver.Validator); ok {
		return validator.IsValid()
	}
	return true
}

type _Stmt struct {
	driver.Stmt
	conn *_Conn
}

// CheckNamedValue converts UUID parameters and delegates to the statement
// checker, falling back to the connection checker.
func (s *_Stmt) CheckNamedValue(value *driver.NamedValue) error {
	convertUUIDParameter(value)
	if checker, ok := s.Stmt.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return s.conn.CheckNamedValue(value)
}

// ColumnConverter returns the underlying statement converter when supported,
// or the default SQL parameter converter.
func (s *_Stmt) ColumnConverter(index int) driver.ValueConverter {
	if converter, ok := s.Stmt.(driver.ColumnConverter); ok {
		return converter.ColumnConverter(index)
	}
	return driver.DefaultParameterConverter
}

// ExecContext delegates context-aware execution when supported.
// The fallback checks cancellation, rejects named parameters, and executes positional values.
func (s *_Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if executor, ok := s.Stmt.(driver.StmtExecContext); ok {
		return executor.ExecContext(ctx, args)
	}
	values, err := positionalValues(ctx, args)
	if err != nil {
		return nil, err
	}
	return s.Stmt.Exec(values)
}

// QueryContext delegates context-aware queries when supported.
// The fallback checks cancellation, rejects named parameters, and queries positional values.
func (s *_Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if querier, ok := s.Stmt.(driver.StmtQueryContext); ok {
		return querier.QueryContext(ctx, args)
	}
	values, err := positionalValues(ctx, args)
	if err != nil {
		return nil, err
	}
	return s.Stmt.Query(values)
}

func positionalValues(ctx context.Context, args []driver.NamedValue) ([]driver.Value, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		if arg.Name != "" {
			return nil, fmt.Errorf("rdb: driver does not support named parameters")
		}
		values[i] = arg.Value
	}
	return values, nil
}
