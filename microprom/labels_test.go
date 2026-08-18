// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package microprom_test

import (
	"fmt"
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

func BenchmarkFormatLabels(b *testing.B) {
	ms := microprom.NewMetricSet(microprom.SyntaxPrometheusLegacy, nil)
	const (
		appVersion = "1.2.3"
		node       = "fcae8f43-87d4-4644-b28c-7bb1c340fcea"
		region     = "us-east"
		status     = "404"
	)
	names := microprom.NewLabelNames("app_version", "node", "region", "status")

	b.Run("method=fmt.Sprintf", func(b *testing.B) {
		for b.Loop() {
			_ = fmt.Sprintf(`app_version=%q,node=%q,region=%q,status=%q`,
				appVersion, node, region, status,
			)
		}
	})
	b.Run("method=FormatLabels", func(b *testing.B) {
		for b.Loop() {
			_ = ms.FormatLabels(names, appVersion, node, region, status)
		}
	})
}
