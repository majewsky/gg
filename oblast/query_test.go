// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package oblast_test

import (
	"database/sql"
	"strconv"
	"testing"
	"time"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/gsql"
	"go.xyrillian.de/gg/oblast"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/mock"
	"go.xyrillian.de/gg/oblast/internal/testhelpers/must"
)

func TestInsertBasic(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}

	// testing with the SQLite dialect exercises the Exec()-based codepath
	t.Run("driver=sqlite", func(t *testing.T) {
		store := oblast.MustNewStore[basicRecord](
			oblast.SqliteDialect(),
			oblast.TableNameIs("basic_records"),
			oblast.PrimaryKeyIs("id"),
		)

		for _, batchSize := range []int{1, oblast.PrepareThreshold - 1, oblast.PrepareThreshold + 1} {
			t.Run("N="+strconv.Itoa(batchSize), func(t *testing.T) {
				records := make([]*basicRecord, batchSize)
				for idx := range batchSize {
					records[idx] = &basicRecord{Name: "new"}
					md.ForQuery(`INSERT INTO "basic_records" ("name") VALUES (?)`).
						ExpectExecWithArgs("new").
						AndReturnLastInsertId(int64(42 + idx))
				}
				must.Succeed(t, store.Insert(ctx, db, records...))
				for idx, r := range records {
					assert.Equal(t, r.ID, int64(42+idx))
				}
			})
		}
	})

	// testing with the Postgres dialect exercises the QueryRow()-based codepath
	t.Run("driver=postgres", func(t *testing.T) {
		store := oblast.MustNewStore[basicRecord](
			oblast.PostgresDialect(),
			oblast.TableNameIs("basic_records"),
			oblast.PrimaryKeyIs("id"),
		)

		for _, batchSize := range []int{1, oblast.PrepareThreshold - 1, oblast.PrepareThreshold + 1} {
			t.Run("N="+strconv.Itoa(batchSize), func(t *testing.T) {
				records := make([]*basicRecord, batchSize)
				for idx := range batchSize {
					records[idx] = &basicRecord{Name: "new"}
					md.ForQuery(`INSERT INTO "basic_records" ("name") VALUES ($1) RETURNING "id"`).
						ExpectQueryWithArgs("new").
						AndReturnColumns("id").
						WithRow(int64(42 + idx))
				}
				must.Succeed(t, store.Insert(ctx, db, records...))
				for idx, r := range records {
					assert.Equal(t, r.ID, int64(42+idx))
				}
			})
		}
	})
}

