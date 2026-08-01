// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"testing"

	_ "github.com/lib/pq"
	"go.xyrillian.de/gg/pgruntime"
)

var (
	defaultBehavior = pgruntime.ConnectionBehavior{}
	connector       = pgruntime.StdConnector("postgres")
)

func TestMain(m *testing.M) {
	pgruntime.WithTestDB(m, m.Run)
}
