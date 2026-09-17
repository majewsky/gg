// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-gorp/gorp/v3"
	_ "github.com/mattn/go-sqlite3"
	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/benchmark/internal/testhelpers/must"
	"go.xyrillian.de/gg/gsql"
	"go.xyrillian.de/gg/oblast"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NOTE: In this file, we benchmark different ORMs against each other and against hand-written operations using plain database/sql.
// All benchmarks are called "BenchmarkORM...".

// Do not use b.Context() within benchmarks, or you will merely demonstrate that using a deep stack of Context objects is expensive.
var noctx = context.Background()

// This is not a real benchmark (obviously).
// Its purpose is to be the first line that is printed, while having one of the longest names,
// so that all other results are aligned with it and the table looks nice.
func BenchmarkORMHeadingHeadingHeadingHeadingHeadingHeadingHeadingHeading(b *testing.B) {
	for b.Loop() {
		time.Sleep(time.Microsecond)
	}
}

var (
	totalRecordCountForSelect = 10000
	batchSizesForSelect       = []int{1, 10, 100, 1000}
	batchSizesForInsertDelete = []int{1, 2, 4, 8, 16, 100}
	batchSizesForUpdate       = []int{1, 2, 4, 8, 16, 100}
)

func makeSqliteTestDB(t testing.TB, recordCount int) (db *gsql.DB, dsn string) {
	dsn = fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db = gsql.NewDB(must.Return(sql.Open("sqlite3", dsn))(t))
	_ = must.Return(db.Exec(`CREATE TABLE entries (id INTEGER, message TEXT, PRIMARY KEY (id AUTOINCREMENT))`))(t)

	if recordCount > 0 {
		// fill in some random-looking, but deterministic data
		stmt := must.Return(db.Prepare(`INSERT INTO entries (id, message) VALUES (?, ?)`))(t)
		for idx := range recordCount {
			buf := sha256.Sum256([]byte(strconv.Itoa(idx)))
			_ = must.Return(stmt.Exec(idx, fmt.Sprintf("sha256:%x", buf[:])))(t)
		}
		must.Succeed(t, stmt.Close())
	}

	return db, dsn
}

type OblastEntry struct {
	ID      int    `db:"id,auto"`
	Message string `db:"message"`
}

type GorpEntry struct {
	ID      int    `db:"id"`
	Message string `db:"message"`
}

type GormEntry struct {
	ID      int `gorm:"primaryKey"`
	Message string
}

func (GormEntry) TableName() string { return "entries" }

func BenchmarkORMSelectMany(b *testing.B) {
	db, dsn := makeSqliteTestDB(b, totalRecordCountForSelect)

	// test with different sizes of resultsets (N=1 is an OLTP-like workload,
	// then the larger N lean more towards the OLAP side of things)
	for _, batchSize := range batchSizesForSelect {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			// prepare the functions that will be benched
			store := oblast.MustNewStore[OblastEntry](
				oblast.SqliteDialect(),
				oblast.TableNameIs("entries"),
				oblast.PrimaryKeyIs("id"),
			)
			gorpDB := gorp.DbMap{Db: db.DB, Dialect: gorp.SqliteDialect{}}
			gormDB := must.Return(gorm.Open(sqlite.Open(dsn), &gorm.Config{}))(b)
			partialQuery := `id < ` + strconv.Itoa(batchSize)
			query := `SELECT * FROM entries WHERE ` + partialQuery
			precomputedQuery := store.MustPrepareSelectQueryWhere(partialQuery)

			selectWithOblast := func(b *testing.B) {
				records := must.Return(store.Select(noctx, db, query).Collect())(b)
				assert.Equal(b, len(records), batchSize)
			}

			selectWithOblastWhere := func(b *testing.B) {
				records := must.Return(precomputedQuery.Select(noctx, db).Collect())(b)
				assert.Equal(b, len(records), batchSize)
			}

			selectWithGorp := func(b *testing.B) {
				var records []GorpEntry
				_ = must.Return(gorpDB.Select(&records, query))(b)
				assert.Equal(b, len(records), batchSize)
			}

			selectWithGorm := func(b *testing.B) {
				records := must.Return(gorm.G[GormEntry](gormDB).Where(partialQuery).Find(b.Context()))(b)
				assert.Equal(b, len(records), batchSize)
			}

			selectWithSqlite := func(b *testing.B) {
				var count int
				rows := must.Return(db.Query(query))(b) //nolint:rowserrcheck // false positive
				var (
					id      int64
					message string
				)
				for rows.Next() {
					must.Succeed(b, rows.Scan(&id, &message))
					if id != 20000 && message != "" { // always true; ensures that values are not optimized away
						count++
					}
				}
				must.Succeed(b, rows.Close())
				assert.Equal(b, count, batchSize)
			}

			// run once to prewarm caches (if any)
			selectWithOblast(b)
			selectWithGorp(b)
			selectWithGorm(b)
			if b.Failed() {
				b.FailNow()
			}

			// run actual benchmark
			b.Run("via Gorm using Find", func(b *testing.B) {
				for b.Loop() {
					selectWithGorm(b)
				}
			})
			b.Run("via Gorp using Select", func(b *testing.B) {
				for b.Loop() {
					selectWithGorp(b)
				}
			})
			b.Run("via Oblast using Select", func(b *testing.B) {
				for b.Loop() {
					selectWithOblast(b)
				}
			})
			b.Run("via Oblast using SelectWhere", func(b *testing.B) {
				for b.Loop() {
					selectWithOblastWhere(b)
				}
			})
			b.Run("just SQLite", func(b *testing.B) {
				for b.Loop() {
					selectWithSqlite(b)
				}
			})
		})
	}
}

