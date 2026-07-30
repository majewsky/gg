// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/errext"
	"go.xyrillian.de/gg/gsql"
)

// Connector describes how to connect to a PostgreSQL database given a [libpq-style connection URI].
// This type acts as a dependency injection surface, abstracting the different ways in which different database drivers and libraries perform connection.
//
// [libpq-style connection URI]: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-URIS
type Connector[T gsql.ConnectionHandle] func(ctx context.Context, dbURL string) (T, error)

// StdConnector returns a [Connector] for database/sql drivers.
// When used with [lib/pq], the driver name must be "postgres".
func StdConnector(driverName string) Connector[*gsql.DB] {
	return func(ctx context.Context, dbURL string) (*gsql.DB, error) {
		db, err := sql.Open(driverName, dbURL)
		if err != nil {
			return nil, err
		}
		return gsql.NewDB(db), nil
	}
}

// Connect connects to a PostgreSQL database.
func (c Connector[T]) Connect(ctx context.Context, target ConnectionTarget, behavior ConnectionBehavior) (T, error) {
	var none T // shorthand for error return paths

	u, err := target.IntoURL()
	if err != nil {
		return none, err
	}
	db, err := c(ctx, u.String())
	if err != nil {
		return none, err
	}
	err = behavior.applyTo(ctx, db)
	if err != nil {
		return none, errext.WithCleanup(err, "db.Close", db.GSQLClose(ctx))
	}
	return db, nil
}

// ConnectForTest connects to the test database server managed by [WithTestDB].
//
// Each test will run in its own separate database (whose name is the same as t.Name()),
// so it is safe to mark tests as t.Parallel() to run multiple tests within the same package concurrently.
// ConnectForTest will always drop and recreate the database to ensure deterministic test behavior.
//
// Some tests require setting up multiple separate connections to the same database.
// The second return argument can be be used with [Connector.Connect] to obtain additional connections as needed.
func (c Connector[T]) ConnectForTest(t assert.TestingTB, behavior ConnectionBehavior) (T, ConnectionTarget) {
	ctx := t.Context()

	// normalize t.Name() into an acceptable database name for PostgreSQL
	//   - only alphanumerics and underscore -> replace all other symbols with _
	//   - max 63 chars -> reject longer names
	dbName := strings.ToLower(t.Name())
	dbName = regexp.MustCompile(`[^a-z_]`).ReplaceAllString(dbName, "_")
	if len(dbName) > 63 {
		t.Fatalf("cannot use t.Name() = %q (normalized to %q) as a database name because it is longer than 63 chars", t.Name(), dbName)
	}

	// connect to "postgres" database for the DROP/CREATE DATABASE queries
	target := ConnectionTarget{
		HostName:          "127.0.0.1",
		Port:              testdbPort,
		UserName:          testdbUserName,
		DatabaseName:      "postgres",
		ConnectionOptions: "sslmode=disable",
	}
	err := c.prepareTestDatabase(ctx, target, dbName)
	if err != nil {
		t.Fatal(err.Error())
	}

	// connect to actual test database
	target.DatabaseName = dbName
	handle, err := c.Connect(ctx, target, behavior)
	if err != nil {
		t.Fatal(err.Error())
	}
	return handle, target
}

func (c Connector[T]) prepareTestDatabase(ctx context.Context, target ConnectionTarget, dbName string) error {
	u, err := target.IntoURL()
	if err != nil {
		return err
	}
	db, err := c(ctx, u.String())
	if err != nil {
		return fmt.Errorf("%w (if this error is about the database server not running, check if your TestMain() calls pgruntime.WithTestDB())", err)
	}
	_, err = execQuery(ctx, db, "DROP DATABASE IF EXISTS "+quoteIdentifier(dbName), nil)
	if err != nil {
		return errext.WithCleanup(err, "db.Close", db.GSQLClose(ctx))
	}
	_, err = execQuery(ctx, db, "CREATE DATABASE "+quoteIdentifier(dbName), nil)
	return errext.WithCleanup(err, "db.Close", db.GSQLClose(ctx))
}
