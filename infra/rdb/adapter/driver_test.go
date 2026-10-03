package adapter

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	gosqlite "github.com/glebarez/go-sqlite"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUUIDQueryParameters(t *testing.T) {
	url := "sqlite://" + t.TempDir() + "/params.sqlite"
	db, err := openDriverTestDB(t, url)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(new(driverTestModel)))
	for _, prepared := range []bool{false, true} {
		name := "direct"
		if prepared {
			name = "prepared"
		}
		t.Run(name, func(t *testing.T) {
			db := db.Session(&gorm.Session{PrepareStmt: prepared})
			if prepared {
				t.Cleanup(func() { db.ConnPool.(*gorm.PreparedStmtDB).Close() })
			}
			rows := []*driverTestModel{new(driverTestModel{Name: "a"}), new(driverTestModel{Name: "b"})}
			require.NoError(t, db.Create(&rows).Error)
			ids := []uuid.UUID{rows[0].Id, rows[1].Id}
			var found []driverTestModel
			require.NoError(t, db.Where("id IN ?", ids).Find(&found).Error)
			require.Len(t, found, 2)
			found = nil
			require.NoError(t, db.Raw("SELECT * FROM driver_test_models WHERE id IN (?)", ids).Scan(&found).Error)
			require.Len(t, found, 2)
			var row driverTestModel
			require.NoError(t, db.Where("id = ?", &ids[0]).First(&row).Error)
			assert.Equal(t, ids[0], row.Id)
			require.NoError(t, db.Raw("SELECT * FROM driver_test_models WHERE id = @id", sql.Named("id", ids[1])).Scan(&row).Error)
			assert.Equal(t, ids[1], row.Id)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return tx.Exec("UPDATE driver_test_models SET name = ? WHERE id = ?", "updated", ids[0]).Error
			}))
			row = driverTestModel{}
			require.NoError(t, db.Where("id = ?", ids[0]).First(&row).Error)
			assert.Equal(t, "updated", row.Name)
			found = nil
			require.NoError(t, db.Where("id IN ?", []uuid.UUID{}).Find(&found).Error)
			assert.Empty(t, found)
		})
	}
}

func TestUUIDSQLPreparedParameters(t *testing.T) {
	url := "sqlite://" + t.TempDir() + "/sql.sqlite"
	db, err := openDriverTestDB(t, url)
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	ctx := context.Background()
	stmt, err := pool.PrepareContext(ctx, "SELECT ?, ?, ?, ?")
	require.NoError(t, err)
	t.Cleanup(func() { _ = stmt.Close() })
	id := uuid.NewV7()
	var text string
	var number int
	var bytes []byte
	var null any
	require.NoError(t, stmt.QueryRowContext(ctx, id, 42, []byte{1, 2, 3}, (*uuid.UUID)(nil)).Scan(&text, &number, &bytes, &null))
	assert.Equal(t, id.String(), text)
	assert.Equal(t, 42, number)
	assert.Equal(t, []byte{1, 2, 3}, bytes)
	assert.Nil(t, null)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, tx.StmtContext(ctx, stmt).QueryRowContext(ctx, &id, 7, []byte{4}, nil).Scan(&text, &number, &bytes, &null))
	assert.Equal(t, id.String(), text)
	require.NoError(t, tx.Commit())
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	assert.ErrorIs(t, stmt.QueryRowContext(canceled, id, 1, nil, nil).Err(), context.Canceled)
}

// Fixtures deliberately use plain GORM models without depending on the parent rdb package.
type driverTestModel struct {
	Id   uuid.UUID `gorm:"primaryKey;type:uuid;serializer:vine-rdb-uuid"`
	Name string
}

func openDriverTestDB(t *testing.T, url string) (*gorm.DB, error) {
	t.Helper()
	db, err := gorm.Open(NewDialector(url), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = pool.Close() })
	return db, nil
}

func TestDriverPreservesSQLiteRegisteredFunctions(t *testing.T) {
	// Registration is process-global; a unique name also supports repeated test runs.
	name := "vine_test_" + strings.ReplaceAll(uuid.NewV7().String(), "-", "")
	require.NoError(t, gosqlite.RegisterScalarFunction(name, 0,
		func(_ *gosqlite.FunctionContext, _ []driver.Value) (driver.Value, error) {
			return int64(17), nil
		}))
	for _, driverName := range []string{"sqlite", "vine-rdb-sqlite"} {
		t.Run(driverName, func(t *testing.T) {
			db, err := sql.Open(driverName, ":memory:")
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			var result int
			require.NoError(t, db.QueryRow("SELECT "+name+"()").Scan(&result))
			require.Equal(t, 17, result)
		})
	}
}