func TestInsertWithUintPrimaryKey(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type exoticRecord struct {
		ID   uint64 `oblast:"id,auto"`
		Name string `oblast:"name"`
	}
	store := oblast.MustNewStore[exoticRecord](
		oblast.SqliteDialect(),
		oblast.StructTagKeyIs("oblast"), // this test also randomly provides coverage for this option
		oblast.TableNameIs("exotic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	// success case: positive ID fits into uint64
	md.ForQuery(`INSERT INTO "exotic_records" ("name") VALUES (?)`).
		ExpectExecWithArgs("new").
		AndReturnLastInsertId(42)
	record := exoticRecord{Name: "new"}
	must.Succeed(t, store.Insert(ctx, db, &record))
	assert.Equal(t, record.ID, 42)

	// error case: negative ID cannot be converted to uint64
	md.ForQuery(`INSERT INTO "exotic_records" ("name") VALUES (?)`).
		ExpectExecWithArgs("another").
		AndReturnLastInsertId(-42)
	record = exoticRecord{Name: "another"}
	err := store.Insert(ctx, db, &record)
	assert.ErrEqual(t, err, "LastInsertId() = -42 for record with idx = 0 cannot be converted to uint")
}

func TestUpdateBasic(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	for _, batchSize := range []int{1, oblast.PrepareThreshold - 1, oblast.PrepareThreshold + 1} {
		t.Run("N="+strconv.Itoa(batchSize), func(t *testing.T) {
			records := make([]basicRecord, batchSize)
			for idx := range batchSize {
				r := basicRecord{ID: int64(42 + idx), Name: "updated"}
				records[idx] = r
				md.ForQuery(`UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`).
					ExpectExecWithArgs(r.Name, r.ID).
					AndReturnRowsAffected(1)
			}
			must.Succeed(t, store.Update(ctx, db, records...))
		})
	}
}

func TestDeleteBasic(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	for _, batchSize := range []int{1, oblast.PrepareThreshold - 1, oblast.PrepareThreshold + 1} {
		t.Run("N="+strconv.Itoa(batchSize), func(t *testing.T) {
			records := make([]basicRecord, batchSize)
			for idx := range batchSize {
				r := basicRecord{ID: int64(42 + idx), Name: "removed"}
				records[idx] = r
				md.ForQuery(`DELETE FROM "basic_records" WHERE "id" = ?`).
					ExpectExecWithArgs(r.ID).
					AndReturnRowsAffected(1)
			}
			must.Succeed(t, store.Delete(ctx, db, records...))
		})
	}
}

func TestUpsertBasicWithAutoColumn(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	md.ForQuery(`INSERT INTO "basic_records" ("name") VALUES (?)`).
		ExpectExecWithArgs("first needs insert").
		AndReturnLastInsertId(1)
	md.ForQuery(`UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`).
		ExpectExecWithArgs("second needs update", 2).
		AndReturnRowsAffected(1)
	md.ForQuery(`INSERT INTO "basic_records" ("name") VALUES (?)`).
		ExpectExecWithArgs("third needs insert").
		AndReturnLastInsertId(3)
	md.ForQuery(`UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`).
		ExpectExecWithArgs("fourth needs update", 4).
		AndReturnRowsAffected(1)

	records := []*basicRecord{
		{Name: "first needs insert"},
		{ID: 2, Name: "second needs update"},
		{Name: "third needs insert"},
		{ID: 4, Name: "fourth needs update"},
	}
	must.Succeed(t, store.Upsert(ctx, db, records...))

	assert.Equal(t, records, []*basicRecord{
		{ID: 1, Name: "first needs insert"},
		{ID: 2, Name: "second needs update"},
		{ID: 3, Name: "third needs insert"},
		{ID: 4, Name: "fourth needs update"},
	})
}

func TestWriteQueriesNotPossible(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		// no TableNameIs() or PrimaryKeyIs() given
	)

	r := basicRecord{Name: "foo"}
	err := store.Insert(ctx, db, &r)
	assert.ErrEqual(t, err, "cannot execute Insert() because query could not be autogenerated")

	err = store.Upsert(ctx, db, &r)
	assert.ErrEqual(t, err, "cannot execute Insert() because query could not be autogenerated")

	r.ID = 42
	err = store.Update(ctx, db, r)
	assert.ErrEqual(t, err, "cannot execute Update() because query could not be autogenerated")

	err = store.Delete(ctx, db, r)
	assert.ErrEqual(t, err, "cannot execute Delete() because query could not be autogenerated")

	store = oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("records"),
		oblast.PrimaryKeyIs("id"),
		oblast.ReadOnly(),
	)

	r = basicRecord{Name: "foo"}
	err = store.Insert(ctx, db, &r)
	assert.ErrEqual(t, err, "cannot execute Insert() because query planning used the ReadOnly() option")

	err = store.Upsert(ctx, db, &r)
	assert.ErrEqual(t, err, "cannot execute Insert() because query planning used the ReadOnly() option")

	r.ID = 42
	err = store.Update(ctx, db, r)
	assert.ErrEqual(t, err, "cannot execute Update() because query planning used the ReadOnly() option")

	err = store.Delete(ctx, db, r)
	assert.ErrEqual(t, err, "cannot execute Delete() because query planning used the ReadOnly() option")
}

func TestWriteQueriesFailDuringPrepare(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	for _, batchSize := range []int{1, oblast.PrepareThreshold - 1, oblast.PrepareThreshold + 1} {
		records := make([]basicRecord, batchSize)
		recordsForInsert := make([]*basicRecord, batchSize)
		for idx := range batchSize {
			records[idx] = basicRecord{ID: int64(42 + idx), Name: "foo"}
			recordsForInsert[idx] = &basicRecord{Name: "foo"}
		}

		err := store.Insert(ctx, db, recordsForInsert...)
		baseError := `unexpected query: INSERT INTO "basic_records" ("name") VALUES (?)`
		if batchSize < oblast.PrepareThreshold {
			assert.ErrEqual(t, err, "while inserting record with idx = 0: "+baseError)
		} else {
			assert.ErrEqual(t, err, "during Prepare(): "+baseError)
		}

		err = store.Update(ctx, db, records...)
		baseError = `unexpected query: UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`
		if batchSize < oblast.PrepareThreshold {
			assert.ErrEqual(t, err, "while updating record with idx = 0: "+baseError)
		} else {
			assert.ErrEqual(t, err, "during Prepare(): "+baseError)
		}

		err = store.Delete(ctx, db, records...)
		baseError = `unexpected query: DELETE FROM "basic_records" WHERE "id" = ?`
		if batchSize < oblast.PrepareThreshold {
			assert.ErrEqual(t, err, "while deleting record with idx = 0: "+baseError)
		} else {
			assert.ErrEqual(t, err, "during Prepare(): "+baseError)
		}
	}
}

