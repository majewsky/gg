// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert_test

import (
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

// TODO: check coverage
// TODO: implement special case for ~[]byte containing valid UTF-8
