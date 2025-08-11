// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package jsonmatch_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/jsonmatch"
	. "github.com/majewsky/gg/option"
)

// assert that types implement the expected interfaces
// (CaptureField needs dynamic casts because CaptureField() returns `any`)
var (
	_ jsonmatch.Diffable = jsonmatch.Array{}
	_ jsonmatch.Diffable = jsonmatch.Object{}
	_ jsonmatch.Diffable = jsonmatch.Null()
	_ jsonmatch.Diffable = jsonmatch.Scalar("foo")
	_ jsonmatch.Diffable = jsonmatch.Scalar(42)
	_ jsonmatch.Diffable = jsonmatch.Scalar(false)
	_                    = jsonmatch.CaptureField(Some(1).AsPointer()).(json.Marshaler)
	_                    = jsonmatch.CaptureField(Some(1).AsPointer()).(json.Unmarshaler)
)

func TestCanonicalizesActualPayload(t *testing.T) {
	testCases := [][]byte{
		// all of these are functionally identical, so they should produce an empty diff
		// against our expectations regardless of key order and whitespace
		[]byte(`{"data": {"qux":[5,null,15], "foo": 42, "bar": "hello world"}}`),
		[]byte(`{"data":{"bar":"hello world","foo":42,"qux":[5,null,15]}}`),
		[]byte(`{
			"data": {
				"bar": "hello world",
				"qux": [
					5,
					null,
					15
				],
				"foo": 42
			}
		}`),
	}

	for _, message := range testCases {
		t.Logf("message = %q", message)

		// we test with several variants of `expected` using different underlying
		// types that represent identical JSON payloads, but in different ways
		match := jsonmatch.Object{
			"data": jsonmatch.Object{
				"foo": 42,
				"bar": "hello world",
				"qux": []any{5, nil, 15},
			},
		}
		AssertEqual(t, match.DiffAgainst(message), nil)

		// changing the type of `data` to map[string]any does not change anything at all;
		// using the jsonmatch.Object name on this level is mostly syntactic sugar to communicate intent
		match = jsonmatch.Object{
			"data": map[string]any{
				"foo": 42,
				"bar": "hello world",
				"qux": []any{5, nil, 15},
			},
		}
		AssertEqual(t, match.DiffAgainst(message), nil)

		// this is using subtypes that our logic cannot recurse into
		// (map[opaqueString]any instead of map[string]any and []Option[int] instead of []any);
		// comparison will be less granular and only be able to fail on the level of the opaque subtype, but it will still work
		type opaqueString string
		match = jsonmatch.Object{
			"data": map[opaqueString]any{
				"foo": 42,
				"bar": "hello world",
				"qux": []Option[int]{Some(5), None[int](), Some(15)},
			},
		}
		AssertEqual(t, match.DiffAgainst(message), nil)

		// this is using a specific struct type instead of a map[string]any, which results in a different serialization
		// (map[string]any serializes with keys sorted alphabetically, but structs serialize with keys sorted by field declaration order;
		// jsonmatch knows how to normalize this and thus correctly reports an empty diff because the serializations are identical except for field order)
		match = jsonmatch.Object{
			"data": struct {
				Foo int           `json:"foo"`
				Bar string        `json:"bar"`
				Qux []Option[int] `json:"qux"`
			}{
				Foo: 42,
				Bar: "hello world",
				Qux: []Option[int]{Some(5), None[int](), Some(15)},
			},
		}
		AssertEqual(t, match.DiffAgainst(message), nil)

		// to try and trip up the normalization shown above, this match deliberately contains an unmarshalable object;
		// jsonmatch should recognize that marshaling and unmarshaling does not work and skip the normalization
		match = jsonmatch.Object{
			"data": unmarshalableObject{},
		}
		AssertEqual(t, match.DiffAgainst(message), []jsonmatch.Diff{{
			Kind:         "type mismatch",
			Pointer:      "/data",
			ExpectedJSON: `<not marshalable to JSON, %#v is jsonmatch_test.unmarshalableObject{}>`,
			ActualJSON:   `{"bar":"hello world","foo":42,"qux":[5,null,15]}`,
		}})
	}
}

