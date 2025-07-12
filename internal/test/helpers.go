/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package option

import (
	"reflect"
	"testing"
)

func AssertEqual[V any](t *testing.T, actual, expected V) {
	if reflect.DeepEqual(actual, expected) {
		return
	}
	t.Helper()
	t.Errorf("expected %#v, but got %#v", expected, actual)
}

func AssertErrorEqual(t *testing.T, actual error, expected string) {
	t.Helper()
	if actual == nil {
		t.Errorf("expected err = %q, but got nil", expected)
	} else if actual.Error() != expected {
		t.Errorf("expected err = %q, but got %q", expected, actual.Error())
	}
}

func AssertPanics(t *testing.T, action func()) (recovered any) {
	t.Helper()
	defer func() {
		recovered = recover()
		if recovered == nil {
			t.Error("action did not panic as expected")
		}
	}()
	action()
	return
}

func PointerTo[V any](value V) *V {
	return &value
}
