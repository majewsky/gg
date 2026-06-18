// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"strconv"
	"testing"

	"go.xyrillian.de/gg/assert"
	. "go.xyrillian.de/gg/option"
)

func TestFromPointer(t *testing.T) {
	assert.Equal(t, FromPointer[int](nil), None[int]())
	assert.Equal(t, FromPointer(new(int(42))), Some(42))
}

func TestIsNoneOrZero(t *testing.T) {
	assert.Equal(t, IsNoneOrZero(None[int]()), true)
	assert.Equal(t, IsNoneOrZero(Some(0)), true)
	assert.Equal(t, IsNoneOrZero(Some(1)), false)
}

func TestMap(t *testing.T) {
	assert.Equal(t, Map(None[int](), strconv.Itoa), None[string]())
	assert.Equal(t, Map(Some(42), strconv.Itoa), Some("42"))
}

func TestMax(t *testing.T) {
	assert.Equal(t, Max[int](), None[int]())
	assert.Equal(t, Max(None[int]()), None[int]())
	assert.Equal(t, Max(None[int](), None[int]()), None[int]())

	assert.Equal(t, Max(Some(5)), Some(5))
	assert.Equal(t, Max(Some(5), None[int]()), Some(5))
	assert.Equal(t, Max(None[int](), Some(5)), Some(5))

	assert.Equal(t, Max(Some(5), Some(23)), Some(23))
	assert.Equal(t, Max(None[int](), Some(5), Some(23)), Some(23))
	assert.Equal(t, Max(Some(5), None[int](), Some(23)), Some(23))
	assert.Equal(t, Max(Some(5), Some(23), None[int]()), Some(23))

	assert.Equal(t, Max(Some(23), Some(5)), Some(23))
	assert.Equal(t, Max(None[int](), Some(23), Some(5)), Some(23))
	assert.Equal(t, Max(Some(23), None[int](), Some(5)), Some(23))
	assert.Equal(t, Max(Some(23), Some(5), None[int]()), Some(23))
}

func TestMin(t *testing.T) {
	assert.Equal(t, Min[int](), None[int]())
	assert.Equal(t, Min(None[int]()), None[int]())
	assert.Equal(t, Min(None[int](), None[int]()), None[int]())

	assert.Equal(t, Min(Some(5)), Some(5))
	assert.Equal(t, Min(Some(5), None[int]()), Some(5))
	assert.Equal(t, Min(None[int](), Some(5)), Some(5))

	assert.Equal(t, Min(Some(5), Some(23)), Some(5))
	assert.Equal(t, Min(None[int](), Some(5), Some(23)), Some(5))
	assert.Equal(t, Min(Some(5), None[int](), Some(23)), Some(5))
	assert.Equal(t, Min(Some(5), Some(23), None[int]()), Some(5))

	assert.Equal(t, Min(Some(23), Some(5)), Some(5))
	assert.Equal(t, Min(None[int](), Some(23), Some(5)), Some(5))
	assert.Equal(t, Min(Some(23), None[int](), Some(5)), Some(5))
	assert.Equal(t, Min(Some(23), Some(5), None[int]()), Some(5))
}
