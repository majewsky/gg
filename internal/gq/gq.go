// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package gq contains helpers for gsql ("gsql query" -> gq).
// TODO: most of these functions should be either in gsql as public API, or in Oblast
package gq

import (
	"context"
	"database/sql"

	"go.xyrillian.de/gg/errext"
	"go.xyrillian.de/gg/gsql"
)

// ExecQuery is a convenience function for executing a one-off SQL query returning no rows.
func ExecQuery(ctx context.Context, db gsql.Handle, query string, args []any) (sql.Result, error) {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		return nil, err
	}
	result, err := stmt.Exec(ctx, args)
	return result, errext.WithCleanup(err, "stmt.Close", stmt.Close())
}

// QueryRow is a convenience function for executing a one-off SQL query returning one row.
func QueryRow(ctx context.Context, db gsql.Handle, query string, args, slots []any) error {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		return err
	}
	err = stmt.QueryRow(ctx, args, slots)
	return errext.WithCleanup(err, "stmt.Close", stmt.Close())
}

// SelectOneValue is a convenience function for executing a one-off SQL query returning one value.
func SelectOneValue[T any](ctx context.Context, db gsql.Handle, query string, args ...any) (T, error) {
	stmt, err := db.GSQLPrepare(ctx, query, false)
	if err != nil {
		var none T
		return none, err
	}

	var result T
	err = stmt.QueryRow(ctx, args, []any{&result})
	return result, errext.WithCleanup(err, "stmt.Close", stmt.Close())
}

// SelectSeveralValues is a convenience function for executing a one-off SQL query returning several single-column rows.
func SelectSeveralValues[T any](ctx context.Context, db gsql.Handle, query string, args ...any) ([]T, error) {
	rows, err := db.GSQLQuery(ctx, query, args)
	if err != nil {
		return nil, err
	}
	var result []T
	for rows.Next() {
		// TODO: this should share growRecordSlice() from Oblast to optimize allocations
		var value T
		err := rows.Scan(&value)
		if err != nil {
			return nil, errext.WithCleanup(err, "rows.Close", rows.Close())
		}
		result = append(result, value)
	}
	return result, errext.WithCleanup(nil, "rows.Err", rows.Err())
}
