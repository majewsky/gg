// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

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

func TestMax(t *testing.T) {
	AssertEqual(t, Max[int](), None[int]())
	AssertEqual(t, Max(None[int]()), None[int]())
	AssertEqual(t, Max(None[int](), None[int]()), None[int]())

	AssertEqual(t, Max(Some(5)), Some(5))
	AssertEqual(t, Max(Some(5), None[int]()), Some(5))
	AssertEqual(t, Max(None[int](), Some(5)), Some(5))

	AssertEqual(t, Max(Some(5), Some(23)), Some(23))
	AssertEqual(t, Max(None[int](), Some(5), Some(23)), Some(23))
	AssertEqual(t, Max(Some(5), None[int](), Some(23)), Some(23))
	AssertEqual(t, Max(Some(5), Some(23), None[int]()), Some(23))

	AssertEqual(t, Max(Some(23), Some(5)), Some(23))
	AssertEqual(t, Max(None[int](), Some(23), Some(5)), Some(23))
	AssertEqual(t, Max(Some(23), None[int](), Some(5)), Some(23))
	AssertEqual(t, Max(Some(23), Some(5), None[int]()), Some(23))
}

func TestMin(t *testing.T) {
	AssertEqual(t, Min[int](), None[int]())
	AssertEqual(t, Min(None[int]()), None[int]())
	AssertEqual(t, Min(None[int](), None[int]()), None[int]())

	AssertEqual(t, Min(Some(5)), Some(5))
	AssertEqual(t, Min(Some(5), None[int]()), Some(5))
	AssertEqual(t, Min(None[int](), Some(5)), Some(5))

	AssertEqual(t, Min(Some(5), Some(23)), Some(5))
	AssertEqual(t, Min(None[int](), Some(5), Some(23)), Some(5))
	AssertEqual(t, Min(Some(5), None[int](), Some(23)), Some(5))
	AssertEqual(t, Min(Some(5), Some(23), None[int]()), Some(5))

	AssertEqual(t, Min(Some(23), Some(5)), Some(5))
	AssertEqual(t, Min(None[int](), Some(23), Some(5)), Some(5))
	AssertEqual(t, Min(Some(23), None[int](), Some(5)), Some(5))
	AssertEqual(t, Min(Some(23), Some(5), None[int]()), Some(5))
}