func TestCapturesFields(t *testing.T) {
	const (
		uuid1 = "2cff2c65-f775-4ed5-8f86-be0998b19781"
		uuid2 = "ce38aa5c-62ed-4367-a2f8-cbe2d73094a8"
	)
	message := fmt.Appendf(nil, `{"objects":[{"id":"%s","tags":["foo"]},{"id":"%s","tags":["bar"]}]}`, uuid1, uuid2)

	// check that CaptureField() works as intended when contained within one of the supported container types
	type opaqueString string
	var (
		capturedUUID1 string
		capturedUUID2 string
		capturedTag1  opaqueString // check that capturing also works for custom types
	)
	match := jsonmatch.Object{
		"objects": []jsonmatch.Object{
			{
				"id":   jsonmatch.CaptureField(&capturedUUID1),
				"tags": []string{"foo"},
			},
			{
				"id":   jsonmatch.CaptureField(&capturedUUID2),
				"tags": []any{jsonmatch.CaptureField(&capturedTag1)},
			},
		},
	}

	AssertEqual(t, match.DiffAgainst(message), nil)
	AssertEqual(t, capturedUUID1, uuid1)
	AssertEqual(t, capturedUUID2, uuid2)
	AssertEqual(t, capturedTag1, "bar")

	// check that CaptureField() complains when unmarshaling JSON messages into incompatible types
	var (
		capturedUUID3 int
	)
	match = jsonmatch.Object{
		"objects": []jsonmatch.Object{
			{
				"id":   jsonmatch.CaptureField(&capturedUUID3),
				"tags": []string{"foo"},
			},
			{
				"id":   uuid2,
				"tags": []string{"bar"},
			},
		},
	}

	AssertEqual(t, match.DiffAgainst(message), []jsonmatch.Diff{{
		Kind:         "cannot unmarshal into capture slot (json: cannot unmarshal string into Go value of type int)",
		Pointer:      "/objects/0/id",
		ExpectedJSON: "<capture slot of type *int>",
		ActualJSON:   fmt.Sprintf("%q", uuid1),
	}})

	// check that CaptureField() does not work when contained within unsupported types
	//
	// This is a restriction that could be lifted in the future, but it would involve using advanced
	// reflection shenanigans that complicate the implementation. The fact that this example uses
	// somewhat contrived types to even be able to place a capture inside another structure shows that
	// this restriction ought not be too problematic in practice.
	capturedUUID1 = "unset"
	capturedUUID2 = "unset"
	capturedTag1 = "unset"
	match = jsonmatch.Object{
		"objects": []struct {
			ID   any   `json:"id"`
			Tags []any `json:"tags"`
		}{
			{
				ID:   jsonmatch.CaptureField(&capturedUUID1),
				Tags: []any{"foo"},
			},
			{
				ID:   jsonmatch.CaptureField(&capturedUUID2),
				Tags: []any{jsonmatch.CaptureField(&capturedTag1)},
			},
		},
	}

	AssertEqual(t, match.DiffAgainst(message), []jsonmatch.Diff{{
		Kind:         "value mismatch",
		Pointer:      "/objects",
		ActualJSON:   fmt.Sprintf(`[{"id":"%s","tags":["foo"]},{"id":"%s","tags":["bar"]}]`, uuid1, uuid2),
		ExpectedJSON: `[{"id":"unset","tags":["foo"]},{"id":"unset","tags":["unset"]}]`,
	}})
}

func TestFailsOnValueMismatch(t *testing.T) {
	message := []byte(`{"users": [
		{"id":23,"name":"Alice","tags":[{"name":"admin"},{"name":"senior"}]},
		{"id":42,"name":"Bob","tags":[{"name":"support"}]}
	]}`)
	match := jsonmatch.Object{
		"users": []map[string]any{ // also side-note, because we did not have it anywhere else, this covers recursion into []map[string]any
			{
				"id":     23,
				"name":   "Alicia",                                      // should be "Alice"
				"status": "fixing stuff",                                // unexpected field
				"tags":   []jsonmatch.Object{{"name": "administrator"}}, // name should be "admin"; second list entry missing
			},
			{
				// "id" field is missing
				"name": "Bob",
				"tags": []jsonmatch.Object{{"name": "support"}, {"name": "postmaster"}}, // unexpected list entry
			},
		},
	}

	AssertEqual(t, match.DiffAgainst(message), []jsonmatch.Diff{
		{
			Kind:         "value mismatch",
			Pointer:      "/users/0/name",
			ActualJSON:   `"Alice"`,
			ExpectedJSON: `"Alicia"`,
		},
		{
			Kind:         "value mismatch",
			Pointer:      "/users/0/tags/0/name",
			ActualJSON:   `"admin"`,
			ExpectedJSON: `"administrator"`,
		},
		{
			Kind:         "value mismatch",
			Pointer:      "/users/0/tags/1",
			ActualJSON:   `{"name":"senior"}`,
			ExpectedJSON: `<missing>`,
		},
		{
			Kind:         "value mismatch",
			Pointer:      "/users/0/status",
			ActualJSON:   `<missing>`,
			ExpectedJSON: `"fixing stuff"`,
		},
		{
			Kind:         "value mismatch",
			Pointer:      "/users/1/id",
			ActualJSON:   `42`,
			ExpectedJSON: `<missing>`,
		},
		{
			Kind:         "value mismatch",
			Pointer:      "/users/1/tags/1",
			ActualJSON:   `<missing>`,
			ExpectedJSON: `{"name":"postmaster"}`,
		},
	})
}

