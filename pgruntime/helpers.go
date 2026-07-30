// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime

import (
	"context"
	"database/sql"
	"strings"

	"go.xyrillian.de/gg/errext"
	"go.xyrillian.de/gg/gsql"
)

// Convenience function for executing a one-off SQL query returning no rows.
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

// Convenience function for preparing an identifier that needs to be inserted into a query verbatim
// (e.g. a database name for CREATE DATABASE).
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