type driverOpenStub struct {
	open func(string) (driver.Conn, error)
}

func (d *driverOpenStub) Open(name string) (driver.Conn, error) { return d.open(name) }

type driverConnectorStub struct {
	driver.Connector
	connect func(context.Context) (driver.Conn, error)
}

func (c *driverConnectorStub) Connect(ctx context.Context) (driver.Conn, error) {
	return c.connect(ctx)
}

type driverContextStub struct {
	driverOpenStub
	connector func(string) (driver.Connector, error)
}

func (d *driverContextStub) OpenConnector(name string) (driver.Connector, error) {
	return d.connector(name)
}

type driverConnStub struct {
	driver.Conn
	prepare func(string) (driver.Stmt, error)
	begin   func() (driver.Tx, error)
}

func (c *driverConnStub) Prepare(query string) (driver.Stmt, error) { return c.prepare(query) }
func (c *driverConnStub) Begin() (driver.Tx, error)                 { return c.begin() }

type driverContextConnStub struct {
	driverConnStub
	check          func(*driver.NamedValue) error
	prepareContext func(context.Context, string) (driver.Stmt, error)
	beginTx        func(context.Context, driver.TxOptions) (driver.Tx, error)
	execContext    func(context.Context, string, []driver.NamedValue) (driver.Result, error)
	queryContext   func(context.Context, string, []driver.NamedValue) (driver.Rows, error)
}

func (c *driverContextConnStub) CheckNamedValue(value *driver.NamedValue) error {
	return c.check(value)
}
func (c *driverContextConnStub) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.prepareContext(ctx, query)
}
func (c *driverContextConnStub) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return c.beginTx(ctx, opts)
}
func (c *driverContextConnStub) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.execContext(ctx, query, args)
}
func (c *driverContextConnStub) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.queryContext(ctx, query, args)
}

type driverStmtStub struct {
	driver.Stmt
	exec  func([]driver.Value) (driver.Result, error)
	query func([]driver.Value) (driver.Rows, error)
}

func (s *driverStmtStub) Exec(args []driver.Value) (driver.Result, error) { return s.exec(args) }
func (s *driverStmtStub) Query(args []driver.Value) (driver.Rows, error)  { return s.query(args) }

type driverContextStmtStub struct {
	driverStmtStub
	check        func(*driver.NamedValue) error
	execContext  func(context.Context, []driver.NamedValue) (driver.Result, error)
	queryContext func(context.Context, []driver.NamedValue) (driver.Rows, error)
}

func (s *driverContextStmtStub) CheckNamedValue(value *driver.NamedValue) error {
	return s.check(value)
}
func (s *driverContextStmtStub) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	return s.execContext(ctx, args)
}
func (s *driverContextStmtStub) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	return s.queryContext(ctx, args)
}

func TestDriverConnectorDelegation(t *testing.T) {
	ctx := t.Context()
	failure := errors.New("connect failed")
	raw := new(driverConnStub)
	connector := &driverConnectorStub{connect: func(received context.Context) (driver.Conn, error) {
		assert.Same(t, ctx, received)
		return raw, nil
	}}
	wrapped := &_Driver{Driver: &driverContextStub{connector: func(name string) (driver.Connector, error) {
		assert.Equal(t, "dsn", name)
		return connector, nil
	}}}
	opened, err := wrapped.OpenConnector("dsn")
	require.NoError(t, err)
	assert.Same(t, wrapped, opened.Driver())
	conn, err := opened.Connect(ctx)
	require.NoError(t, err)
	assert.Same(t, raw, conn.(*_Conn).Conn)
	connector.connect = func(context.Context) (driver.Conn, error) { return nil, failure }
	conn, err = opened.Connect(ctx)
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, failure)
	wrapped.Driver.(*driverContextStub).connector = func(string) (driver.Connector, error) { return nil, failure }
	opened, err = wrapped.OpenConnector("dsn")
	assert.Nil(t, opened)
	assert.ErrorIs(t, err, failure)
	wrapped.Driver = &driverOpenStub{open: func(name string) (driver.Conn, error) {
		assert.Equal(t, "fallback", name)
		return raw, nil
	}}
	opened, err = wrapped.OpenConnector("fallback")
	require.NoError(t, err)
	conn, err = opened.Connect(ctx)
	require.NoError(t, err)
	assert.Same(t, raw, conn.(*_Conn).Conn)
	wrapped.Driver.(*driverOpenStub).open = func(string) (driver.Conn, error) { return nil, failure }
	conn, err = opened.Connect(ctx)
	assert.Nil(t, conn)
	assert.ErrorIs(t, err, failure)
}

