// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package option

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"go.xyrillian.de/gg/assert"
)

func TestZeroValue(t *testing.T) {
	var zero Option[string]
	assert.Equal(t, zero, None[string]())
}

////////////////////////////////////////////////////////////////////////////////
// core API (methods sorted by name)

func TestAsPointer(t *testing.T) {
	assert.Equal(t, None[int]().AsPointer(), nil)
	assert.Equal(t, Some(42).AsPointer(), new(42))
}

func TestAsSlice(t *testing.T) {
	assert.Equal(t, None[int]().AsSlice(), []int(nil))
	assert.Equal(t, Some(42).AsSlice(), []int{42})
}

func isEven(x int) bool {
	// an example predicate for testing the various functions that take predicates
	return x%2 == 0
}

func TestFilter(t *testing.T) {
	assert.Equal(t, None[int]().Filter(isEven), None[int]())
	assert.Equal(t, Some(41).Filter(isEven), None[int]())
	assert.Equal(t, Some(42).Filter(isEven), Some(42))
}

func TestIsNone(t *testing.T) {
	assert.Equal(t, None[int]().IsNone(), true)
	assert.Equal(t, Some(42).IsNone(), false)
}

func TestIsNoneOr(t *testing.T) {
	assert.Equal(t, None[int]().IsNoneOr(isEven), true)
	assert.Equal(t, Some(41).IsNoneOr(isEven), false)
	assert.Equal(t, Some(42).IsNoneOr(isEven), true)
}

func TestIsSome(t *testing.T) {
	assert.Equal(t, None[int]().IsSome(), false)
	assert.Equal(t, Some(42).IsSome(), true)
}

func TestIsSomeAnd(t *testing.T) {
	assert.Equal(t, None[int]().IsSomeAnd(isEven), false)
	assert.Equal(t, Some(41).IsSomeAnd(isEven), false)
	assert.Equal(t, Some(42).IsSomeAnd(isEven), true)
}

func TestIsZero(t *testing.T) {
	assert.Equal(t, None[int]().IsZero(), true)
	assert.Equal(t, Some(42).IsZero(), false)
}

func TestIter(t *testing.T) {
	assert.Equal(t, slices.Collect(None[int]().Iter()), []int(nil))
	assert.Equal(t, slices.Collect(Some(42).Iter()), []int{42})
}

func TestOr(t *testing.T) {
	none := None[int]()
	assert.Equal(t, none.Or(none), None[int]())
	assert.Equal(t, Some(42).Or(none), Some(42))
	assert.Equal(t, none.Or(Some(43)), Some(43))
	assert.Equal(t, Some(42).Or(Some(43)), Some(42))
}

func TestOrElse(t *testing.T) {
	callCount := 0
	makeNone := func() Option[int] {
		callCount++
		return None[int]()
	}
	makeSome := func() Option[int] {
		callCount++
		return Some(43)
	}

	assert.Equal(t, None[int]().OrElse(makeNone), None[int]())
	assert.Equal(t, callCount, 1)
	assert.Equal(t, Some(42).OrElse(makeNone), Some(42))
	assert.Equal(t, callCount, 1)
	assert.Equal(t, None[int]().OrElse(makeSome), Some(43))
	assert.Equal(t, callCount, 2)
	assert.Equal(t, Some(42).OrElse(makeSome), Some(42))
	assert.Equal(t, callCount, 2)
}

func TestUnpack(t *testing.T) {
	val, ok := None[int]().Unpack()
	assert.Equal(t, val, 0)
	assert.Equal(t, ok, false)

	val, ok = Some(42).Unpack()
	assert.Equal(t, val, 42)
	assert.Equal(t, ok, true)
}

func TestUnwrapOr(t *testing.T) {
	assert.Equal(t, None[int]().UnwrapOr(5), 5)
	assert.Equal(t, Some(42).UnwrapOr(5), 42)
}

func TestUnwrapOrElse(t *testing.T) {
	callCount := 0
	five := func() int {
		callCount++
		return 5
	}

	assert.Equal(t, None[int]().UnwrapOrElse(five), 5)
	assert.Equal(t, callCount, 1)
	assert.Equal(t, Some(42).UnwrapOrElse(five), 42)
	assert.Equal(t, callCount, 1)
}

func TestXor(t *testing.T) {
	none := None[int]()
	assert.Equal(t, none.Xor(none), None[int]())
	assert.Equal(t, Some(42).Xor(none), Some(42))
	assert.Equal(t, none.Xor(Some(43)), Some(43))
	assert.Equal(t, Some(42).Xor(Some(43)), None[int]())
}

////////////////////////////////////////////////////////////////////////////////
// formatting/marshalling support

func TestFormat(t *testing.T) {
	none := None[int]()
	some := Some(42)

	assert.Equal(t, fmt.Sprintf("value is %d", none), "value is <none>")
	assert.Equal(t, fmt.Sprintf("value is %d", some), "value is 42")
	assert.Equal(t, fmt.Sprintf("value is %010d", none), "value is 0000<none>")
	assert.Equal(t, fmt.Sprintf("value is %010d", some), "value is 0000000042")

	noneList := None[[]int]()
	someList := Some([]int{4, 2})
	assert.Equal(t, fmt.Sprintf("value is %v", noneList), "value is <none>")
	assert.Equal(t, fmt.Sprintf("value is %v", someList), "value is [4 2]")
	assert.Equal(t, fmt.Sprintf("value is %#v", noneList), "value is <none>")
	assert.Equal(t, fmt.Sprintf("value is %#v", someList), "value is []int{4, 2}")

	listOfOptions := []Option[int]{none, some}
	assert.Equal(t, fmt.Sprintf("value is %#v", listOfOptions), "value is []option.Option[int]{<none>, 42}")
}

func TestMarshalSQL(t *testing.T) {
	value, err := None[string]().Value()
	assert.Equal(t, err, nil)
	assert.Equal(t, value, nil)

	value, err = Some("hello").Value()
	assert.Equal(t, err, nil)
	assert.Equal(t, value, "hello")

	_, err = Some(struct{}{}).Value()
	assert.Equal(t, err.Error(), "unsupported type struct {}, a struct")
}

func TestUnmarshalSQL(t *testing.T) {
	var o1 Option[string]
	err := o1.Scan(nil)
	assert.Equal(t, err, nil)
	assert.Equal(t, o1, None[string]())

	var o2 Option[string]
	err = o2.Scan("hello")
	assert.Equal(t, err, nil)
	assert.Equal(t, o2, Some("hello"))

	var o3 Option[struct{}]
	err = o3.Scan("hello")
	assert.Equal(t, err.Error(), "unsupported Scan, storing driver.Value type string into type *struct {}")
}

func TestMarshalAndUnmarshalJSON(t *testing.T) {
	type payload struct {
		N1 Option[int] `json:"n1"`
		N2 Option[int] `json:"n2,omitempty"`
		N3 Option[int] `json:"n3,omitzero"`
		S1 Option[int] `json:"s1"`
		S2 Option[int] `json:"s2,omitempty"`
		S3 Option[int] `json:"s3,omitzero"`
	}
	original := payload{
		N1: None[int](),
		N2: None[int](),
		N3: None[int](),
		S1: Some(1),
		S2: Some(2),
		S3: Some(3),
	}
	buf, err := json.Marshal(original)
	assert.Equal(t, err, nil)
	assert.Equal(t, string(buf), `{"n1":null,"n2":null,"s1":1,"s2":2,"s3":3}`)

	var decoded payload
	err = json.Unmarshal(buf, &decoded)
	assert.Equal(t, err, nil)
	assert.Equal(t, decoded, original)
}
