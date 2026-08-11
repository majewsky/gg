// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package microprom

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// Handler is an [http.Handler] rendering metrics in Prometheus exposition formats.
//
// If SortOutput is false:
//   - Metric families will be printed in undefined order.
//   - Metrics within the same family will be printed in the order in which they were added.
//   - This behavior is the default because it is more efficient.
//
// If SortOutput is true:
//   - Metric families will be sorted by name.
//   - Metrics within the same family will be sorted by Labels.
//   - This behavior may be useful in tests because it produces deterministic output.
//
// When asserting on metrics in tests, it may be useful to set SortOutput equal to testing.Testing().
type Handler struct {
	// The set of metric families for which this handler can report metrics.
	Families map[MetricFamilyName]MetricFamilyInfo
	// This function will be called for each request to the handler.
	// The implementation shall provide metrics by calling [MetricSet.Add].
	Collect func(context.Context, *MetricSet) error

	// See documentation on type for details.
	SortOutput bool
}

var _ http.Handler = Handler{}

// ServeHTTP implements the [http.Handler] interface.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ms := NewMetricSet(SyntaxOpenMetricsV1, h.Families)
	err := h.Collect(r.Context(), ms)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: add support for `Content-Type: application/openmetrics-text; version=1.0.0; charset=utf-8` if requested in `Accept` header
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8; escaping=underscores")
	w.WriteHeader(http.StatusOK)

	bw := bufio.NewWriter(w)
	if h.SortOutput {
		for _, familyName := range slices.Sorted(maps.Keys(h.Families)) {
			h.printMetricFamily(bw, familyName, h.Families[familyName], ms.metrics[familyName])
		}
	} else {
		for familyName, familyInfo := range h.Families {
			h.printMetricFamily(bw, familyName, familyInfo, ms.metrics[familyName])
		}
	}

	fmt.Fprint(bw, "# EOF\n")
	err = bw.Flush()
	if err != nil {
		// We do not have a way to log this because we do not know what log library the application uses,
		// and I also do not want to add a dependency injection slot to type Handler for this one extremely unlikely codepath.
		// So instead, we're just going to wreck the response body and hope that Prometheus
		// or whatever else receives this logs this as a syntax error or something.
		fmt.Fprintf(w, "flush error: %s\n", err.Error())
	}
}

func (h Handler) printMetricFamily(w io.Writer, familyName MetricFamilyName, info MetricFamilyInfo, metrics []metric) {
	if len(metrics) == 0 {
		return
	}

	var metricName string
	switch info.Type {
	case MetricTypeGauge:
		metricName = string(familyName)
	case MetricTypeCounter:
		metricName = string(familyName) + "_total"
	case MetricTypeInfo:
		metricName = string(familyName) + "_info"
	default:
		panic("unreachable") // NewMetricSet() should have rejected unknown MetricType values
	}

	fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n", familyName, info.Help, familyName, metricTypeNames[info.Type])

	if h.SortOutput {
		slices.SortFunc(metrics, func(lhs, rhs metric) int {
			return strings.Compare(string(lhs.labels), string(rhs.labels))
		})
	}
	for _, m := range metrics {
		if m.labels == "" {
			fmt.Fprintf(w, "%s %g\n", metricName, m.value)
		} else {
			fmt.Fprintf(w, "%s{%s} %g\n", metricName, m.labels, m.value)
		}
	}
}