func TestDriverNamedValueConversionAndCheckerPrecedence(t *testing.T) {
	id := uuid.NewV7()
	failure := errors.New("checker error")
	for _, item := range []struct {
		name     string
		input    any
		expected any
	}{
		{"uuid", id, id.String()}, {"pointer", &id, id.String()}, {"null", (*uuid.UUID)(nil), nil}, {"other", int64(7), int64(7)},
	} {
		t.Run(item.name, func(t *testing.T) {
			calls := []string{}
			underlying := &driverContextConnStub{check: func(value *driver.NamedValue) error {
				calls = append(calls, "connection")
				assert.Equal(t, driver.NamedValue{Name: "arg", Ordinal: 2, Value: item.expected}, *value)
				return failure
			}}
			conn := &_Conn{Conn: underlying}
			value := driver.NamedValue{Name: "arg", Ordinal: 2, Value: item.input}
			assert.ErrorIs(t, conn.CheckNamedValue(&value), failure)
			statement := &_Stmt{conn: conn, Stmt: new(driverStmtStub)}
			value.Value = item.input
			assert.ErrorIs(t, statement.CheckNamedValue(&value), failure)
			statement.Stmt = &driverContextStmtStub{check: func(value *driver.NamedValue) error {
				calls = append(calls, "statement")
				assert.Equal(t, item.expected, value.Value)
				return driver.ErrRemoveArgument
			}}
			value.Value = item.input
			assert.ErrorIs(t, statement.CheckNamedValue(&value), driver.ErrRemoveArgument)
			assert.Equal(t, []string{"connection", "connection", "statement"}, calls)
			conn.Conn = new(driverConnStub)
			value.Value = item.input
			assert.ErrorIs(t, conn.CheckNamedValue(&value), driver.ErrSkip)
			assert.Equal(t, item.expected, value.Value)
		})
	}
}

func TestDriverConnectionContextDelegation(t *testing.T) {
	ctx := t.Context()
	failure := errors.New("driver failure")
	args := []driver.NamedValue{{Name: "value", Ordinal: 1, Value: "uuid-text"}}
	options := driver.TxOptions{ReadOnly: true, Isolation: driver.IsolationLevel(sql.LevelSerializable)}
	calls := []string{}
	statement := new(driverStmtStub)
	underlying := &driverContextConnStub{
		prepareContext: func(received context.Context, query string) (driver.Stmt, error) {
			calls = append(calls, "prepare")
			assert.Same(t, ctx, received)
			assert.Equal(t, "query", query)
			return statement, nil
		},
		beginTx: func(received context.Context, receivedOptions driver.TxOptions) (driver.Tx, error) {
			calls = append(calls, "begin")
			assert.Same(t, ctx, received)
			assert.Equal(t, options, receivedOptions)
			return nil, failure
		},
		execContext: func(received context.Context, query string, receivedArgs []driver.NamedValue) (driver.Result, error) {
			calls = append(calls, "exec")
			assert.Same(t, ctx, received)
			assert.Equal(t, "query", query)
			assert.Equal(t, args, receivedArgs)
			return driver.RowsAffected(3), failure
		},
		queryContext: func(received context.Context, query string, receivedArgs []driver.NamedValue) (driver.Rows, error) {
			calls = append(calls, "query")
			assert.Same(t, ctx, received)
			assert.Equal(t, "query", query)
			assert.Equal(t, args, receivedArgs)
			return nil, failure
		},
	}
	conn := &_Conn{Conn: underlying}
	prepared, err := conn.PrepareContext(ctx, "query")
	require.NoError(t, err)
	assert.Same(t, statement, prepared.(*_Stmt).Stmt)
	assert.Same(t, conn, prepared.(*_Stmt).conn)
	_, err = conn.BeginTx(ctx, options)
	assert.ErrorIs(t, err, failure)
	result, err := conn.ExecContext(ctx, "query", args)
	assert.ErrorIs(t, err, failure)
	assert.Equal(t, driver.RowsAffected(3), result)
	_, err = conn.QueryContext(ctx, "query", args)
	assert.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"prepare", "begin", "exec", "query"}, calls)
	underlying.prepareContext = func(context.Context, string) (driver.Stmt, error) { return nil, failure }
	prepared, err = conn.PrepareContext(ctx, "query")
	assert.Nil(t, prepared)
	assert.ErrorIs(t, err, failure)
}