func TestUpdateOrUpsertFailsOnMissingRecord(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	// test Update()
	md.ForQuery(`UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`).
		ExpectExecWithArgs("changed", 42).
		AndReturnRowsAffected(0)
	err := store.Update(ctx, db, basicRecord{ID: 42, Name: "changed"})
	assert.ErrEqual(t, err, "could not UPDATE record that does not exist in the database: id = 42")
	_, hasCorrectType := err.(oblast.MissingRecordError[basicRecord]) //nolint:errorlint // we explicitly do not want a wrapped error
	assert.Equal(t, hasCorrectType, true)

	// test Upsert() -> this will not try inserting because the strategy
	// is chosen based on the fill state of the "auto" field
	md.ForQuery(`UPDATE "basic_records" SET "name" = ? WHERE "id" = ?`).
		ExpectExecWithArgs("changed", 42).
		AndReturnRowsAffected(0)
	err = store.Upsert(ctx, db, &basicRecord{ID: 42, Name: "changed"})
	assert.ErrEqual(t, err, "could not UPDATE record that does not exist in the database: id = 42")
	_, hasCorrectType = err.(oblast.MissingRecordError[basicRecord]) //nolint:errorlint // we explicitly do not want a wrapped error
	assert.Equal(t, hasCorrectType, true)
}

func TestInsertFailsOnFilledAutoField(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type basicRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
	}
	store := oblast.MustNewStore[basicRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("basic_records"),
		oblast.PrimaryKeyIs("id"),
	)

	err := store.Insert(ctx, db, &basicRecord{ID: 23, Name: "third"})
	assert.ErrEqual(t, err, `refusing to INSERT record with idx = 0 that already has non-zero values in its "auto" columns`)
}

func TestInsertAndUpsertWithNoAutoColumns(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type relation struct {
		FooID int64 `db:"foo_id"`
		BarID int64 `db:"bar_id"`
	}
	store := oblast.MustNewStore[relation](
		oblast.SqliteDialect(),
		oblast.TableNameIs("foo_bar_relations"),
		oblast.PrimaryKeyIs("foo_id", "bar_id"),
	)

	// test Insert()
	md.ForQuery(`INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?)`).
		ExpectExecWithArgs(23, 42).
		AndReturnRowsAffected(1)
	must.Succeed(t, store.Insert(ctx, db, &relation{23, 42}))

	// test Upsert()
	md.ForQuery(`INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?) ON CONFLICT ("foo_id", "bar_id") DO NOTHING`).
		ExpectExecWithArgs(1, 2).
		AndReturnRowsAffected(1)
	md.ForQuery(`INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?) ON CONFLICT ("foo_id", "bar_id") DO NOTHING`).
		ExpectExecWithArgs(3, 4).
		AndReturnRowsAffected(1)
	must.Succeed(t, store.Upsert(ctx, db, &relation{1, 2}, &relation{3, 4}))
}

func TestUpsertFailsOnMixedAutoFieldState(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	type complexRecord struct {
		ID        int64     `db:"id,auto"`
		Name      string    `db:"name"`
		CreatedAt time.Time `db:"created_at,auto"`
	}
	store := oblast.MustNewStore[complexRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("complex_records"),
		oblast.PrimaryKeyIs("id"),
	)

	brokenRecord := complexRecord{
		ID:        42, // this looks like we need to UPDATE
		Name:      "foo",
		CreatedAt: time.Time{}, // this looks like we need to INSERT
	}
	err := store.Upsert(ctx, db, &brokenRecord)
	assert.ErrEqual(t, err, `cannot decide whether to INSERT or UPDATE record with idx = 0: some "auto" columns are zero, others are not`)
}

