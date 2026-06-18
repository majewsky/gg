// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert

import "reflect"

// Equal checks whether both supplied values are equal according to the rules of [reflect.DeepEqual].
func Equal[V any](t TestingTB, actual, expected V) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}
	t.Helper()
	t.Errorf("expected %#v, but got %#v", expected, actual)
	return false
}
