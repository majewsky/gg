// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package oblast_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
	"go.xyrillian.de/gg/oblast"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/mock"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/must"
)

func TestSelectReturningSomeRecords(t *testing.T) {
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

	t.Run("using Store.Select", func(t *testing.T) {
		md.ForQuery(`SELECT * FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name", "id").
			WithRow("foo", 1).
			WithRow("bar", 2)
		records := must.Return(store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, records, []basicRecord{
			{1, "foo"},
			{2, "bar"},
		})
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "ffoo").
			WithRow(2, "bbar")
		records := must.Return(store.SelectWhere(ctx, db, `id < ?`, 3).Collect())(t)
		assert.Equal(t, records, []basicRecord{
			{1, "ffoo"},
			{2, "bbar"},
		})
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "fffoo").
			WithRow(2, "bbbar")
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		records := must.Return(query.Select(ctx, db, 3).Collect())(t)
		assert.Equal(t, records, []basicRecord{
			{1, "fffoo"},
			{2, "bbbar"},
		})
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		md.ForQuery(`SELECT * FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name", "id").
			WithRow("ffffoo", 1).
			WithRow("bbbbar", 2)
		record := must.Return(store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3))(t)
		assert.Equal(t, record, basicRecord{1, "ffffoo"})
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "fffffoo").
			WithRow(2, "bbbbbar")
		record := must.Return(store.SelectOneWhere(ctx, db, `id < ?`, 3))(t)
		assert.Equal(t, record, basicRecord{1, "fffffoo"})
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "ffffffoo").
			WithRow(2, "bbbbbbar")
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		record := must.Return(query.SelectOne(ctx, db, 3))(t)
		assert.Equal(t, record, basicRecord{1, "ffffffoo"})
	})

	commonSetupForValueSelect := func() {
		md.ForQuery(`SELECT name FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name").
			WithRow("foo").
			WithRow("bar")
	}

	t.Run("using oblast.Select", func(t *testing.T) {
		commonSetupForValueSelect()
		names := must.Return(oblast.Select[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, names, []string{"foo", "bar"})
	})

	t.Run("using oblast.SelectOne", func(t *testing.T) {
		commonSetupForValueSelect()
		name := must.Return(oblast.SelectOne[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3))(t)
		assert.Equal(t, name, "foo")
	})

	type tupleRecord struct {
		ID   int64
		Name string
	}
	commonSetupForTupleSelect := func() {
		md.ForQuery(`SELECT id, name FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "foo").
			WithRow(2, "bar")
	}

	t.Run("using TupleSelect", func(t *testing.T) {
		commonSetupForTupleSelect()
		records := must.Return(oblast.TupleSelect[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, records, []tupleRecord{
			{1, "foo"},
			{2, "bar"},
		})
	})

	t.Run("using TupleSelectOne", func(t *testing.T) {
		commonSetupForTupleSelect()
		record := must.Return(oblast.TupleSelectOne[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3))(t)
		assert.Equal(t, record, tupleRecord{1, "foo"})
	})
}

func TestSelectReturningNoRecords(t *testing.T) {
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

	t.Run("using Store.Select", func(t *testing.T) {
		md.ForQuery(`SELECT * FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name", "id")
		records := must.Return(store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, records, nil)
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name")
		records := must.Return(store.SelectWhere(ctx, db, `id < ?`, 3).Collect())(t)
		assert.Equal(t, records, nil)
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name")
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		records := must.Return(query.Select(ctx, db, 3).Collect())(t)
		assert.Equal(t, records, nil)
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		md.ForQuery(`SELECT * FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name", "id")
		_, err := store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, sql.ErrNoRows.Error())
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name")
		_, err := store.SelectOneWhere(ctx, db, `id < ?`, 3)
		assert.ErrEqual(t, err, sql.ErrNoRows.Error())
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		md.ForQuery(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name")
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.SelectOne(ctx, db, 3)
		assert.ErrEqual(t, err, sql.ErrNoRows.Error())
	})

	commonSetupForValueSelect := func() {
		md.ForQuery(`SELECT name FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name")
	}

	t.Run("using oblast.Select", func(t *testing.T) {
		commonSetupForValueSelect()
		names := must.Return(oblast.Select[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, names, nil)
	})

	t.Run("using oblast.SelectOne", func(t *testing.T) {
		commonSetupForValueSelect()
		_, err := oblast.SelectOne[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, sql.ErrNoRows.Error())
	})

	type tupleRecord struct {
		ID   int64
		Name string
	}
	commonSetupForTupleSelect := func() {
		md.ForQuery(`SELECT id, name FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name")
	}

	t.Run("using TupleSelect", func(t *testing.T) {
		commonSetupForTupleSelect()
		records := must.Return(oblast.TupleSelect[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3).Collect())(t)
		assert.Equal(t, records, nil)
	})

	t.Run("using TupleSelectOne", func(t *testing.T) {
		commonSetupForTupleSelect()
		_, err := oblast.TupleSelectOne[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, sql.ErrNoRows.Error())
	})
}

func TestSelectIntoUnexpectedField(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID          int64  `db:"id"`
		Description string `db:"desc"` // but DB knows only the field "name"!
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	expectedError := "result has column \"name\" in position 0, but no field in type basicRecord has `db:\"name\"`"
	commonSetup := func() {
		md.ForQuery(`SELECT * FROM basic_records WHERE id < ?`).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name", "id").
			WithRow("foo", 1).
			WithRow("bar", 2)
	}

	// NOTE: This problem cannot occur with SelectWhere() and SelectOneWhere() because of their use of query generation.

	t.Run("using Store.Select", func(t *testing.T) {
		commonSetup()
		_, err := store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		commonSetup()
		_, err := store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, expectedError)
	})
}

func TestSelectWithScanError(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID        int64     `db:"id"`
		CreatedAt time.Time `db:"created_at"` // but the DB will give us strings that are not timestamps
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	expectedError := `sql: Scan error on column index 1, name "created_at": unsupported Scan, storing driver.Value type string into type *time.Time`
	commonSetup := func(query string) {
		md.ForQuery(query).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "created_at").
			WithRow(1, "foo").
			WithRow(2, "bar")
	}

	t.Run("using Store.Select", func(t *testing.T) {
		commonSetup(`SELECT * FROM basic_records WHERE id < ?`)
		_, err := store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at" FROM "basic_records" WHERE id < ?`)
		_, err := store.SelectWhere(ctx, db, `id < ?`, 3).Collect()
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at" FROM "basic_records" WHERE id < ?`)
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.Select(ctx, db, 3).Collect()
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT * FROM basic_records WHERE id < ?`)
		_, err := store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at" FROM "basic_records" WHERE id < ?`)
		_, err := store.SelectOneWhere(ctx, db, `id < ?`, 3)
		assert.ErrEqual(t, err, expectedError)
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at" FROM "basic_records" WHERE id < ?`)
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.SelectOne(ctx, db, 3)
		assert.ErrEqual(t, err, expectedError)
	})
}

