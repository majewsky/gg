// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package oblast

// ^ NOTE: This is testing internal types and thus must reside in the same package.

import (
	"reflect"
	"testing"
	"time"

	"go.xyrillian.de/gg/assert"
)

// Clears all fields of `p` that are not used when actually running queries,
// i.e. intermediate fields holding analysis results that went into query planning.
// This is used to shorten the assertions in TestQueryConstruction...() below.
func onlyQueryPlans(p plan) plan {
	p.AllColumnNames = nil
	p.AutoColumnNames = nil
	p.IndexByColumnName = nil
	p.PrimaryKeyColumnNames = nil
	p.TableName = ""
	p.TypeName = ""
	return p
}

// Basically the opposite of onlyQueryPlans.
// TestPlanFieldTraversal() only cares about the analysis part, not the query construction phase.
func onlyAnalysisResult(p plan) plan {
	p.Select = plannedQuery{}
	p.Insert = plannedQuery{}
	p.Upsert = plannedQuery{}
	p.Update = plannedQuery{}
	p.Delete = plannedQuery{}
	return p
}

func TestPlanFieldTraversal(t *testing.T) {
	type Timestamps struct {
		CreatedAt time.Time  `db:"created_at"`
		UpdatedAt *time.Time `db:"updated_at"`
	}
	type yetMoreTimestamps struct {
		DeletedAt *time.Time `db:"deleted_at"`
	}
	type Log struct {
		ID       int64 `db:"id,auto"`
		Message  string
		private1 bool `db:"private1"` //nolint:unused
		Ignored  any  `db:"-"`
		Timestamps
		*yetMoreTimestamps
		MoreText struct {
			Description string
		}
		YetMoreText struct {
			Payload string
		} `db:"-"`
		OpaqueText struct {
			ShortMessage string
			LongMessage  string
		} `db:"OpaqueText"`
	}

	// check that the plan for Log:
	// 1. uses the field name as a column name for "Message"
	// 2. ignores "private1" because it cannot be written through reflection
	// 3. ignores "Ignored" because its column name is "-"
	// 4. traverses into "Timestamps" and includes its fields as well
	// 5. traverses into "yetMoreTimestamps" as well (despite the extra pointer and the type being private)
	// 6. traverses into "MoreText" and includes its fields as well
	// 7. does not traverse into "YetMoreText" and does not include its fields because of `db:"-"`
	// 8. does not traverse into "OpaqueText" because the struct is mapped as a whole
	// 9. recognizes "id" as an autofilled column
	p, err := buildPlan(reflect.TypeFor[Log](), PostgresDialect(), planOpts{
		StructTagKey:          "db",
		TableName:             "log_entries",
		PrimaryKeyColumnNames: []string{"id"},
	})
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, onlyAnalysisResult(p), plan{
		TypeName:              "Log",
		TableName:             "log_entries",
		AllColumnNames:        []string{"id", "Message", "created_at", "updated_at", "deleted_at", "Description", "OpaqueText"},
		PrimaryKeyColumnNames: []string{"id"},
		AutoColumnNames:       []string{"id"},
		IndexByColumnName: map[string][]int{
			"id":          {0},
			"Message":     {1},
			"created_at":  {4, 0},
			"updated_at":  {4, 1},
			"deleted_at":  {5, 0},
			"Description": {6, 0},
			"OpaqueText":  {8},
		},
		InsertUsesQueryRow: true,
		TransparentPointerStructFields: []fieldInfo{{
			Name:  "yetMoreTimestamps",
			Index: []int{5},
		}},
	})
}

