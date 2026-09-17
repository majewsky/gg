// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package main_test

import (
	"cmp"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	gg_pgx "go.xyrillian.de/gg-pgx"
	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/benchmark/internal/testhelpers/must"
	"go.xyrillian.de/gg/gsql"
	"go.xyrillian.de/gg/oblast"
)

// NOTE: In this file, we benchmark different PostgreSQL database drivers against each other with or without Oblast in between.
// All benchmarks are called "BenchmarkPostgres...".
// To run these benchmarks, you need to have provide a DSN to a PostgreSQL database in $BENCHMARK_POSTGRES_DSN.

// This is not a real benchmark (obviously).
// Its purpose is to be the first line that is printed, while having one of the longest names,
// so that all other results are aligned with it and the table looks nice.
func BenchmarkPostgresHeadingHeadingHeadingHeadingHeadingHeadingHeadingHeading(b *testing.B) {
	for b.Loop() {
		time.Sleep(time.Microsecond)
	}
}

const defaultPostgresDSN = "host=localhost user=postgres dbname=oblast_benchmark sslmode=disable"

func connectToPostgresTestDB(t testing.TB, recordCount int) *gsql.DB {
	dsn := cmp.Or(os.Getenv("BENCHMARK_POSTGRES_DSN"), defaultPostgresDSN)
	db := gsql.NewDB(must.Return(sql.Open("postgres", dsn))(t))
	_ = must.Return(db.Exec(`CREATE TEMPORARY TABLE entries (id BIGSERIAL, message TEXT)`))(t)

	if recordCount > 0 {
		// fill in some random-looking, but deterministic data
		stmt := must.Return(db.Prepare(`INSERT INTO entries (id, message) VALUES ($1, $2)`))(t)
		for idx := range recordCount {
			buf := sha256.Sum256([]byte(strconv.Itoa(idx)))
			_ = must.Return(stmt.Exec(idx, fmt.Sprintf("sha256:%x", buf[:])))(t)
		}
		must.Succeed(t, stmt.Close())
	}

	return db
}

func connectToPgxTestDB(t testing.TB, recordCount int) *pgx.Conn {
	ctx := t.Context()
	dsn := cmp.Or(os.Getenv("BENCHMARK_POSTGRES_DSN"), defaultPostgresDSN)
	conn := must.Return(pgx.Connect(ctx, dsn))(t)
	_ = must.Return(conn.Exec(ctx, `CREATE TEMPORARY TABLE entries (id BIGSERIAL, message TEXT)`))(t)

	if recordCount > 0 {
		// fill in some random-looking, but deterministic data
		query := `INSERT INTO entries (id, message) VALUES ($1, $2)`
		stmt := must.Return(conn.Prepare(ctx, query, query))(t)
		for idx := range recordCount {
			buf := sha256.Sum256([]byte(strconv.Itoa(idx)))
			_ = must.Return(conn.Exec(ctx, query, idx, fmt.Sprintf("sha256:%x", buf[:])))(t)
		}
		must.Succeed(t, conn.Deallocate(ctx, stmt.Name))
	}

	return conn
}

func BenchmarkPostgresSelect(b *testing.B) {
	pqDB := connectToPostgresTestDB(b, totalRecordCountForSelect)
	pgxConn := connectToPgxTestDB(b, totalRecordCountForSelect)
	pgxConnH := gg_pgx.NewConn(pgxConn)

	store := oblast.MustNewStore[OblastEntry](
		oblast.PostgresDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)

	for _, batchSize := range batchSizesForSelect {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			partialQuery := `id < ` + strconv.Itoa(batchSize)
			query := `SELECT * FROM entries WHERE ` + partialQuery

			b.Run("driver=pq/strategy=oblast", func(b *testing.B) {
				for b.Loop() {
					records := must.Return(store.Select(noctx, pqDB, query).Collect())(b)
					assert.Equal(b, len(records), batchSize)
				}
			})

			b.Run("driver=pgx/strategy=oblast", func(b *testing.B) {
				for b.Loop() {
					records := must.Return(store.Select(noctx, pgxConnH, query).Collect())(b)
					assert.Equal(b, len(records), batchSize)
				}
			})

			b.Run("driver=pq/strategy=straight", func(b *testing.B) {
				for b.Loop() {
					var records []OblastEntry
					rows := must.Return(pqDB.Query(query))(b) //nolint:rowserrcheck // false positive
					for rows.Next() {
						var e OblastEntry
						must.Succeed(b, rows.Scan(&e.ID, &e.Message))
						records = append(records, e)
					}
					must.Succeed(b, rows.Close())
					assert.Equal(b, len(records), batchSize)
				}
			})

			b.Run("driver=pgx/strategy=straight", func(b *testing.B) {
				for b.Loop() {
					var records []OblastEntry
					rows := must.Return(pgxConn.Query(noctx, query))(b)
					for rows.Next() {
						var e OblastEntry
						must.Succeed(b, rows.Scan(&e.ID, &e.Message))
						records = append(records, e)
					}
					rows.Close()
					assert.Equal(b, len(records), batchSize)
				}
			})
		})
	}
}

