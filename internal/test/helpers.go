/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package test

import (
	"reflect"
	"testing"
)

func AssertEqual[V any](t *testing.T, actual, expected V) bool {
	if reflect.DeepEqual(actual, expected) {
		return true
	}
	t.Helper()
	t.Errorf("expected %#v, but got %#v", expected, actual)
	return false
}

func PointerTo[V any](value V) *V {
	return &value
}
