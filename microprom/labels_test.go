// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package microprom_test

import (
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/microprom"
)

func TestFormatLabels(t *testing.T) {
	ms := microprom.NewMetricSet(microprom.SyntaxOpenMetricsV1, nil)

	// the basic example from the documentation
	names := microprom.NewLabelNames("foo", "hello")
	labels := ms.FormatLabels(names, "bar", "world")
	assert.Equal(t, labels, `foo="bar",hello="world"`)

	// test escaping label values
	labels = ms.FormatLabels(names, "bar\\\n\\bar", `"universe\world"`)
	assert.Equal(t, labels, `foo="bar\\\n\\bar",hello="\"universe\\world\""`)
}