func TestDriverConnectionFallbackRejectsUnsupportedOperations(t *testing.T) {
	failure := errors.New("prepare failure")
	calls := []string{}
	underlying := &driverConnStub{
		prepare: func(query string) (driver.Stmt, error) {
			calls = append(calls, "prepare")
			assert.Equal(t, "query", query)
			return nil, failure
		},
		begin: func() (driver.Tx, error) {
			calls = append(calls, "begin")
			return nil, failure
		},
	}
	conn := &_Conn{Conn: underlying}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := conn.PrepareContext(ctx, "query")
	assert.ErrorIs(t, err, context.Canceled)
	_, err = conn.BeginTx(ctx, driver.TxOptions{})
	assert.ErrorIs(t, err, context.Canceled)
	for _, opts := range []driver.TxOptions{{ReadOnly: true}, {Isolation: driver.IsolationLevel(sql.LevelSerializable)}} {
		_, err = conn.BeginTx(t.Context(), opts)
		assert.ErrorContains(t, err, "does not support transaction options")
	}
	assert.Empty(t, calls, "rejected requests must never reach the underlying driver")
	_, err = conn.PrepareContext(t.Context(), "query")
	assert.ErrorIs(t, err, failure)
	_, err = conn.BeginTx(t.Context(), driver.TxOptions{})
	assert.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"prepare", "begin"}, calls)
	_, err = conn.ExecContext(t.Context(), "query", nil)
	assert.ErrorIs(t, err, driver.ErrSkip)
	_, err = conn.QueryContext(t.Context(), "query", nil)
	assert.ErrorIs(t, err, driver.ErrSkip)
	assert.ErrorIs(t, conn.Ping(ctx), context.Canceled)
	assert.ErrorIs(t, conn.ResetSession(ctx), context.Canceled)
	assert.True(t, conn.IsValid())
}

func TestDriverStatementFallbackPreservesArgumentsAndCancellation(t *testing.T) {
	failure := errors.New("statement failure")
	calls := []string{}
	expected := []driver.Value{"uuid-text", int64(7), nil, []byte{1, 2}}
	underlying := &driverStmtStub{
		exec: func(values []driver.Value) (driver.Result, error) {
			calls = append(calls, "exec")
			assert.Equal(t, expected, values)
			return driver.RowsAffected(2), failure
		},
		query: func(values []driver.Value) (driver.Rows, error) {
			calls = append(calls, "query")
			assert.Equal(t, expected, values)
			return nil, failure
		},
	}
	statement := &_Stmt{Stmt: underlying}
	args := make([]driver.NamedValue, len(expected))
	for i, value := range expected {
		args[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := statement.ExecContext(ctx, args)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = statement.QueryContext(ctx, args)
	assert.ErrorIs(t, err, context.Canceled)
	named := []driver.NamedValue{{Name: "arg", Ordinal: 1, Value: 7}}
	_, err = statement.ExecContext(t.Context(), named)
	assert.ErrorContains(t, err, "does not support named parameters")
	_, err = statement.QueryContext(t.Context(), named)
	assert.ErrorContains(t, err, "does not support named parameters")
	assert.Empty(t, calls)
	result, err := statement.ExecContext(t.Context(), args)
	assert.Equal(t, driver.RowsAffected(2), result)
	assert.ErrorIs(t, err, failure)
	_, err = statement.QueryContext(t.Context(), args)
	assert.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"exec", "query"}, calls)
	assert.Equal(t, driver.DefaultParameterConverter, statement.ColumnConverter(0))
}

func TestDriverStatementContextDelegation(t *testing.T) {
	ctx := t.Context()
	failure := errors.New("statement failure")
	args := []driver.NamedValue{{Name: "value", Ordinal: 1, Value: "uuid-text"}}
	calls := []string{}
	underlying := &driverContextStmtStub{
		execContext: func(received context.Context, receivedArgs []driver.NamedValue) (driver.Result, error) {
			calls = append(calls, "exec")
			assert.Same(t, ctx, received)
			assert.Equal(t, args, receivedArgs)
			return driver.RowsAffected(4), failure
		},
		queryContext: func(received context.Context, receivedArgs []driver.NamedValue) (driver.Rows, error) {
			calls = append(calls, "query")
			assert.Same(t, ctx, received)
			assert.Equal(t, args, receivedArgs)
			return nil, failure
		},
	}
	statement := &_Stmt{Stmt: underlying}
	result, err := statement.ExecContext(ctx, args)
	assert.Equal(t, driver.RowsAffected(4), result)
	assert.ErrorIs(t, err, failure)
	_, err = statement.QueryContext(ctx, args)
	assert.ErrorIs(t, err, failure)
	assert.Equal(t, []string{"exec", "query"}, calls)
}
