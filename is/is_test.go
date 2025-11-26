// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package is_test

import (
	"testing"
	"time"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/is"
	. "github.com/majewsky/gg/option"
)

func TestComparable(t *testing.T) {
	AssertEqual(t, Some("foo").IsSomeAnd(is.EqualTo("foo")), true)
	AssertEqual(t, Some("bar").IsSomeAnd(is.EqualTo("foo")), false)

	AssertEqual(t, Some("foo").IsSomeAnd(is.DifferentFrom("foo")), false)
	AssertEqual(t, Some("bar").IsSomeAnd(is.DifferentFrom("foo")), true)
}

func TestOrdered(t *testing.T) {
	AssertEqual(t, Some(3).IsSomeAnd(is.Above(4)), false)
	AssertEqual(t, Some(4).IsSomeAnd(is.Above(4)), false)
	AssertEqual(t, Some(5).IsSomeAnd(is.Above(4)), true)

	AssertEqual(t, Some(3).IsSomeAnd(is.Below(4)), true)
	AssertEqual(t, Some(4).IsSomeAnd(is.Below(4)), false)
	AssertEqual(t, Some(5).IsSomeAnd(is.Below(4)), false)

	AssertEqual(t, Some(3).IsSomeAnd(is.NotAbove(4)), true)
	AssertEqual(t, Some(4).IsSomeAnd(is.NotAbove(4)), true)
	AssertEqual(t, Some(5).IsSomeAnd(is.NotAbove(4)), false)

	AssertEqual(t, Some(3).IsSomeAnd(is.NotBelow(4)), false)
	AssertEqual(t, Some(4).IsSomeAnd(is.NotBelow(4)), true)
	AssertEqual(t, Some(5).IsSomeAnd(is.NotBelow(4)), true)
}

func TestTime(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Second)
	t3 := t2.Add(time.Second)

	AssertEqual(t, Some(t1).IsSomeAnd(is.After(t2)), false)
	AssertEqual(t, Some(t2).IsSomeAnd(is.After(t2)), false)
	AssertEqual(t, Some(t3).IsSomeAnd(is.After(t2)), true)

	AssertEqual(t, Some(t1).IsSomeAnd(is.Before(t2)), true)
	AssertEqual(t, Some(t2).IsSomeAnd(is.Before(t2)), false)
	AssertEqual(t, Some(t3).IsSomeAnd(is.Before(t2)), false)

	AssertEqual(t, Some(t1).IsSomeAnd(is.NotAfter(t2)), true)
	AssertEqual(t, Some(t2).IsSomeAnd(is.NotAfter(t2)), true)
	AssertEqual(t, Some(t3).IsSomeAnd(is.NotAfter(t2)), false)

	AssertEqual(t, Some(t1).IsSomeAnd(is.NotBefore(t2)), false)
	AssertEqual(t, Some(t2).IsSomeAnd(is.NotBefore(t2)), true)
	AssertEqual(t, Some(t3).IsSomeAnd(is.NotBefore(t2)), true)
}
