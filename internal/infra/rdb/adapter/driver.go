package adapter

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"uuid"

	gosqlite "github.com/glebarez/go-sqlite"
	"github.com/jackc/pgx/v5/stdlib"
)

// Register separate drivers so UUID conversion is limited to RDB connections.
func init() {
	sql.Register("vine-infra-rdb-sqlite", &_Driver{Driver: new(gosqlite.Driver)})
	sql.Register("vine-infra-rdb-pgx", &_Driver{Driver: stdlib.GetDefaultDriver()})
}

type _Driver struct{ driver.Driver }

func (d *_Driver) Open(name string) (driver.Conn, error) {
	conn, err := d.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return &_Conn{Conn: conn}, nil
}

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

func (c *_Connector) Driver() driver.Driver { return c.driver }
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

func (c *_Conn) CheckNamedValue(value *driver.NamedValue) error {
	convertUUIDParameter(value)
	if checker, ok := c.Conn.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return driver.ErrSkip
}

func (c *_Conn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &_Stmt{Stmt: stmt, conn: c}, nil
}

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

func (c *_Conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if executor, ok := c.Conn.(driver.ExecerContext); ok {
		return executor.ExecContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

func (c *_Conn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if querier, ok := c.Conn.(driver.QueryerContext); ok {
		return querier.QueryContext(ctx, query, args)
	}
	return nil, driver.ErrSkip
}

func (c *_Conn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return ctx.Err()
}

func (c *_Conn) ResetSession(ctx context.Context) error {
	if resetter, ok := c.Conn.(driver.SessionResetter); ok {
		return resetter.ResetSession(ctx)
	}
	return ctx.Err()
}

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

func (s *_Stmt) CheckNamedValue(value *driver.NamedValue) error {
	convertUUIDParameter(value)
	if checker, ok := s.Stmt.(driver.NamedValueChecker); ok {
		return checker.CheckNamedValue(value)
	}
	return s.conn.CheckNamedValue(value)
}

func (s *_Stmt) ColumnConverter(index int) driver.ValueConverter {
	if converter, ok := s.Stmt.(driver.ColumnConverter); ok {
		return converter.ColumnConverter(index)
	}
	return driver.DefaultParameterConverter
}

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