func BenchmarkPostgresSelectOne(b *testing.B) {
	pqDB := connectToPostgresTestDB(b, totalRecordCountForSelect)
	pgxConn := connectToPgxTestDB(b, totalRecordCountForSelect)
	pgxConnH := gg_pgx.NewConn(pgxConn)

	// grab a "random" record from the DB, not just the first or the last
	recordID := min(totalRecordCountForSelect*2/3, totalRecordCountForSelect)

	store := oblast.MustNewStore[OblastEntry](
		oblast.PostgresDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)

	partialQuery := `id = ` + strconv.Itoa(recordID)
	query := `SELECT * FROM entries WHERE ` + partialQuery
	precomputedQuery := store.MustPrepareSelectQueryWhere(partialQuery)

	b.Run("driver=pq/strategy=oblast", func(b *testing.B) {
		for b.Loop() {
			r := must.Return(precomputedQuery.SelectOne(noctx, pqDB))(b)
			assert.Equal(b, r.ID, recordID)
		}
	})

	b.Run("driver=pgx/strategy=oblast", func(b *testing.B) {
		for b.Loop() {
			r := must.Return(precomputedQuery.SelectOne(noctx, pgxConnH))(b)
			assert.Equal(b, r.ID, recordID)
		}
	})

	b.Run("driver=pq/strategy=straight", func(b *testing.B) {
		for b.Loop() {
			var (
				id      int64
				message string
			)
			must.Succeed(b, pqDB.QueryRow(query).Scan(&id, &message))
			assert.Equal(b, id, int64(recordID))
		}
	})

	b.Run("driver=pgx/strategy=straight", func(b *testing.B) {
		for b.Loop() {
			var (
				id      int64
				message string
			)
			must.Succeed(b, pgxConn.QueryRow(noctx, query).Scan(&id, &message))
			assert.Equal(b, id, int64(recordID))
		}
	})
}

func BenchmarkPostgresInsertAndDelete(b *testing.B) {
	pqDB := connectToPostgresTestDB(b, 0)
	pgxConn := connectToPgxTestDB(b, 0)
	pgxConnH := gg_pgx.NewConn(pgxConn)

	store := oblast.MustNewStore[OblastEntry](
		oblast.PostgresDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)

	// test with different amounts of records
	for _, batchSize := range batchSizesForInsertDelete {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			insertAndDeleteWithOblast := func(b *testing.B, dbh gsql.Handle) {
				records := make([]OblastEntry, batchSize)
				recordsForInsert := make([]*OblastEntry, batchSize)
				for idx := range records {
					records[idx] = OblastEntry{Message: "hello"}
					recordsForInsert[idx] = &records[idx]
				}
				must.Succeed(b, store.Insert(noctx, dbh, recordsForInsert...))
				for _, r := range records {
					if r.ID == 0 {
						b.Errorf("ID was not filled!")
					}
				}
				must.Succeed(b, store.Delete(noctx, dbh, records...))
			}

			b.Run("driver=pq/strategy=oblast", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithOblast(b, pqDB)
				}
			})

			b.Run("driver=pgx/strategy=oblast", func(b *testing.B) {
				for b.Loop() {
					insertAndDeleteWithOblast(b, pgxConnH)
				}
			})

			insertQuery := `INSERT INTO entries (message) VALUES ($1) RETURNING id`
			deleteQuery := `DELETE FROM entries WHERE id = $1`

			b.Run("driver=pq/strategy=straight", func(b *testing.B) {
				for b.Loop() {
					ids := make([]int64, batchSize)
					for idx := range ids {
						must.Succeed(b, pqDB.QueryRow(insertQuery, "hello").Scan(&ids[idx]))
					}
					for _, id := range ids {
						_ = must.Return(pqDB.Exec(deleteQuery, id))(b)
					}
				}
			})

			b.Run("driver=pgx/strategy=straight", func(b *testing.B) {
				for b.Loop() {
					ids := make([]int64, batchSize)
					for idx := range ids {
						must.Succeed(b, pgxConn.QueryRow(noctx, insertQuery, "hello").Scan(&ids[idx]))
					}
					for _, id := range ids {
						_ = must.Return(pgxConn.Exec(noctx, deleteQuery, id))(b)
					}
				}
			})

			b.Run("driver=pq/strategy=prepared", func(b *testing.B) {
				for b.Loop() {
					ids := make([]int64, batchSize)
					stmtInsert := must.Return(pqDB.Prepare(insertQuery))(b)
					defer stmtInsert.Close()
					for idx := range ids {
						must.Succeed(b, stmtInsert.QueryRow("hello").Scan(&ids[idx]))
					}
					stmtDelete := must.Return(pqDB.Prepare(deleteQuery))(b)
					defer stmtDelete.Close()
					for _, id := range ids {
						_ = must.Return(stmtDelete.Exec(id))(b)
					}
				}
			})

			b.Run("driver=pgx/strategy=prepared", func(b *testing.B) {
				for b.Loop() {
					stmtInsert := must.Return(pgxConn.Prepare(noctx, "my-insert", insertQuery))(b)
					ids := make([]int64, batchSize)
					for idx := range ids {
						must.Succeed(b, pgxConn.QueryRow(noctx, stmtInsert.Name, "hello").Scan(&ids[idx]))
					}
					must.Succeed(b, pgxConn.Deallocate(noctx, stmtInsert.Name))
					stmtDelete := must.Return(pgxConn.Prepare(noctx, "my-delete", deleteQuery))(b)
					for _, id := range ids {
						_ = must.Return(pgxConn.Exec(noctx, stmtDelete.Name, id))(b)
					}
					must.Succeed(b, pgxConn.Deallocate(noctx, stmtDelete.Name))
				}
			})
		})
	}
}

