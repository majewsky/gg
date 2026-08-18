// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assert_test

import (
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/testcapture"
)

func TestPanics(t *testing.T) {
	//only testing error cases here, coverage of the happy path is provided by
	//usage of assert.Panics() in actual tests of other packages
	tc := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		assert.Panics(t, func() {
			// no panic
		})
	})
	assert.Equal(t, tc.Outcome, testcapture.OutcomeFailed)
	assert.Equal(t, tc.Messages, []testcapture.Message{
		testcapture.Log("did not panic"),
	})

	tc = testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) {
		assert.PanicsWith[string](t, func() {
			panic(42)
		})
	})
	assert.Equal(t, tc.Outcome, testcapture.OutcomeFailed)
	assert.Equal(t, tc.Messages, []testcapture.Message{
		testcapture.Log("panicked with incorrect type: expected string, but got int: 42"),
	})
}