func BenchmarkORMSelectManyValues(b *testing.B) {
	db, _ := makeSqliteTestDB(b, totalRecordCountForSelect)

	// test with different sizes of resultsets (N=1 is an OLTP-like workload,
	// then the larger N lean more towards the OLAP side of things)
	for _, batchSize := range batchSizesForSelect {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			// prepare the functions that will be benched
			query := `SELECT message FROM entries WHERE id < ` + strconv.Itoa(batchSize)
			selectWithOblast := func(b *testing.B) {
				messages := must.Return(oblast.Select[string](noctx, db, query).Collect())(b)
				assert.Equal(b, len(messages), batchSize)
			}
			selectWithSqlite := func(b *testing.B) {
				var count int
				rows := must.Return(db.Query(query))(b) //nolint:rowserrcheck // false positive
				var message string
				for rows.Next() {
					must.Succeed(b, rows.Scan(&message))
					count++
				}
				must.Succeed(b, rows.Close())
				assert.Equal(b, count, batchSize)
			}

			// run actual benchmark
			b.Run("via Oblast", func(b *testing.B) {
				for b.Loop() {
					selectWithOblast(b)
				}
			})
			b.Run("just SQLite", func(b *testing.B) {
				for b.Loop() {
					selectWithSqlite(b)
				}
			})
		})
	}
}

func BenchmarkORMSelectOne(b *testing.B) {
	db, dsn := makeSqliteTestDB(b, totalRecordCountForSelect)

	// grab a "random" record from the DB, not just the first or the last
	recordID := min(totalRecordCountForSelect*2/3, totalRecordCountForSelect)

	// prepare the functions that will be benched
	store := oblast.MustNewStore[OblastEntry](
		oblast.SqliteDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)
	gorpDB := gorp.DbMap{Db: db.DB, Dialect: gorp.SqliteDialect{}}
	gormDB := must.Return(gorm.Open(sqlite.Open(dsn), &gorm.Config{}))(b)
	partialQuery := `id = ` + strconv.Itoa(recordID)
	query := `SELECT * FROM entries WHERE ` + partialQuery
	precomputedQuery := store.MustPrepareSelectQueryWhere(partialQuery)

	selectWithOblast := func(b *testing.B) {
		r := must.Return(store.SelectOne(noctx, db, query))(b)
		assert.Equal(b, r.ID, recordID)
	}

	selectWithOblastWhere := func(b *testing.B) {
		r := must.Return(precomputedQuery.SelectOne(noctx, db))(b)
		assert.Equal(b, r.ID, recordID)
	}

	selectWithGorp := func(b *testing.B) {
		var r GorpEntry
		must.Succeed(b, gorpDB.SelectOne(&r, query))
		assert.Equal(b, r.ID, recordID)
	}

	selectWithGorm := func(b *testing.B) {
		r := must.Return(gorm.G[GormEntry](gormDB).Where(partialQuery).First(b.Context()))(b)
		assert.Equal(b, r.ID, recordID)
	}

	selectWithSqlite := func(b *testing.B) {
		var (
			id      int64
			message string
		)
		must.Succeed(b, db.QueryRow(query).Scan(&id, &message))
		assert.Equal(b, id, int64(recordID))
	}

	// run once to prewarm caches (if any)
	selectWithOblast(b)
	selectWithGorp(b)
	selectWithGorm(b)
	if b.Failed() {
		b.FailNow()
	}

	// run actual benchmark
	b.Run("via Gorm using First", func(b *testing.B) {
		for b.Loop() {
			selectWithGorm(b)
		}
	})
	b.Run("via Gorp using SelectOne", func(b *testing.B) {
		for b.Loop() {
			selectWithGorp(b)
		}
	})
	b.Run("via Oblast using SelectOne", func(b *testing.B) {
		for b.Loop() {
			selectWithOblast(b)
		}
	})
	b.Run("via Oblast using SelectOneWhere", func(b *testing.B) {
		for b.Loop() {
			selectWithOblastWhere(b)
		}
	})
	b.Run("just SQLite", func(b *testing.B) {
		for b.Loop() {
			selectWithSqlite(b)
		}
	})
}