func BenchmarkPostgresUpdate(b *testing.B) {
	pqDB := connectToPostgresTestDB(b, 0)
	pgxConn := connectToPgxTestDB(b, 0)
	pgxConnH := gg_pgx.NewConn(pgxConn)

	store := oblast.MustNewStore[OblastEntry](
		oblast.PostgresDialect(),
		oblast.TableNameIs("entries"),
		oblast.PrimaryKeyIs("id"),
	)

	// test with different amounts of records
	for _, batchSize := range batchSizesForInsertDelete {
		b.Run("N="+strconv.Itoa(batchSize), func(b *testing.B) {
			// prepare a bunch of records that we can update, in a reproducible way
			_ = must.Return(pqDB.Exec(`DELETE FROM entries`))
			_ = must.Return(pgxConn.Exec(noctx, `DELETE FROM entries`))
			pqRecords := make([]OblastEntry, batchSize)
			pqRecordsForInsert := make([]*OblastEntry, batchSize)
			pgxRecords := make([]OblastEntry, batchSize)
			pgxRecordsForInsert := make([]*OblastEntry, batchSize)
			for idx := range batchSize {
				pqRecords[idx] = OblastEntry{Message: "hello"}
				pqRecordsForInsert[idx] = &pqRecords[idx]
				pgxRecords[idx] = OblastEntry{Message: "hello"}
				pgxRecordsForInsert[idx] = &pgxRecords[idx]
			}
			must.Succeed(b, store.Insert(noctx, pqDB, pqRecordsForInsert...))
			must.Succeed(b, store.Insert(noctx, pgxConnH, pgxRecordsForInsert...))

			// each benchmark will, while looping, write changing values each time in the same way
			loop := func(b *testing.B, action func(string)) {
				idx := 0
				for b.Loop() {
					idx++
					message := fmt.Sprintf("round %d", idx)
					action(message)
				}
			}

			updateWithOblast := func(b *testing.B, dbh gsql.Handle, records []OblastEntry) func(string) {
				return func(message string) {
					for idx := range records {
						records[idx].Message = message
					}
					must.Succeed(b, store.Update(noctx, dbh, records...))
				}
			}

			b.Run("driver=pq/strategy=oblast", func(b *testing.B) {
				loop(b, updateWithOblast(b, pqDB, pqRecords))
			})

			b.Run("driver=pgx/strategy=oblast", func(b *testing.B) {
				loop(b, updateWithOblast(b, pgxConnH, pgxRecords))
			})

			updateQuery := `UPDATE entries SET message = $1 WHERE id = $2`

			b.Run("driver=pq/strategy=straight", func(b *testing.B) {
				loop(b, func(message string) {
					for _, r := range pqRecords {
						_ = must.Return(pqDB.Exec(updateQuery, message, r.ID))(b)
					}
				})
			})

			b.Run("driver=pgx/strategy=straight", func(b *testing.B) {
				loop(b, func(message string) {
					for _, r := range pgxRecords {
						_ = must.Return(pgxConn.Exec(noctx, updateQuery, message, r.ID))(b)
					}
				})
			})

			b.Run("driver=pq/strategy=prepared", func(b *testing.B) {
				loop(b, func(message string) {
					stmt := must.Return(pqDB.Prepare(updateQuery))(b)
					for _, r := range pqRecords {
						_ = must.Return(stmt.Exec(message, r.ID))(b)
					}
				})
			})

			b.Run("driver=pgx/strategy=prepared", func(b *testing.B) {
				loop(b, func(message string) {
					stmt := must.Return(pgxConn.Prepare(noctx, "my-update", updateQuery))(b)
					for _, r := range pgxRecords {
						_ = must.Return(pgxConn.Exec(noctx, stmt.Name, message, r.ID))(b)
					}
					must.Succeed(b, pgxConn.Deallocate(noctx, stmt.Name))
				})
			})
		})
	}
}
