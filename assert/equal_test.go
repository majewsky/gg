// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert_test

import (
	"encoding/json"
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/testcapture"
)

func expectErrors(t *testing.T, test func(assert.TestingTB), expected string) {
	t.Helper()
	r := testcapture.Result{Outcome: testcapture.OutcomeFailed}
	for line := range strings.SplitSeq(expected, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			r.Messages = append(r.Messages, testcapture.Log(line))
		}
	}
	assert.Equal(t, testcapture.Capture(t.Context(), t.Name(), test), r)
}

func TestEqual(t *testing.T) {
	// basic test for scalars
	assert.Equal(t, true, true)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, false, true)
	}, `expected true, but got false`)

	// basic test for slices
	correctSlice := []int{1, 2, 3}
	wrongSlice := []int{1, 42, 3}
	assert.Equal(t, correctSlice, correctSlice)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, wrongSlice, correctSlice)
	}, `at actual[1]: expected 2, but got 42`)

	// basic test for slices of different lengths
	overlongSlice := []int{1, 2, 3, 4}
	truncatedSlice := []int{1, 2}
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, overlongSlice, correctSlice)
	}, `at actual[3]: expected <missing>, but got 4`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, truncatedSlice, correctSlice)
	}, `at actual[2]: expected 3, but got <missing>`)

	// slices: report the entire slice if it comes out more compact than reporting each different element
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = []int{1, 2, 3, 4, 5, 6, 7, 8}
			actual   = []int{2, 3, 4, 5, 6, 7, 8, 9}
		)
		assert.Equal(t, actual, expected)
	}, `expected []int{1, 2, 3, 4, 5, 6, 7, 8}, but got []int{2, 3, 4, 5, 6, 7, 8, 9}`)
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = make([]map[string]int, 6)
			actual   = make([]map[string]int, 6)
		)
		for idx := range expected {
			expected[idx] = map[string]int{"foo": 1 + idx, "bar": 2 + idx}
			actual[idx] = map[string]int{"foo": 1 + idx, "bar": 3 + idx}
		}
		assert.Equal(t, actual, expected)
	}, `
		at actual[0]["bar"]: expected 2, but got 3
		at actual[1]["bar"]: expected 3, but got 4
		at actual[2]["bar"]: expected 4, but got 5
		at actual[3]["bar"]: expected 5, but got 6
		at actual[4]["bar"]: expected 6, but got 7
		at actual[5]["bar"]: expected 7, but got 8
	`)

	// slices: omit the longest common prefix/suffix when reporting the entire slice as different
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = make([]int, 100)
			actual   = make([]int, 100)
		)
		for idx := range expected {
			expected[idx] = idx
			if idx >= 30 && idx <= 40 {
				actual[idx] = 70 - idx // This flips the run [30:40] around.
			} else {
				actual[idx] = idx
			}
		}
		assert.Equal(t, actual, expected)
	}, `at actual[30:41]: expected []int{30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}, but got []int{40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 30}`)

	// slices: as a corner case, omission of the longest common prefix/suffix may truncate one side to an empty slice
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = make([]int, 100)
			actual   = make([]int, 102)
		)
		for idx := range expected {
			expected[idx] = idx
		}
		for idx := range actual {
			if idx < 34 {
				actual[idx] = idx
			} else {
				actual[idx] = idx - 2
			}
			// The end result of this is [0, 1, ...,, 32, 33, 32, 33, 34, ..., 99].
		}
		assert.Equal(t, actual, expected)
	}, `at actual[34:34]: expected []int{}, but got []int{32, 33}`)

	// slices: special handling for []byte that renders them like strings if they are all ASCII
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, []byte("hallo"), []byte("hello"))
	}, `expected []byte("hello"), but got []byte("hallo")`)
	correctPayload, err := json.Marshal(map[string]int{"foo": 1, "bar": 2})
	if err != nil {
		t.Fatal(err)
	}
	wrongPayload, err := json.Marshal(map[string]int{"foo": 1, "bar": 3})
	if err != nil {
		t.Fatal(err)
	}
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, json.RawMessage(wrongPayload), json.RawMessage(correctPayload))
		// ^ Without the special handling, this would produce the nonsensical output:
		//   at actual[7]: expected 0x32, but got 0x33
	}, "expected []byte(`{\"bar\":2,\"foo\":1}`), but got []byte(`{\"bar\":3,\"foo\":1}`)")
	// ^ This checks that backticks are used when it makes a nicer output.

	// basic test for maps
	correctMap := map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
	}
	wrongMap := map[string]int{
		"one":  1,
		"two":  23,
		"four": 4,
	}
	assert.Equal(t, correctMap, correctMap)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, wrongMap, correctMap)
	}, `
		at actual["four"]: expected <missing>, but got 4
		at actual["three"]: expected 3, but got <missing>
		at actual["two"]: expected 2, but got 23
	`)

	// basic test for structs
	type record struct {
		ID      int
		Name    string
		private bool
	}
	correctStruct := record{1, "Alice", false}
	wrongStruct1 := record{2, "Bob", false}
	assert.Equal(t, correctStruct, correctStruct)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, wrongStruct1, correctStruct)
	}, `
		at actual.ID: expected 1, but got 2
		at actual.Name: expected "Alice", but got "Bob"
	`)

	// test for structs: diff in private field (we cannot Interface() the values in such fields, so we have to report the entire struct)
	wrongStruct2 := record{1, "Alice", true}
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, wrongStruct2, correctStruct)
	}, `expected assert_test.record{ID:1, Name:"Alice", private:false}, but got assert_test.record{ID:1, Name:"Alice", private:true}`)

	// basic test for pointers
	assert.Equal(t, (*bool)(nil), (*bool)(nil))
	assert.Equal(t, new(true), new(true))
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, new(false), new(true))
	}, `at (*actual): expected true, but got false`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, nil, new(true))
	}, `expected pointer to true, but got nil`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, new(false), nil)
	}, `expected nil, but got pointer to false`)

	// basic test for interfaces
	type slot struct {
		Value any
	}
	correctSlot := slot{[]int{1, 2, 3}}
	wrongSlot := slot{[]int{1, 42, 3}}
	mistypedSlot := slot{42}
	assert.Equal(t, correctSlot, correctSlot)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, wrongSlot, correctSlot)
	}, `at actual.Value.([]int)[1]: expected 2, but got 42`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, mistypedSlot, correctSlot)
	}, `at actual.Value: expected []int{1, 2, 3}, but got 42`)

	// check that derefences are elided from reported paths, but only if possible
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = map[string]*[]int{"foo": {1, 2, 3}}
			actual   = map[string]*[]int{"foo": {1, 42, 3}}
		)
		assert.Equal(t, actual, expected)
	}, `at (*actual["foo"])[1]: expected 2, but got 42`)
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = []*map[string]int{{"foo": 1, "bar": 2}}
			actual   = []*map[string]int{{"foo": 1}}
		)
		assert.Equal(t, actual, expected)
	}, `at (*actual[0])["bar"]: expected 2, but got <missing>`)
	expectErrors(t, func(t assert.TestingTB) {
		var (
			expected = []*record{{1, "Alice", false}}
			actual   = []*record{{2, "Bob", false}}
		)
		assert.Equal(t, actual, expected)
	}, `
		at actual[0].ID: expected 1, but got 2
		at actual[0].Name: expected "Alice", but got "Bob"
	`)
}
