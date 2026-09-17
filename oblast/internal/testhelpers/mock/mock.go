// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package mock

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////
// type Driver

// Driver is a mock SQL driver that only accepts queries that were preannounced.
type Driver struct {
	responseSetsByQuery map[string]*ResponseSet
}

// assert that interface is implemented
var _ driver.Connector = &Driver{}

// NewDriver instantiates a new driver.
// The result returns [driver.Connector] and can be given to [sql.OpenDB].
func NewDriver() *Driver {
	return &Driver{
		responseSetsByQuery: make(map[string]*ResponseSet),
	}
}

// Connect implements the [driver.Connector] interface.
func (d *Driver) Connect(ctx context.Context) (driver.Conn, error) {
	return &connection{d: d}, nil
}

// Driver implements the [driver.Connector] interface.
func (d *Driver) Driver() driver.Driver {
	// Not needed. Implementing the Driver interface would only be necessary if
	// we wanted to use sql.Open() instead of sql.OpenDB(), or if we wanted to
	// use sql.DB.Driver().
	panic("unimplemented")
}

// ForQuery tells the driver to expect the given query string to be sent soon.
// The return value can be used to plan what to return when the query is actually executed.
func (d *Driver) ForQuery(query string) *ResponseSet {
	if d.responseSetsByQuery[query] == nil {
		d.responseSetsByQuery[query] = &ResponseSet{}
	}
	return d.responseSetsByQuery[query]
}

////////////////////////////////////////////////////////////////////////////////
// type ResponseSet

// ResponseSet is a set of mock responses for a query sent to type [Driver].
type ResponseSet struct {
	expectedExecs   []expectation[Result]
	expectedQueries []expectation[Rows]
}

type expectation[T any] struct {
	args   []driver.Value
	output *T
}

func newExpectation[T any](args []any) expectation[T] {
	e := expectation[T]{
		args:   make([]driver.Value, len(args)),
		output: new(T),
	}
	for idx, arg := range args {
		var err error
		e.args[idx], err = driver.DefaultParameterConverter.ConvertValue(arg)
		if err != nil {
			panic(fmt.Sprintf("could not convert value %#v into driver.Value: %s", arg, err.Error()))
		}
	}
	return e
}

// ExpectExecWithArgs plans a response to an Exec() call.
func (rs *ResponseSet) ExpectExecWithArgs(args ...any) *Result {
	e := newExpectation[Result](args)
	rs.expectedExecs = append(rs.expectedExecs, e)
	return e.output
}

// ExpectQueryWithArgs plans a response to a Query() or QueryRows() call.
func (rs *ResponseSet) ExpectQueryWithArgs(args ...any) *Rows {
	e := newExpectation[Rows](args)
	rs.expectedQueries = append(rs.expectedQueries, e)
	return e.output
}

////////////////////////////////////////////////////////////////////////////////
// type connection

type connection struct {
	d      *Driver
	closed bool
}