func TestUninitializedTransparentPointerStructs(t *testing.T) {
	ctx := t.Context()
	md := mock.NewDriver()
	db := gsql.NewDB(sql.OpenDB(md))

	// declare a record type that has a transparent pointer struct containing non-primary-key fields
	type timestamps struct {
		CreatedAt time.Time  `db:"created_at"`
		DeletedAt *time.Time `db:"deleted_at"`
	}
	type nestedRecord struct {
		ID   int64  `db:"id,auto"`
		Name string `db:"name"`
		*timestamps
	}
	nestedRecordStore := oblast.MustNewStore[nestedRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("nested_records"),
		oblast.PrimaryKeyIs("id"),
	)

	// declare another record type that has a primary key field within a transparent pointer struct
	type commonFields struct {
		ID        int64      `db:"id,auto"`
		CreatedAt time.Time  `db:"created_at"`
		DeletedAt *time.Time `db:"deleted_at"`
	}
	type weirdRecord struct {
		*commonFields
		Name string `db:"name"`
	}
	weirdRecordStore := oblast.MustNewStore[weirdRecord](
		oblast.SqliteDialect(),
		oblast.TableNameIs("weird_records"),
		oblast.PrimaryKeyIs("id"),
	)

	// check detection on INSERT
	freshBrokenRecord := nestedRecord{
		Name:       "foo",
		timestamps: nil, // problem: cannot access `freshBrokenRecord.CreatedAt` or `freshBrokenRecord.DeletedAt`
	}
	err := nestedRecordStore.Insert(ctx, db, &freshBrokenRecord)
	assert.ErrEqual(t, err, `refusing to INSERT record with idx = 0: cannot access all mapped fields because field "timestamps" holds a nil pointer`)
	err = nestedRecordStore.Upsert(ctx, db, &freshBrokenRecord)
	assert.ErrEqual(t, err, `refusing to INSERT or UPDATE record with idx = 0: cannot access all mapped fields because field "timestamps" holds a nil pointer`)

	// check success case on INSERT
	now := time.Now()
	freshIntactRecord := nestedRecord{
		Name:       "foo",
		timestamps: &timestamps{CreatedAt: now, DeletedAt: nil},
	}
	md.ForQuery(`INSERT INTO "nested_records" ("name", "created_at", "deleted_at") VALUES (?, ?, ?)`).
		ExpectExecWithArgs("foo", now, (*time.Time)(nil)).
		AndReturnLastInsertId(1)
	must.Succeed(t, nestedRecordStore.Insert(ctx, db, &freshIntactRecord))
	assert.Equal(t, freshIntactRecord.ID, 1)

	// check detection on UPDATE
	existingBrokenRecord := nestedRecord{
		ID:         42,
		Name:       "bar",
		timestamps: nil, // same problem as above
	}
	err = nestedRecordStore.Update(ctx, db, existingBrokenRecord)
	assert.ErrEqual(t, err, `refusing to UPDATE record with idx = 0: cannot access all mapped fields because field "timestamps" holds a nil pointer`)
	err = nestedRecordStore.Upsert(ctx, db, &freshBrokenRecord)
	assert.ErrEqual(t, err, `refusing to INSERT or UPDATE record with idx = 0: cannot access all mapped fields because field "timestamps" holds a nil pointer`)

	// check success case on UPDATE
	now = time.Now()
	existingIntactRecord := nestedRecord{
		ID:         42,
		Name:       "bar",
		timestamps: &timestamps{CreatedAt: now, DeletedAt: nil},
	}
	md.ForQuery(`UPDATE "nested_records" SET "name" = ?, "created_at" = ?, "deleted_at" = ? WHERE "id" = ?`).
		ExpectExecWithArgs("bar", now, (*time.Time)(nil), 42).
		AndReturnRowsAffected(1)
	must.Succeed(t, nestedRecordStore.Update(ctx, db, existingIntactRecord))

	// check that detection on DELETE does not care about transparent pointer structs as long as they do not contain PK fields
	md.ForQuery(`DELETE FROM "nested_records" WHERE "id" = ?`).
		ExpectExecWithArgs(42).
		AndReturnRowsAffected(1)
	must.Succeed(t, nestedRecordStore.Delete(ctx, db, existingBrokenRecord))

	// check detection on DELETE where it matters
	existingWeirdRecord := weirdRecord{
		commonFields: nil, // problem: cannot access `existingWeirdRecord.ID`
		Name:         "qux",
	}
	err = weirdRecordStore.Delete(ctx, db, existingWeirdRecord)
	assert.ErrEqual(t, err, `refusing to DELETE record with idx = 0: cannot access all primary key fields because field "commonFields" holds a nil pointer`)
}
