// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package oblast_test

import (
	"database/sql"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
	"go.xyrillian.de/gg/oblast"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/mock"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/must"
)

func TestRuntimeIndex(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	commonSetup := func() {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id > 0`).
			ExpectQueryWithArgs().
			AndReturnColumns("id", "name").
			WithRow(1, "foo").
			WithRow(2, "bar").
			WithRow(3, "baz")
	}

	t.Run("Index", func(t *testing.T) {
		byName := oblast.NewRuntimeIndex(func(r basicRecord) string { return r.Name })

		// test success path
		commonSetup()
		result := must.Return(byName.IndexFrom(store.SelectWhere(ctx, db, `id > 0`)))(t)
		assert.Equal(t, result, map[string]basicRecord{
			"foo": {1, "foo"},
			"bar": {2, "bar"},
			"baz": {3, "baz"},
		})

		// test error path
		_, err := byName.IndexFrom(store.SelectWhere(ctx, db, `id = 1`))
		assert.ErrEqual(t, err, `during Query(): unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id = 1`)
	})

	t.Run("Partition", func(t *testing.T) {
		byFirstLetter := oblast.NewRuntimeIndex(func(r basicRecord) string { return r.Name[0:1] })

		// test success path
		commonSetup()
		result := must.Return(byFirstLetter.PartitionFrom(store.SelectWhere(ctx, db, `id > 0`)))(t)
		assert.Equal(t, result, map[string][]basicRecord{
			"f": {{1, "foo"}},
			"b": {{2, "bar"}, {3, "baz"}},
		})

		// test error path
		_, err := byFirstLetter.PartitionFrom(store.SelectWhere(ctx, db, `id = 1`))
		assert.ErrEqual(t, err, `during Query(): unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id = 1`)
	})
}
