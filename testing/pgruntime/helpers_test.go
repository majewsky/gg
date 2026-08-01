// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"context"
	"database/sql"

	"go.xyrillian.de/gg/errext"
	"go.xyrillian.de/gg/gsql"
)

// Convenience function for executing a one-off SQL query returning no rows.
// TODO: move to gsql
func execQuery(ctx context.Context, db gsql.Handle, query string, args []any) (sql.Result, error) {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		return nil, err
	}
	result, err := stmt.Exec(ctx, args)
	return result, errext.WithCleanup(err, "stmt.Close", stmt.Close())
}

// Convenience function for executing a one-off SQL query returning one row.
func queryRow(ctx context.Context, db gsql.Handle, query string, args, slots []any) error {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		return err
	}
	err = stmt.QueryRow(ctx, args, slots)
	return errext.WithCleanup(err, "stmt.Close", stmt.Close())
}

// Convenience function for executing a one-off SQL query returning one value.
// TODO: move to gsql
func selectOneValue[T any](ctx context.Context, db gsql.Handle, query string, args ...any) (T, error) {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		var none T
		return none, err
	}

	var result T
	err = stmt.QueryRow(ctx, args, []any{&result})
	return result, errext.WithCleanup(err, "stmt.Close", stmt.Close())
}
