// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package pgtest contains test assertions for checking the contents of PostgreSQL databases.
package pgtest

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
)

// Tracker keeps a copy of the database contents and allows for checking the database contents (or changes made to them) during tests.
type Tracker struct {
	t   assert.TestingTB
	dbh gsql.Handle
	s   Snapshot

	topo topology
}

// NewTracker creates a new Tracker.
//
// The initial creation involves taking a snapshot, which is returned as a second value.
// This is an optimization, since it is often desirable to assert on the full DB contents when creating the tracker.
// Calling [Tracker.DBContent] directly after [NewTracker] would take a superfluous second snapshot.
func NewTracker(t assert.TestingTB, db gsql.Handle) (*Tracker, Snapshot) {
	ctx := t.Context()
	t.Helper()

	topo, err := newTopology(ctx, db)
	if err != nil {
		t.Fatal(err.Error())
	}
	s, err := newSnapshot(ctx, db, topo)
	if err != nil {
		t.Fatal(err.Error())
	}
	return &Tracker{t, db, s, topo}, s
}

// DBChanges produces a diff of the current database contents against the state at the last Tracker call,
// as a set of INSERT/UPDATE/DELETE statements on which test assertions can be executed.
func (t *Tracker) DBChanges() Snapshot {
	panic("TODO")
}

// DBContent produces a dump of the current database contents,
// as a sequence of INSERT statements on which test assertions can be executed.
func (t *Tracker) DBContent() Snapshot {
	panic("TODO")
}

// sqlLiteral implements [sql.Scanner] by storing a representation of the captured value as an SQL literal.
// For time.Time, the UNIX timestamp is stored instead.
type sqlLiteral string

// Scan implements the [sql.Scanner] interface.
func (l *sqlLiteral) Scan(src any) error {
	switch src := src.(type) {
	case int64:
		*l = sqlLiteral(strconv.FormatInt(src, 10))
	case float64:
		*l = sqlLiteral(fmt.Sprintf("%g", src))
	case bool:
		if src {
			*l = "TRUE"
		} else {
			*l = "FALSE"
		}
	case []byte:
		*l = makeSQLStringLiteral(string(src))
	case string:
		*l = makeSQLStringLiteral(src)
	case time.Time:
		*l = sqlLiteral(strconv.FormatInt(src.Unix(), 10))
	case nil:
		*l = "NULL"
	default:
		return fmt.Errorf("sqlLiteral.Scan(): do not know how to serialize type %T", src)
	}
	return nil
}

func makeSQLStringLiteral(in string) sqlLiteral {
	return sqlLiteral("'" + strings.ReplaceAll(in, "'", "''") + "'")
}
