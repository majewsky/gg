// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package gsql_test

import (
	"database/sql"
	"errors"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
	. "go.xyrillian.de/gg/option"
)

func TestNoneIfNoRows(t *testing.T) {
	x, err := gsql.NoneIfNoRows(42, nil)
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, x, Some(42))
	}

	x, err = gsql.NoneIfNoRows(0, sql.ErrNoRows)
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, x, None[int]())
	}

	_, err = gsql.NoneIfNoRows(0, errors.New("kaboom"))
	assert.ErrEqual(t, err, "kaboom")
}
