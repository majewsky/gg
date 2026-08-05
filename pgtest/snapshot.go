// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgtest

import (
	"fmt"

	"go.xyrillian.de/gg/assert"
)

// Snapshot contains a set of SQL statements.
// Instances are produced by methods of [Tracker].
type Snapshot struct {
	t assert.TestingTB
}

// AssertEmpty is a shorthand for AssertEqual("").
func (s Snapshot) AssertEmpty() {
	s.t.Helper()
	s.AssertEqual("")
}

// AssertEqual compares the set of SQL statements to those in the given string literal.
// A test error is generated for each difference.
//
// This assertion is lenient with regards to whitespace to enable callers
// to format their string literals in a way that fits nicely in the surrounding code:
//   - Leading whitespace on each line is ignored.
//   - Empty lines (containing only whitespace) are ignored.
func (s Snapshot) AssertEqual(expected string) {
	s.t.Helper()
	panic("TODO")
}

// AssertEqualf is a shorthand for AssertEqual(fmt.Sprintf(format, args...)).
func (s Snapshot) AssertEqualf(format string, args ...any) {
	s.t.Helper()
	s.AssertEqual(fmt.Sprintf(format, args...))
}

// Ignore is a no-op. It is commonly used like `tr.DBChanges().Ignore()`,
// to clarify that a certain set of DB changes is not asserted on.
func (s Snapshot) Ignore() {
	// intentionally empty
}