func BenchmarkORMSelectOneValue(b *testing.B) {
	db, _ := makeSqliteTestDB(b, totalRecordCountForSelect)

	// grab a "random" record from the DB, not just the first or the last
	recordID := min(totalRecordCountForSelect*2/3, totalRecordCountForSelect)

	// prepare the functions that will be benched
	query := `SELECT message FROM entries WHERE id = ` + strconv.Itoa(recordID)
	selectWithOblast := func(b *testing.B) {
		message := must.Return(oblast.SelectOne[string](noctx, db, query))(b)
		assert.Equal(b, len(message), 71)
	}
	selectWithSqlite := func(b *testing.B) {
		var message string
		must.Succeed(b, db.QueryRow(query).Scan(&message))
		assert.Equal(b, len(message), 71)
	}

	// run actual benchmark
	b.Run("via Oblast", func(b *testing.B) {
		for b.Loop() {
			selectWithOblast(b)
		}
	})
	b.Run("just SQLite", func(b *testing.B) {
		for b.Loop() {
			selectWithSqlite(b)
		}
	})
}

func BenchmarkORMInsertAndDelete(b *testing.B) {
	db, dsn := makeSqliteTestDB(b, 0)

	store := oblast.MustNewStore[OblastEntry](
		oblast.SqliteDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)
	gorpDB := gorp.DbMap{Db: db.DB, Dialect: gorp.SqliteDialect{}}
	gorpDB.AddTableWithName(GorpEntry{}, "entries").SetKeys(true, "id")
	gormDB := must.Return(gorm.Open(sqlite.Open(dsn), &gorm.Config{}))(b)

	// test with different amounts of records
	for _, batchSize := range batchSizesForInsertDelete {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			// prepare the functions that will be benched
			insertAndDeleteWithOblast := func(b *testing.B) {
				records := make([]OblastEntry, batchSize)
				recordsForInsert := make([]*OblastEntry, batchSize)
				for idx := range records {
					records[idx] = OblastEntry{Message: "hello"}
					recordsForInsert[idx] = &records[idx]
				}
				must.Succeed(b, store.Insert(noctx, db, recordsForInsert...))
				for _, r := range records {
					if r.ID == 0 {
						b.Errorf("ID was not filled!")
					}
				}
				must.Succeed(b, store.Delete(noctx, db, records...))
			}
			if batchSize == 1 {
				insertAndDeleteWithOblast = func(b *testing.B) {
					record := OblastEntry{Message: "hello"}
					must.Succeed(b, store.Insert(noctx, db, &record))
					if record.ID == 0 {
						b.Errorf("ID was not filled!")
					}
					must.Succeed(b, store.Delete(noctx, db, record))
				}
			}

			insertAndDeleteWithGorp := func(b *testing.B) {
				records := make([]any, batchSize)
				for idx := range records {
					records[idx] = &GorpEntry{Message: "hello"}
				}
				must.Succeed(b, gorpDB.Insert(records...))
				for _, r := range records {
					if r.(*GorpEntry).ID == 0 {
						b.Errorf("ID was not filled!")
					}
				}
				_ = must.Return(gorpDB.Delete(records...))(b)
			}
			if batchSize == 1 {
				insertAndDeleteWithGorp = func(b *testing.B) {
					record := GorpEntry{Message: "hello"}
					must.Succeed(b, gorpDB.Insert(&record))
					if record.ID == 0 {
						b.Errorf("ID was not filled!")
					}
					_ = must.Return(gorpDB.Delete(&record))(b)
				}
			}

			insertAndDeleteWithGorm := func(b *testing.B) {
				records := make([]GormEntry, batchSize)
				for idx := range records {
					records[idx] = GormEntry{Message: "hello"}
				}
				must.Succeed(b, gorm.G[GormEntry](gormDB).CreateInBatches(b.Context(), &records, batchSize))
				for _, r := range records {
					if r.ID == 0 {
						b.Errorf("ID was not filled!")
					}
				}
				result := gormDB.Delete(&records)
				assert.ErrEqual(b, result.Error, nil)
				assert.Equal(b, result.RowsAffected, int64(batchSize))
			}
			if batchSize == 1 {
				insertAndDeleteWithGorm = func(b *testing.B) {
					record := GormEntry{Message: "hello"}
					must.Succeed(b, gorm.G[GormEntry](gormDB).Create(b.Context(), &record))
					result := gormDB.Delete(&record)
					assert.ErrEqual(b, result.Error, nil)
					assert.Equal(b, result.RowsAffected, 1)
				}
			}

			insertAndDeleteWithStraightExec := func(b *testing.B) {
				ids := make([]int64, batchSize)
				for idx := range ids {
					result := must.Return(db.Exec(`INSERT INTO entries (message) VALUES (?)`, "hello"))(b)
					ids[idx] = must.Return(result.LastInsertId())(b)
				}
				for _, id := range ids {
					_ = must.Return(db.Exec(`DELETE FROM entries WHERE id = ?`, id))(b)
				}
			}

			insertAndDeleteWithPreparedExec := func(b *testing.B) {
				ids := make([]int64, batchSize)
				stmtInsert := must.Return(db.Prepare(`INSERT INTO entries (message) VALUES (?)`))(b)
				defer stmtInsert.Close()
				for idx := range ids {
					result := must.Return(stmtInsert.Exec("hello"))(b)
					ids[idx] = must.Return(result.LastInsertId())(b)
				}
				stmtDelete := must.Return(db.Prepare(`DELETE FROM entries WHERE id = ?`))(b)
				defer stmtDelete.Close()
				for _, id := range ids {
					_ = must.Return(stmtDelete.Exec(id))(b)
				}
			}

			insertAndDeleteWithStraightQueryRow := func(b *testing.B) {
				ids := make([]int64, batchSize)
				for idx := range ids {
					must.Succeed(b, db.QueryRow(`INSERT INTO entries (message) VALUES (?) RETURNING id`, "hello").Scan(&ids[idx]))
				}
				for _, id := range ids {
					_ = must.Return(db.Exec(`DELETE FROM entries WHERE id = ?`, id))(b)
				}
			}

			insertAndDeleteWithPreparedQueryRow := func(b *testing.B) {
				ids := make([]int64, batchSize)
				stmtInsert := must.Return(db.Prepare(`INSERT INTO entries (message) VALUES (?) RETURNING id`))(b)
				defer stmtInsert.Close()
				for idx := range ids {
					must.Succeed(b, stmtInsert.QueryRow("hello").Scan(&ids[idx]))
				}
				stmtDelete := must.Return(db.Prepare(`DELETE FROM entries WHERE id = ?`))(b)
				defer stmtDelete.Close()
				for _, id := range ids {
					_ = must.Return(stmtDelete.Exec(id))(b)
				}
			}

			// run once to prewarm caches (if any)
			insertAndDeleteWithOblast(b)
			insertAndDeleteWithGorp(b)
			insertAndDeleteWithGorm(b)

			b.Run("via Gorm", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithGorm(b)
				}
			})
			b.Run("via Gorp", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithGorp(b)
				}
			})
			b.Run("via Oblast", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithOblast(b)
				}
			})
			b.Run("just SQLite (straight Exec)", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithStraightExec(b)
				}
			})
			b.Run("just SQLite (prepared Exec)", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithPreparedExec(b)
				}
			})
			b.Run("just SQLite (straight QueryRow)", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithStraightQueryRow(b)
				}
			})
			b.Run("just SQLite (prepared QueryRow)", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithPreparedQueryRow(b)
				}
			})
		})
	}
}

