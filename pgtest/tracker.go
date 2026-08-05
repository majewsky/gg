// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package pgtest contains test assertions for checking the contents of PostgreSQL databases.
package pgtest

import (
	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
)

// Tracker keeps a copy of the database contents and allows for checking the database contents (or changes made to them) during tests.
type Tracker struct {
	t   assert.TestingTB
	dbh gsql.Handle
	s   Snapshot
}

// NewTracker creates a new Tracker.
//
// The initial creation involves taking a snapshot, which is returned as a second value.
// This is an optimization, since it is often desirable to assert on the full DB contents when creating the tracker.
// Calling [Tracker.DBContent] directly after [NewTracker] would take a superfluous second snapshot.
func NewTracker(t assert.TestingTB, db gsql.Handle) (*Tracker, Snapshot) {
	panic("TODO")
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