func TestQueryConstructionBasic(t *testing.T) {
	type record struct {
		ID          int64 `db:",auto"`
		Description string
		CreatedAt   time.Time `db:"CreatedAt"`
	}
	opts := planOpts{
		StructTagKey:          "db",
		TableName:             "basic_records",
		PrimaryKeyColumnNames: []string{"ID"},
	}

	t.Run("MariaDBDialect", func(t *testing.T) {
		opts.ReadOnly = false
		p, err := buildPlan(reflect.TypeFor[record](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       "SELECT `ID`, `Description`, `CreatedAt` FROM `basic_records` WHERE ",
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           "INSERT INTO `basic_records` (`Description`, `CreatedAt`) VALUES (?, ?)",
				ArgumentIndexes: [][]int{{1}, {2}},
				ScanIndexes:     [][]int{{0}},
			},
			Update: plannedQuery{
				Query:           "UPDATE `basic_records` SET `Description` = ?, `CreatedAt` = ? WHERE `ID` = ?",
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           "DELETE FROM `basic_records` WHERE `ID` = ?",
				ArgumentIndexes: [][]int{{0}},
			},
		})

		opts.ReadOnly = true
		p2, err := buildPlan(reflect.TypeFor[record](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p2), plan{
			ReadOnly: true,
			Select:   p.Select,
		})
	})

	t.Run("PostgresDialect", func(t *testing.T) {
		opts.ReadOnly = false
		p, err := buildPlan(reflect.TypeFor[record](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			InsertUsesQueryRow: true,
			Select: plannedQuery{
				Query:       `SELECT "ID", "Description", "CreatedAt" FROM "basic_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "basic_records" ("Description", "CreatedAt") VALUES ($1, $2) RETURNING "ID"`,
				ArgumentIndexes: [][]int{{1}, {2}},
				ScanIndexes:     [][]int{{0}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "basic_records" SET "Description" = $1, "CreatedAt" = $2 WHERE "ID" = $3`,
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "basic_records" WHERE "ID" = $1`,
				ArgumentIndexes: [][]int{{0}},
			},
		})

		opts.ReadOnly = true
		p2, err := buildPlan(reflect.TypeFor[record](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p2), plan{
			InsertUsesQueryRow: true,
			ReadOnly:           true,
			Select:             p.Select,
		})
	})

	t.Run("SqliteDialect", func(t *testing.T) {
		opts.ReadOnly = false
		p, err := buildPlan(reflect.TypeFor[record](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "ID", "Description", "CreatedAt" FROM "basic_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "basic_records" ("Description", "CreatedAt") VALUES (?, ?)`,
				ArgumentIndexes: [][]int{{1}, {2}},
				ScanIndexes:     [][]int{{0}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "basic_records" SET "Description" = ?, "CreatedAt" = ? WHERE "ID" = ?`,
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "basic_records" WHERE "ID" = ?`,
				ArgumentIndexes: [][]int{{0}},
			},
		})

		opts.ReadOnly = true
		p2, err := buildPlan(reflect.TypeFor[record](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p2), plan{
			ReadOnly: true,
			Select:   p.Select,
		})
	})
}

