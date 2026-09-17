// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package must

import "testing"

// Succeed fails the test if err is not nil.
func Succeed(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err.Error())
	}
}

// Return wraps a function returning two output values,
// and either forwards the result value on success, or fails the test on error.
func Return[V any](value V, err error) func(testing.TB) V {
	return func(t testing.TB) V {
		t.Helper()
		if err != nil {
			t.Fatal(err.Error())
		}
		return value
	}
}
