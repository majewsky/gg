// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgtest

import (
	"context"
	"fmt"
	"slices"

	"go.xyrillian.de/gg/errext"
	"go.xyrillian.de/gg/gsql"
)

type topology struct {
	TableInfoByName map[string]tableInfo
}

type tableInfo struct {
	Columns []columnInfo
}

type columnInfo struct {
	Name         string
	DefaultValue sqlLiteral
	IsPrimaryKey bool
}

const (
	topologyGetColumnsQuery = `
		SELECT table_name, column_name, column_default
		  FROM information_schema.columns
		 WHERE table_schema = 'public'
		 ORDER BY table_name, ordinal_position
	`
	topologyGetObviousPrimaryKeysQuery = `
		SELECT DISTINCT table_name, column_name
		  FROM information_schema.key_column_usage
		 WHERE table_schema = 'public' AND position_in_unique_constraint IS NULL AND constraint_name = table_name || '_pkey'
		 ORDER BY 1, 2
	`
	topologyGetPossiblePrimaryKeysQuery = `
		SELECT DISTINCT table_name, column_name
		  FROM information_schema.key_column_usage
		 WHERE table_schema = 'public' AND position_in_unique_constraint IS NULL
		 ORDER BY 1, 2
	`
)

func newTopology(ctx context.Context, db gsql.Handle) (topology, error) {
	result := topology{
		TableInfoByName: make(map[string]tableInfo),
	}

	// enumerate tables and their columns
	columnInfosByTableName, err := topologyGetColumnInfo(ctx, db)
	if err != nil {
		return topology{}, fmt.Errorf("while querying information_schema.columns: %w", err)
	}
	for tableName, columnInfos := range columnInfosByTableName {
		result.TableInfoByName[tableName] = tableInfo{Columns: columnInfos}
	}

	// find obvious primary keys (columns that are included in UNIQUE constraints with the name `${TABLE}_pkey`)
	obviousPKColumnsByTableName, err := topologyGetColumnsMatching(ctx, db, topologyGetObviousPrimaryKeysQuery)
	if err != nil {
		return topology{}, fmt.Errorf("while querying information_schema.key_column_usage for obvious primary keys: %w", err)
	}
	for tableName, columnNames := range obviousPKColumnsByTableName {
		for idx, col := range result.TableInfoByName[tableName].Columns {
			if slices.Contains(columnNames, col.Name) {
				col.IsPrimaryKey = true
				result.TableInfoByName[tableName].Columns[idx] = col
			}
		}
	}

	// as a fallback, find possible primary keys (columns that are included in any UNIQUE constraint)
	possiblePKColumnsByTableName, err := topologyGetColumnsMatching(ctx, db, topologyGetPossiblePrimaryKeysQuery)
	if err != nil {
		return topology{}, fmt.Errorf("while querying information_schema.key_column_usage for possible primary keys: %w", err)
	}
	for tableName, columnNames := range possiblePKColumnsByTableName {
		if len(obviousPKColumnsByTableName[tableName]) > 0 {
			continue
		}
		for idx, col := range result.TableInfoByName[tableName].Columns {
			if slices.Contains(columnNames, col.Name) {
				col.IsPrimaryKey = true
				result.TableInfoByName[tableName].Columns[idx] = col
			}
		}
	}

	return result, nil
}

func topologyGetColumnInfo(ctx context.Context, db gsql.Handle) (map[string][]columnInfo, error) {
	rows, err := db.GSQLQuery(ctx, topologyGetColumnsQuery, nil)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]columnInfo)
	for rows.Next() {
		var (
			tableName string
			col       columnInfo
		)
		err := rows.Scan(&tableName, &col.Name, &col.DefaultValue)
		if err != nil {
			return nil, errext.WithCleanup(err, "rows.Close", rows.Close())
		}
		result[tableName] = append(result[tableName], col)
	}
	return result, errext.WithCleanup(nil, "rows.Err", rows.Err())
}

func topologyGetColumnsMatching(ctx context.Context, db gsql.Handle, query string) (map[string][]string, error) {
	rows, err := db.GSQLQuery(ctx, query, nil)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for rows.Next() {
		var tableName, columnName string
		err := rows.Scan(&tableName, &columnName)
		if err != nil {
			return nil, errext.WithCleanup(err, "rows.Close", rows.Close())
		}
		result[tableName] = append(result[tableName], columnName)
	}
	return result, errext.WithCleanup(nil, "rows.Err", rows.Err())
}
