// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert_test

import (
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/testcapture"
)

func expectErrors(t *testing.T, test func(assert.TestingTB), expected string) {
	t.Helper()
	r := testcapture.Result{Outcome: testcapture.OutcomeFailed}
	for line := range strings.SplitSeq(expected, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			r.Messages = append(r.Messages, testcapture.Log(line))
		}
	}
	assert.Equal(t, testcapture.Capture(t.Context(), t.Name(), test), r)
}

func TestEqual(t *testing.T) {
	assert.Equal(t, true, true)
	expectErrors(t, func(t assert.TestingTB) {
		assert.Equal(t, false, true)
	}, `expected true, but got false`)
}
