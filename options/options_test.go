/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package options

import (
	"strconv"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	. "github.com/majewsky/gg/option"
)

func TestFromPointer(t *testing.T) {
	AssertEqual(t, FromPointer[int](nil), None[int]())
	AssertEqual(t, FromPointer(PointerTo[int](42)), Some(42))
}

func TestIsNoneOrZero(t *testing.T) {
	AssertEqual(t, IsNoneOrZero(None[int]()), true)
	AssertEqual(t, IsNoneOrZero(Some(0)), true)
	AssertEqual(t, IsNoneOrZero(Some(1)), false)
}

func TestMap(t *testing.T) {
	AssertEqual(t, Map(None[int](), strconv.Itoa), None[string]())
	AssertEqual(t, Map(Some(42), strconv.Itoa), Some("42"))
}