func TestQueryConstructionWithOnlyPrimaryKey(t *testing.T) {
	type relation struct {
		FooID int64 `db:"foo_id"`
		BarID int64 `db:"bar_id"`
	}
	opts := planOpts{
		StructTagKey:          "db",
		TableName:             "foo_bar_relations",
		PrimaryKeyColumnNames: []string{"foo_id", "bar_id"},
	}

	t.Run("MariaDBDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       "SELECT `foo_id`, `bar_id` FROM `foo_bar_relations` WHERE ",
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           "INSERT INTO `foo_bar_relations` (`foo_id`, `bar_id`) VALUES (?, ?)",
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Upsert: plannedQuery{
				Query:           "INSERT INTO `foo_bar_relations` (`foo_id`, `bar_id`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `foo_id` = VALUES(`foo_id`)",
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Delete: plannedQuery{
				Query:           "DELETE FROM `foo_bar_relations` WHERE `foo_id` = ? AND `bar_id` = ?",
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("PostgresDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "foo_id", "bar_id" FROM "foo_bar_relations" WHERE `,
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES ($1, $2)`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Upsert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES ($1, $2) ON CONFLICT ("foo_id", "bar_id") DO NOTHING`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "foo_bar_relations" WHERE "foo_id" = $1 AND "bar_id" = $2`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("SqliteDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "foo_id", "bar_id" FROM "foo_bar_relations" WHERE `,
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?)`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Upsert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?) ON CONFLICT ("foo_id", "bar_id") DO NOTHING`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "foo_bar_relations" WHERE "foo_id" = ? AND "bar_id" = ?`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})
}

func TestQueryConstructionWithoutPrimaryKey(t *testing.T) {
	type relation struct {
		FooID int64 `db:"foo_id"`
		BarID int64 `db:"bar_id"`
	}
	opts := planOpts{
		StructTagKey: "db",
		TableName:    "foo_bar_relations",
	}

	t.Run("MariaDBDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       "SELECT `foo_id`, `bar_id` FROM `foo_bar_relations` WHERE ",
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           "INSERT INTO `foo_bar_relations` (`foo_id`, `bar_id`) VALUES (?, ?)",
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("PostgresDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "foo_id", "bar_id" FROM "foo_bar_relations" WHERE `,
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES ($1, $2)`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("SqliteDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[relation](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "foo_id", "bar_id" FROM "foo_bar_relations" WHERE `,
				ScanIndexes: [][]int{{0}, {1}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "foo_bar_relations" ("foo_id", "bar_id") VALUES (?, ?)`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})
}

func TestQueryConstructionImpossible(t *testing.T) {
	type unstructuredData struct {
		Foo int
		Bar *string
	}
	opts := planOpts{
		StructTagKey: "db",
	}

	testWith := func(dialect Dialect) func(*testing.T) {
		return func(t *testing.T) {
			p, err := buildPlan(reflect.TypeFor[unstructuredData](), dialect, opts)
			if err != nil {
				t.Error(err)
			}

			assert.Equal(t, onlyQueryPlans(p), plan{})
		}
	}

	t.Run("MariaDBDialect", testWith(MariaDBDialect()))
	t.Run("PostgresDialect", testWith(PostgresDialect()))
	t.Run("SqliteDialect", testWith(SqliteDialect()))
}

func TestQueryConstructionWithMultiplePrimaryKeyColumns(t *testing.T) {
	type record struct {
		GroupID   int64     `db:"group_id"`
		Name      string    `db:"name"`
		CreatedAt time.Time `db:"created_at"`
	}
	opts := planOpts{
		StructTagKey:          "db",
		TableName:             "complex_records",
		PrimaryKeyColumnNames: []string{"group_id", "name"},
	}

	t.Run("MariaDBDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       "SELECT `group_id`, `name`, `created_at` FROM `complex_records` WHERE ",
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           "INSERT INTO `complex_records` (`group_id`, `name`, `created_at`) VALUES (?, ?, ?)",
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Upsert: plannedQuery{
				Query:           "INSERT INTO `complex_records` (`group_id`, `name`, `created_at`) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE `created_at` = VALUES(`created_at`)",
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Update: plannedQuery{
				Query:           "UPDATE `complex_records` SET `created_at` = ? WHERE `group_id` = ? AND `name` = ?",
				ArgumentIndexes: [][]int{{2}, {0}, {1}},
			},
			Delete: plannedQuery{
				Query:           "DELETE FROM `complex_records` WHERE `group_id` = ? AND `name` = ?",
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("PostgresDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "group_id", "name", "created_at" FROM "complex_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "complex_records" ("group_id", "name", "created_at") VALUES ($1, $2, $3)`,
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Upsert: plannedQuery{
				Query:           `INSERT INTO "complex_records" ("group_id", "name", "created_at") VALUES ($1, $2, $3) ON CONFLICT ("group_id", "name") DO UPDATE SET "created_at" = EXCLUDED."created_at"`,
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "complex_records" SET "created_at" = $1 WHERE "group_id" = $2 AND "name" = $3`,
				ArgumentIndexes: [][]int{{2}, {0}, {1}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "complex_records" WHERE "group_id" = $1 AND "name" = $2`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})

	t.Run("SqliteDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			Select: plannedQuery{
				Query:       `SELECT "group_id", "name", "created_at" FROM "complex_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "complex_records" ("group_id", "name", "created_at") VALUES (?, ?, ?)`,
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Upsert: plannedQuery{
				Query:           `INSERT INTO "complex_records" ("group_id", "name", "created_at") VALUES (?, ?, ?) ON CONFLICT ("group_id", "name") DO UPDATE SET "created_at" = EXCLUDED."created_at"`,
				ArgumentIndexes: [][]int{{0}, {1}, {2}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "complex_records" SET "created_at" = ? WHERE "group_id" = ? AND "name" = ?`,
				ArgumentIndexes: [][]int{{2}, {0}, {1}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "complex_records" WHERE "group_id" = ? AND "name" = ?`,
				ArgumentIndexes: [][]int{{0}, {1}},
			},
		})
	})
}

func TestQueryConstructionWithMultipleAutoColumns(t *testing.T) {
	type record struct {
		ID        int64     `db:"id,auto"`
		Name      string    `db:"name"`
		CreatedAt time.Time `db:"created_at,auto"`
	}
	opts := planOpts{
		StructTagKey:          "db",
		TableName:             "autogenerated_records",
		PrimaryKeyColumnNames: []string{"id"},
	}

	t.Run("MariaDBDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), MariaDBDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			InsertUsesQueryRow: true,
			Select: plannedQuery{
				Query:       "SELECT `id`, `name`, `created_at` FROM `autogenerated_records` WHERE ",
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           "INSERT INTO `autogenerated_records` (`name`) VALUES (?) RETURNING `id`, `created_at`",
				ArgumentIndexes: [][]int{{1}},
				ScanIndexes:     [][]int{{0}, {2}},
			},
			Update: plannedQuery{
				Query:           "UPDATE `autogenerated_records` SET `name` = ?, `created_at` = ? WHERE `id` = ?",
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           "DELETE FROM `autogenerated_records` WHERE `id` = ?",
				ArgumentIndexes: [][]int{{0}},
			},
		})
	})

	t.Run("PostgresDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), PostgresDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			InsertUsesQueryRow: true,
			Select: plannedQuery{
				Query:       `SELECT "id", "name", "created_at" FROM "autogenerated_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "autogenerated_records" ("name") VALUES ($1) RETURNING "id", "created_at"`,
				ArgumentIndexes: [][]int{{1}},
				ScanIndexes:     [][]int{{0}, {2}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "autogenerated_records" SET "name" = $1, "created_at" = $2 WHERE "id" = $3`,
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "autogenerated_records" WHERE "id" = $1`,
				ArgumentIndexes: [][]int{{0}},
			},
		})
	})

	t.Run("SqliteDialect", func(t *testing.T) {
		p, err := buildPlan(reflect.TypeFor[record](), SqliteDialect(), opts)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, onlyQueryPlans(p), plan{
			InsertUsesQueryRow: true,
			Select: plannedQuery{
				Query:       `SELECT "id", "name", "created_at" FROM "autogenerated_records" WHERE `,
				ScanIndexes: [][]int{{0}, {1}, {2}},
			},
			Insert: plannedQuery{
				Query:           `INSERT INTO "autogenerated_records" ("name") VALUES (?) RETURNING "id", "created_at"`,
				ArgumentIndexes: [][]int{{1}},
				ScanIndexes:     [][]int{{0}, {2}},
			},
			Update: plannedQuery{
				Query:           `UPDATE "autogenerated_records" SET "name" = ?, "created_at" = ? WHERE "id" = ?`,
				ArgumentIndexes: [][]int{{1}, {2}, {0}},
			},
			Delete: plannedQuery{
				Query:           `DELETE FROM "autogenerated_records" WHERE "id" = ?`,
				ArgumentIndexes: [][]int{{0}},
			},
		})
	})
}

