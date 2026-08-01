// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/pgruntime"
)

func TestMigrations(t *testing.T) {
	ctx := t.Context()

	// reset the test DB to empty if it exists
	db, _ := connector.ConnectForTest(t, defaultBehavior)
	for _, tableName := range []string{"comments", "posts", "schema_migrations"} {
		_, err := execQuery(ctx, db, `DROP TABLE IF EXISTS `+tableName, nil)
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	err := db.Close()
	if err != nil {
		t.Fatal(err.Error())
	}

	// start with an initial baseline
	b := pgruntime.ConnectionBehavior{
		Migrations: map[int64]string{
			42: `
				CREATE TABLE posts (
					id       BIGSERIAL  PRIMARY KEY,
					message  TEXT       NOT NULL
				);
				CREATE TABLE comments (
					id       BIGSERIAL  PRIMARY KEY,
					post_id  BIGINT     REFERENCES posts ON DELETE CASCADE,
					message  TEXT       NOT NULL
				);
			`,
		},
	}
	db, target := connector.ConnectForTest(t, b)

	// check that the schema is applied by inserting some basic records
	postID, err := selectOneValue[int64](ctx, db, `INSERT INTO posts (message) VALUES ($1) RETURNING id`, "Hello World!")
	if err != nil {
		t.Fatal(err.Error())
	}
	_, err = execQuery(ctx, db, `INSERT INTO comments (post_id, message) VALUES ($1, $2)`, []any{postID, "Hi there."})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = db.Close()
	if err != nil {
		t.Fatal(err.Error())
	}

	// apply another migration
	b.Migrations[50] = `
		UPDATE comments c SET message = c.message || ' in response to: ' || p.message FROM posts p WHERE p.id = c.post_id;
	`
	db, err = connector.Connect(ctx, target, b)
	if err != nil {
		t.Fatal(err.Error())
	}

	// check that the data was modified appropriately
	message, err := selectOneValue[string](ctx, db, `SELECT message FROM comments`)
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, message, "Hi there. in response to: Hello World!")
	}

	// try to apply a broken migration
	b.Migrations[51] = `
		TRUNCATE commands; -- should be "comments"
	`
	_, err = connector.Connect(ctx, target, b)
	assert.ErrEqual(t, err, `while migrating to schema version 51: could not execute schema migration: pq: relation "commands" does not exist (42P01)`)

	// check that the migration was not applied
	version, err := selectOneValue[int64](ctx, db, `SELECT version FROM schema_migrations`)
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, version, 50)
	}
	commentCount, err := selectOneValue[int64](ctx, db, `SELECT COUNT(*) FROM comments`)
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, commentCount, 1)
	}
}