func BenchmarkORMUpdate(b *testing.B) {
	db, dsn := makeSqliteTestDB(b, 0)

	store := oblast.MustNewStore[OblastEntry](
		oblast.SqliteDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)
	gorpDB := gorp.DbMap{Db: db.DB, Dialect: gorp.SqliteDialect{}}
	gorpDB.AddTableWithName(GorpEntry{}, "entries").SetKeys(true, "id")
	gormDB := must.Return(gorm.Open(sqlite.Open(dsn), &gorm.Config{}))(b)

	// test with different amounts of records
	for _, batchSize := range batchSizesForUpdate {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			// prepare a bunch of records that we can update, in a reproducible way
			_ = must.Return(db.Exec(`DELETE FROM entries`))
			recordsForOblast := make([]OblastEntry, batchSize)
			recordsForOblastForInsert := make([]*OblastEntry, batchSize)
			for idx := range recordsForOblast {
				recordsForOblast[idx] = OblastEntry{Message: "hello"}
				recordsForOblastForInsert[idx] = &recordsForOblast[idx]
			}
			must.Succeed(b, store.Insert(noctx, db, recordsForOblastForInsert...))
			recordsForGorp := make([]any, batchSize)
			for idx, r := range recordsForOblast {
				recordsForGorp[idx] = new(GorpEntry(r))
			}
			recordsForGorm := make([]GormEntry, batchSize)
			for idx, r := range recordsForOblast {
				recordsForGorm[idx] = GormEntry(r)
			}

			// prepare the functions that will be benched
			updateWithOblast := func(b *testing.B, message string) {
				for idx := range recordsForOblast {
					recordsForOblast[idx].Message = message
				}
				must.Succeed(b, store.Update(noctx, db, recordsForOblast...))
			}
			updateWithGorp := func(b *testing.B, message string) {
				for _, r := range recordsForGorp {
					r.(*GorpEntry).Message = message
				}
				_ = must.Return(gorpDB.Update(recordsForGorp...))(b)
			}
			updateWithGorm := func(b *testing.B, message string) {
				for idx := range recordsForGorm {
					recordsForGorm[idx].Message = message
				}
				result := gormDB.Save(&recordsForGorm)
				assert.ErrEqual(b, result.Error, nil)
				assert.Equal(b, result.RowsAffected, int64(batchSize))
			}
			updateWithStraightSqlite := func(b *testing.B, message string) {
				for _, r := range recordsForOblast {
					_ = must.Return(db.Exec(`UPDATE entries SET message = ? WHERE id = ?`, message, r.ID))(b)
				}
			}
			updateWithPreparedSqlite := func(b *testing.B, message string) {
				stmt := must.Return(db.Prepare(`UPDATE entries SET message = ? WHERE id = ?`))(b)
				for _, r := range recordsForOblast {
					_ = must.Return(stmt.Exec(message, r.ID))(b)
				}
				must.Succeed(b, stmt.Close())
			}
			checkRecordsUpdated := func(b *testing.B, message string) {
				var count int64
				must.Succeed(b, db.QueryRow(`SELECT COUNT(*) FROM entries WHERE message = ?`, message).Scan(&count))
				assert.Equal(b, count, int64(batchSize))
			}

			// run once to prewarm caches (if any)
			updateWithGorm(b, "warming up")
			updateWithGorp(b, "warming up")
			updateWithOblast(b, "warming up")

			b.Run("via Gorm", func(b *testing.B) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					updateWithGorm(b, message)
					checkRecordsUpdated(b, message)
				}
			})
			b.Run("via Gorp", func(b *testing.B) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					updateWithGorp(b, message)
					checkRecordsUpdated(b, message)
				}
			})
			b.Run("via Oblast", func(b *testing.B) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					updateWithOblast(b, message)
					checkRecordsUpdated(b, message)
				}
			})
			b.Run("just SQLite (straight)", func(b *testing.B) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					updateWithStraightSqlite(b, message)
					checkRecordsUpdated(b, message)
				}
			})
			b.Run("just SQLite (prepared)", func(b *testing.B) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					updateWithPreparedSqlite(b, message)
					checkRecordsUpdated(b, message)
				}
			})
		})
	}
}
