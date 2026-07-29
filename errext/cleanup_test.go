// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package errext_test

import (
	"errors"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/errext"
)

type fooError struct{}
type barError struct{}
type bazError struct{}

func (fooError) Error() string { return "foo" }
func (barError) Error() string { return "bar" }
func (bazError) Error() string { return "baz" }

func TestIOError(t *testing.T) {
	err := errext.WithCleanup(nil, "File.Close", nil)
	assert.Equal(t, err == nil, true)

	err = errext.WithCleanup(fooError{}, "File.Close", nil)
	assert.ErrEqual(t, err, "foo")
	assert.Equal(t, err, error(fooError{})) // check for no wrapping in type ioError without cleanup error

	err = errext.WithCleanup(nil, "File.Close", barError{})
	assert.ErrEqual(t, err, "during File.Close(): bar")
	assert.Equal(t, errors.Is(err, fooError{}), false)
	assert.Equal(t, errors.Is(err, barError{}), true)
	assert.Equal(t, errors.Is(err, bazError{}), false)

	err = errext.WithCleanup(fooError{}, "File.Close", barError{})
	assert.ErrEqual(t, err, "foo (additional error during File.Close(): bar)")
	assert.Equal(t, errors.Is(err, fooError{}), true)
	assert.Equal(t, errors.Is(err, barError{}), true)
	assert.Equal(t, errors.Is(err, bazError{}), false)
}