// Prepare implements the [driver.Conn] interface.
func (c *connection) Prepare(query string) (driver.Stmt, error) {
	rs := c.d.responseSetsByQuery[query]
	if rs == nil {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	return &statement{c: c, query: query, rs: rs}, nil
}

// Close implements the [driver.Conn] interface.
func (c *connection) Close() error {
	c.closed = true
	return nil
}

// Begin implements the [driver.Conn] interface.
func (c *connection) Begin() (driver.Tx, error) {
	return transaction{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// type transaction

type transaction struct{}

// Commit implements the [driver.Tx] interface.
func (t transaction) Commit() error {
	return nil // unused
}

// Rollback implements the [driver.Tx] interface.
func (t transaction) Rollback() error {
	return nil // unused
}

////////////////////////////////////////////////////////////////////////////////
// type statement

type statement struct {
	c      *connection
	query  string
	rs     *ResponseSet
	closed bool
}

// Close implements the [driver.Stmt] interface.
func (s *statement) Close() error {
	return nil
}

// NumInput implements the [driver.Stmt] interface.
func (s *statement) NumInput() int {
	// option 1: when using SQLite dialect, count `?`
	count := strings.Count(s.query, "?")
	if count > 0 {
		return count
	}

	// option 2: when using PostgreSQL dialect, find `$1`, `$2`, etc.
	for strings.Contains(s.query, fmt.Sprintf("$%d", count+1)) {
		count++
	}
	return count
}

// Exec implements the [driver.Stmt] interface.
func (s *statement) Exec(args []driver.Value) (driver.Result, error) {
	if s.closed {
		return nil, errors.New("statement was closed")
	}
	if s.c.closed {
		return nil, errors.New("connection was closed")
	}
	for idx, e := range s.rs.expectedExecs {
		if reflect.DeepEqual(e.args, args) {
			s.rs.expectedExecs = slices.Delete(s.rs.expectedExecs, idx, idx+1)
			return result{r: *e.output}, nil
		}
	}
	return nil, fmt.Errorf("unexpected arguments for query %q: %#v", s.query, args)
}

// Query implements the [driver.Stmt] interface.
func (s *statement) Query(args []driver.Value) (driver.Rows, error) {
	if s.closed {
		return nil, errors.New("statement was closed")
	}
	if s.c.closed {
		return nil, errors.New("connection was closed")
	}
	for idx, e := range s.rs.expectedQueries {
		if reflect.DeepEqual(e.args, args) {
			s.rs.expectedQueries = slices.Delete(s.rs.expectedQueries, idx, idx+1)
			return &rows{r: *e.output}, nil
		}
	}
	return nil, fmt.Errorf("unexpected arguments for query %q: %#v", s.query, args)
}

///////////////////////////////////////////////////////////////////////////////////////////
// type Result

// Result is a mock response for an Exec() call.
// It is constructed by [ResponseSet.ExpectExec].
type Result struct {
	lastInsertId *int64
	rowsAffected *int64
}

// AndReturnLastInsertId configures a mock LastInsertId() value for this Result.
// Returns the same Result instance to allow chaining additional method calls.
func (r *Result) AndReturnLastInsertId(id int64) *Result {
	r.lastInsertId = &id
	return r
}

// AndReturnRowsAffected configures a mock RowsAffected() value for this Result.
// Returns the same Result instance to allow chaining additional method calls.
func (r *Result) AndReturnRowsAffected(count int64) *Result {
	r.rowsAffected = &count
	return r
}

type result struct {
	r Result
}

// LastInsertId implements the [driver.Result] interface.
func (r result) LastInsertId() (int64, error) {
	if r.r.lastInsertId == nil {
		return 0, errors.New("AndReturnLastInsertId() was not called for this Result")
	}
	return *r.r.lastInsertId, nil
}

// RowsAffected implements the [driver.Result] interface.
func (r result) RowsAffected() (int64, error) {
	if r.r.rowsAffected == nil {
		return 0, errors.New("AndReturnRowsAffected() was not called for this Result")
	}
	return *r.r.rowsAffected, nil
}

// /////////////////////////////////////////////////////////////////////////////////////////
// type Rows

// Rows is a mock response for a Query() or QueryRow() call.
// It is constructed by [ResponseSet.ExpectQuery].
type Rows struct {
	columns    []string
	results    [][]any
	closeError error
}

// AndReturnColumns configures the set of column names that will be returned by this query.
// Returns the same Result instance to allow chaining additional method calls.
func (r *Rows) AndReturnColumns(columns ...string) *Rows {
	if len(r.columns) > 0 {
		panic("AndReturnColumns() called multiple times for the same Rows object")
	}
	r.columns = columns
	return r
}

// WithRow adds a row to the result set that will be returned by this query.
// This may only be called after AndReturnColumns().
func (r *Rows) WithRow(values ...any) *Rows {
	if len(r.columns) == 0 {
		panic("AndReturnColumns() has not been called for this Rows object yet")
	}
	if len(r.columns) != len(values) {
		panic("WithRow() must be called with the same number of args as the preceding AndReturnColumns() call")
	}
	r.results = append(r.results, values)
	return r
}

// AndCloseFailsWith sets up Close() for this Rows to fail with the provided error message.
func (r *Rows) AndCloseFailsWith(err error) {
	r.closeError = err
}

type rows struct {
	r      Rows
	closed bool
}

// Columns implements the [driver.Rows] interface.
func (r *rows) Columns() []string {
	return r.r.columns
}

// Close implements the [driver.Rows] interface.
func (r *rows) Close() error {
	r.closed = true
	return r.r.closeError
}

// Next implements the [driver.Rows] interface.
func (r *rows) Next(dest []driver.Value) error {
	if r.closed {
		return errors.New("rows object was closed")
	}
	if len(r.r.results) == 0 {
		return io.EOF
	}
	for idx, value := range r.r.results[0] {
		dest[idx] = value
	}
	r.r.results = r.r.results[1:]
	return nil
}
