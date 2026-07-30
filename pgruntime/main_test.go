// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"testing"

	"go.xyrillian.de/gg/pgruntime"
)

func TestMain(m *testing.M) {
	pgruntime.WithTestDB(m, m.Run)
}