func TestPlanErrorCases(t *testing.T) {
	type recordUsedViaPointer struct {
		ID int64 `db:"id"`
	}

	_, err := NewStore[*recordUsedViaPointer](SqliteDialect())
	assert.Equal(t, err.Error(), `cannot use type *oblast.recordUsedViaPointer for queries: `+
		`expected struct type, but got kind "ptr"`)

	type recordWithDuplicateTags struct {
		Foo int64 `db:"Bar"`
		Qux float64
		Bar string
	}
	_, err = NewStore[recordWithDuplicateTags](SqliteDialect())
	assert.Equal(t, err.Error(), `cannot use type oblast.recordWithDuplicateTags for queries: `+
		"duplicate tag `db:\"Bar\"` on field index [0], but also on field index [2]")

	type recordWithUnusedTransparentStruct struct {
		ID        int64
		CreatedAt time.Time // has no exported fields!
	}
	_, err = NewStore[recordWithUnusedTransparentStruct](SqliteDialect())
	assert.Equal(t, err.Error(), `cannot use type oblast.recordWithUnusedTransparentStruct for queries: `+
		"field \"CreatedAt\" of type time.Time does not contain any mapped fields (to map this whole field to a DB column, add an explicit `db:\"...\"` tag)")

	type recordWithPKButNoTableName struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	_, err = NewStore[recordWithPKButNoTableName](SqliteDialect(),
		PrimaryKeyIs("id"),
	)
	assert.Equal(t, err.Error(), `cannot use type oblast.recordWithPKButNoTableName for queries: `+
		`cannot declare a primary key without also providing the TableNameIs option`)

	type recordWithUnknownPK struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
	}
	_, err = NewStore[recordWithUnknownPK](SqliteDialect(),
		TableNameIs("records"),
		PrimaryKeyIs("record_id"),
	)
	assert.Equal(t, err.Error(), `cannot use type oblast.recordWithUnknownPK for queries: `+
		"no field has tag `db:\"record_id\"`, but a field of this name was declared in the primary key")

	type recordWithWeirdTagOption struct {
		ID          int64  `db:",auto"`
		Name        string `db:",unique"`
		Description string
	}
	_, err = NewStore[recordWithWeirdTagOption](SqliteDialect())
	assert.Equal(t, err.Error(), `cannot use type oblast.recordWithWeirdTagOption for queries: `+
		"unknown option `db:\",unique\"` on field \"Name\"")
}