func TestSelectIntoEmbeddedTypes(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type HasCreatedAt struct {
		CreatedAt time.Time `db:"created_at"`
	}
	type HasUpdatedAt struct {
		UpdatedAt *time.Time `db:"updated_at"`
	}
	type compositeRecord struct {
		ID int64 `db:"id"`
		HasCreatedAt
		// This test specifically wants to see that this field gets initialized
		// whenever one of the Store.Select methods creates a compositeRecord instance.
		*HasUpdatedAt
	}
	store := oblast.MustNewStore[compositeRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("composite_records"),
		oblast.PrimaryKeyIs("id"),
	)

	commonSetup := func(query string) {
		md.ForQuery(query).
			ExpectQueryWithArgs(nil...).
			AndReturnColumns("id", "created_at", "updated_at").
			WithRow(1, time.Unix(1, 0), time.Unix(3, 0)).
			WithRow(2, time.Unix(2, 0), nil)
	}

	t.Run("using Store.Select", func(t *testing.T) {
		commonSetup(`SELECT * FROM composite_records`)
		records := must.Return(store.Select(ctx, db, `SELECT * FROM composite_records`).Collect())(t)
		assert.Equal(t, records, []compositeRecord{
			{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
			{2, HasCreatedAt{time.Unix(2, 0)}, &HasUpdatedAt{nil}},
		})
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at", "updated_at" FROM "composite_records" WHERE TRUE`)
		records := must.Return(store.SelectWhere(ctx, db, `TRUE`).Collect())(t)
		assert.Equal(t, records, []compositeRecord{
			{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
			{2, HasCreatedAt{time.Unix(2, 0)}, &HasUpdatedAt{nil}},
		})
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at", "updated_at" FROM "composite_records" WHERE TRUE`)
		query := store.MustPrepareSelectQueryWhere(`TRUE`)
		records := must.Return(query.Select(ctx, db).Collect())(t)
		assert.Equal(t, records, []compositeRecord{
			{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
			{2, HasCreatedAt{time.Unix(2, 0)}, &HasUpdatedAt{nil}},
		})
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT * FROM composite_records`)
		record := must.Return(store.SelectOne(ctx, db, `SELECT * FROM composite_records`))(t)
		assert.Equal(t, record,
			compositeRecord{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
		)
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at", "updated_at" FROM "composite_records" WHERE TRUE`)
		record := must.Return(store.SelectOneWhere(ctx, db, `TRUE`))(t)
		assert.Equal(t, record,
			compositeRecord{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
		)
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT "id", "created_at", "updated_at" FROM "composite_records" WHERE TRUE`)
		query := store.MustPrepareSelectQueryWhere(`TRUE`)
		record := must.Return(query.SelectOne(ctx, db))(t)
		assert.Equal(t, record,
			compositeRecord{1, HasCreatedAt{time.Unix(1, 0)}, &HasUpdatedAt{new(time.Unix(3, 0))}},
		)
	})
}

func TestSelectCapturingQueryError(t *testing.T) {
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

	t.Run("using Store.Select", func(t *testing.T) {
		_, err := store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "during Query(): unexpected query: SELECT * FROM basic_records WHERE id < ?")
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		_, err := store.SelectWhere(ctx, db, `id < ?`, 3).Collect()
		assert.ErrEqual(t, err, `during Query(): unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.Select(ctx, db, 3).Collect()
		assert.ErrEqual(t, err, `during Query(): unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		_, err := store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "during Query(): unexpected query: SELECT * FROM basic_records WHERE id < ?")
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		_, err := store.SelectOneWhere(ctx, db, `id < ?`, 3)
		assert.ErrEqual(t, err, `unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.SelectOne(ctx, db, 3)
		assert.ErrEqual(t, err, `unexpected query: SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
	})

	t.Run("using oblast.Select", func(t *testing.T) {
		_, err := oblast.Select[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "during Query(): unexpected query: SELECT name FROM basic_records WHERE id < ?")
	})

	t.Run("using oblast.SelectOne", func(t *testing.T) {
		_, err := oblast.SelectOne[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "unexpected query: SELECT name FROM basic_records WHERE id < ?")
	})

	type tupleRecord struct {
		ID   int64
		Name string
	}

	t.Run("using oblast.TupleSelect", func(t *testing.T) {
		_, err := oblast.TupleSelect[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "during Query(): unexpected query: SELECT id, name FROM basic_records WHERE id < ?")
	})

	t.Run("using oblast.TupleSelectOne", func(t *testing.T) {
		_, err := oblast.TupleSelectOne[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "during Query(): unexpected query: SELECT id, name FROM basic_records WHERE id < ?")
	})
}

func TestSelectCapturingCloseError(t *testing.T) {
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

	commonSetup := func(query string) {
		md.ForQuery(query).
			ExpectQueryWithArgs(3).
			AndReturnColumns("id", "name").
			WithRow(1, "foo").
			WithRow(2, "bar").
			AndCloseFailsWith(errors.New("datacenter on fire"))
	}

	t.Run("using Store.Select", func(t *testing.T) {
		commonSetup(`SELECT * FROM basic_records WHERE id < ?`)
		_, err := store.Select(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
		_, err := store.SelectWhere(ctx, db, `id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using PreparedSelectQuery.Select", func(t *testing.T) {
		commonSetup(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.Select(ctx, db, 3).Collect()
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using Store.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT * FROM basic_records WHERE id < ?`)
		_, err := store.SelectOne(ctx, db, `SELECT * FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		commonSetup(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
		_, err := store.SelectOneWhere(ctx, db, `id < ?`, 3)
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using PreparedSelectQuery.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT "id", "name" FROM "basic_records" WHERE id < ?`)
		query := store.MustPrepareSelectQueryWhere(`id < ?`)
		_, err := query.SelectOne(ctx, db, 3)
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	type tupleRecord struct {
		ID   int64
		Name string
	}

	t.Run("using oblast.TupleSelect", func(t *testing.T) {
		commonSetup(`SELECT id, name FROM basic_records WHERE id < ?`)
		_, err := oblast.TupleSelect[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using oblast.TupleSelectOne", func(t *testing.T) {
		commonSetup(`SELECT id, name FROM basic_records WHERE id < ?`)
		_, err := oblast.TupleSelectOne[tupleRecord](ctx, db, `SELECT id, name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	commonSetup = func(query string) {
		md.ForQuery(query).
			ExpectQueryWithArgs(3).
			AndReturnColumns("name").
			WithRow("foo").
			WithRow("bar").
			AndCloseFailsWith(errors.New("datacenter on fire"))
	}

	t.Run("using oblast.Select", func(t *testing.T) {
		commonSetup(`SELECT name FROM basic_records WHERE id < ?`)
		_, err := oblast.Select[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "datacenter on fire")
	})

	t.Run("using oblast.SelectOne", func(t *testing.T) {
		commonSetup(`SELECT name FROM basic_records WHERE id < ?`)
		_, err := oblast.SelectOne[string](ctx, db, `SELECT name FROM basic_records WHERE id < ?`, 3)
		assert.ErrEqual(t, err, "datacenter on fire")
	})
}

func TestSelectNotPossibleWithoutTableName(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](oblast.SqliteDialect())

	t.Run("using Store.SelectWhere", func(t *testing.T) {
		_, err := store.SelectWhere(ctx, db, `id < ?`, 3).Collect()
		assert.ErrEqual(t, err, "cannot execute SelectWhere() because query could not be autogenerated")
	})

	t.Run("using Store.SelectOneWhere", func(t *testing.T) {
		_, err := store.SelectOneWhere(ctx, db, `id < ?`, 3)
		assert.ErrEqual(t, err, "cannot execute SelectOneWhere() because query could not be autogenerated")
	})

	t.Run("using PreparedSelectQuery", func(t *testing.T) {
		_, err := store.PrepareSelectQueryWhere(`id < ?`)
		assert.ErrEqual(t, err, "cannot execute PrepareSelectQueryWhere() because query could not be autogenerated")
	})
}
