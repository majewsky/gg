// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"testing"

	"go.xyrillian.de/gg/assert"
)

func TestMultipleConnectionsToSameDB(t *testing.T) {
	ctx := t.Context()

	// connect once, create a fresh table and a record
	db1, target := connector.ConnectForTest(t, defaultBehavior)
	for _, query := range []string{
		`DROP TABLE IF EXISTS objects`,
		`CREATE TABLE objects (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL)`,
		`INSERT INTO objects (name) VALUES ('foo')`, // -> id = 1
		`INSERT INTO objects (name) VALUES ('bar')`, // -> id = 2
	} {
		_, err := execQuery(ctx, db1, query, nil)
		if err != nil {
			t.Fatalf("in query %q: %s", query, err.Error())
		}
	}

	// a second connection made with Connect() should see that data
	db2, err := connector.Connect(ctx, target, defaultBehavior)
	if err != nil {
		t.Fatal(err.Error())
	}
	barID, err := selectOneValue[int64](ctx, db2, `SELECT id FROM objects WHERE name = $1`, "bar")
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, barID, 2)
	}

	// another connection with ConnectForTest() should truncate all tables and reset all sequences
	db3, target2 := connector.ConnectForTest(t, defaultBehavior)
	assert.Equal(t, target, target2)
	var count int64
	err = queryRow(ctx, db3, `SELECT COUNT(*) FROM objects`, nil, []any{&count})
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, count, 0)
	}
	nextID, err := selectOneValue[int64](ctx, db3, `INSERT INTO objects (name) VALUES ($1) RETURNING id`, "qux")
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, nextID, 1) // sequence was reset and starts at 1 again
	}
}

func TestOverlongDatabaseName(t *testing.T) {
	t.Run("that is so very long and ridiculous oh my god how is it still going wtf", func(t *testing.T) {
		_, target := connector.ConnectForTest(t, defaultBehavior)
		const expectedPrefix = "testoverlongdatabasename_that_is_so_very_long_and_rid__"
		assert.Equal(t, target.DatabaseName, expectedPrefix+target.DatabaseName[len(expectedPrefix):63])
	})
}