func TestFailsOnTypeMismatch(t *testing.T) {
	// several JSON values with incompatible JSON-level types, paired with their code-level representation
	testCases := []struct {
		JSON   string
		Data   any
		Scalar Option[jsonmatch.Diffable] // for testing calls to jsonmatch.Scalar().DiffAgainst() (see below)
	}{
		{
			JSON:   `null`,
			Data:   nil,
			Scalar: Some(jsonmatch.Null()),
		},
		{
			JSON:   `true`,
			Data:   true,
			Scalar: Some(jsonmatch.Scalar(true)),
		},
		{
			JSON:   `42`,
			Data:   42,
			Scalar: Some(jsonmatch.Scalar(42)),
		},
		{
			JSON:   `"foo"`,
			Data:   "foo",
			Scalar: Some(jsonmatch.Scalar("foo")),
		},
		{
			JSON:   `{"value":42}`,
			Data:   map[string]any{"value": 42},
			Scalar: None[jsonmatch.Diffable](),
		},
		{
			JSON:   `[42]`,
			Data:   []any{42},
			Scalar: None[jsonmatch.Diffable](),
		}}

	for idx1, tc1 := range testCases {
		objectMessage := fmt.Appendf(nil, `{"payload":%s}`, tc1.JSON)
		arrayMessage := fmt.Appendf(nil, `[1,%s]`, tc1.JSON)
		plainMessage := []byte(tc1.JSON)

		for idx2, tc2 := range testCases {
			// type mismatch inside of an object
			objectMatch := jsonmatch.Object{"payload": tc2.Data}
			if idx1 == idx2 {
				// if we chose matching JSON and data types, then everything works as intended
				AssertEqual(t, objectMatch.DiffAgainst(objectMessage), nil)
			} else {
				// otherwise we expect a "type mismatch" error
				AssertEqual(t, objectMatch.DiffAgainst(objectMessage), []jsonmatch.Diff{{
					Kind:         "type mismatch",
					Pointer:      "/payload",
					ActualJSON:   tc1.JSON,
					ExpectedJSON: tc2.JSON,
				}})
			}

			// type mismatch inside of an array
			arrayMatch := jsonmatch.Array{1, tc2.Data}
			if idx1 == idx2 {
				AssertEqual(t, arrayMatch.DiffAgainst(arrayMessage), nil)
			} else {
				AssertEqual(t, arrayMatch.DiffAgainst(arrayMessage), []jsonmatch.Diff{{
					Kind:         "type mismatch",
					Pointer:      "/1",
					ActualJSON:   tc1.JSON,
					ExpectedJSON: tc2.JSON,
				}})
			}

			// type mismatch for plain scalar
			if scalarMatch, ok := tc2.Scalar.Unpack(); ok {
				if idx1 == idx2 {
					AssertEqual(t, scalarMatch.DiffAgainst(plainMessage), nil)
				} else {
					AssertEqual(t, scalarMatch.DiffAgainst(plainMessage), []jsonmatch.Diff{{
						Kind:         "type mismatch",
						Pointer:      "",
						ActualJSON:   tc1.JSON,
						ExpectedJSON: tc2.JSON,
					}})
				}
			}
		}
	}
}

func TestFailsOnUnmarshalError(t *testing.T) {
	// all of these things are definitely not valid JSON messages
	testCases := [][]byte{
		// empty string
		[]byte(""),
		// looks like text/plain
		[]byte("Not found\n"),
		// looks like text/yaml
		[]byte("data:\n- 23\n- 42\n"),
		// incomplete JSON
		[]byte(`{"data":[23,`),
		// this one is not even a valid UTF-8 string
		[]byte("a\xffb\xC0\xAFc\xff"),
	}
	match := jsonmatch.Object{
		"data": jsonmatch.Array{23, 42},
	}

	for _, message := range testCases {
		diffs := match.DiffAgainst(message)
		if AssertEqual(t, len(diffs), 1) {
			diff := diffs[0]
			AssertEqual(t, strings.HasPrefix(diff.Kind, "unmarshal error ("), true)
			AssertEqual(t, strings.HasSuffix(diff.Kind, ")"), true)
			AssertEqual(t, diff.Pointer, "")
			AssertEqual(t, diff.ExpectedJSON, `{"data":[23,42]}`)
			AssertEqual(t, strings.ReplaceAll(diff.ActualJSON, "\uFFFD", ""), strings.ToValidUTF8(string(message), ""))
		}
	}
}

type unmarshalableObject struct{}

func (unmarshalableObject) MarshalJSON() ([]byte, error) {
	return nil, errors.New("this object is unmarshalable")
}
