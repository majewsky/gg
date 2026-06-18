// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert_test

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/testcapture"
)

func TestErrEqual(t *testing.T) {
	noError := error(nil)
	fooError := errors.New("foo error")
	barError := errors.New("bar error")
	nestedFooError := fmt.Errorf("nested error: %w", fooError)

	// test matching against nil
	assert.ErrEqual(t, noError, nil)
	assert.ErrEqual(t, noError, (*os.PathError)(nil))
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, fooError, nil)
	}, `expected no error, but got "foo error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, fooError, (*os.PathError)(nil))
	}, `expected no error, but got "foo error"`)

	// test matching against error
	assert.ErrEqual(t, fooError, fooError)
	assert.ErrEqual(t, nestedFooError, fooError)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, fooError, nestedFooError) // error nesting does not work the other way, we need to see the full `expected`error in`actual`
	}, `expected "nested error: foo error", but got "foo error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, barError, fooError) // check with unrelated errors for completeness
	}, `expected "foo error", but got "bar error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, nil, fooError)
	}, `expected "foo error", but got no error`)

	// test matching against string
	assert.ErrEqual(t, fooError, "foo error")
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, nestedFooError, "foo error") // partial matches do not work
	}, `expected "foo error", but got "nested error: foo error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, barError, "foo error") // check with unrelated errors for completeness
	}, `expected "foo error", but got "bar error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, nil, "foo error")
	}, `expected "foo error", but got no error`)

	// test matching against regexp
	fooRegexp := regexp.MustCompile(`foo`)
	assert.ErrEqual(t, fooError, fooRegexp) // partial matches allowed here as long as regexp does not use ^ and $
	assert.ErrEqual(t, nestedFooError, fooRegexp)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, barError, fooRegexp) // check with unrelated errors for completeness
	}, `expected an error matching /foo/, but got "bar error"`)
	expectErrors(t, func(t assert.TestingTB) {
		assert.ErrEqual(t, nil, fooRegexp)
	}, `expected an error matching /foo/, but got no error`)

	// test matching against unexpected type
	result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		assert.ErrEqual(t, errors.New("42"), 42)
	})
	assert.Equal(t, result, testcapture.Result{
		Outcome: testcapture.OutcomePanicked,
		Panic:   "cannot handle `expected` of type int",
	})
}
