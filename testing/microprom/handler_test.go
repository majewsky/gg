// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package microprom_test

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/microprom"
)

func TestHandlerFunctionallyIdenticalToPromhttp(t *testing.T) {
	// build a microprom.Handler rendering two metric families
	h1 := microprom.Handler{
		Families: map[microprom.MetricFamilyName]microprom.MetricFamilyInfo{
			"events": {
				Type: microprom.MetricTypeCounter,
				Help: "Counts events that happened.",
			},
			"memory_usage_bytes": {
				Type: microprom.MetricTypeGauge,
				Help: "How much memory is currently used.",
			},
		},
		SortOutput: true,
		Collect: func(ctx context.Context, ms *microprom.MetricSet) error {
			labelNames := microprom.NewLabelNames("shard", "type")
			for idx := range 5 {
				labels := ms.FormatLabels(labelNames, fmt.Sprintf("node%d", idx), "update")
				ms.Add("events", labels, float64(10*idx))
			}
			ms.Add("memory_usage_bytes", "", 42<<20)
			return nil
		},
	}

	// build a promhttp.Handler rendering the same metric families
	eventsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "events_total",
		Help: "Counts events that happened.",
	}, []string{"type", "shard"})
	for idx := range 5 {
		eventsCounter.With(prometheus.Labels{
			"type":  "update",
			"shard": fmt.Sprintf("node%d", idx),
		}).Add(float64(10 * idx))
	}
	memoryUsageBytesGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_usage_bytes",
		Help: "How much memory is currently used.",
	})
	memoryUsageBytesGauge.Set(42 << 20)
	r := prometheus.NewRegistry()
	r.MustRegister(eventsCounter)
	r.MustRegister(memoryUsageBytesGauge)
	h2 := promhttp.HandlerFor(r, promhttp.HandlerOpts{EnableOpenMetrics: true})

	// test identical behavior for Prometheus Text Format
	body1, headers1 := getMetrics(t, h1, nil)
	body2, headers2 := getMetrics(t, h2, nil)
	assert.Equal(t, strings.Split(body1, "\n"), strings.Split(body2, "\n"))
	assert.Equal(t, headers1, headers2)

	// test identical behavior for OpenMetrics 1.0 text format
	body1, headers1 = getMetrics(t, h1, http.Header{"Accept": {"application/openmetrics-text; version=1.0.0"}})
	body2, headers2 = getMetrics(t, h2, http.Header{"Accept": {"application/openmetrics-text; version=1.0.0"}})
	assert.Equal(t, strings.Split(body1, "\n"), strings.Split(body2, "\n"))
	assert.Equal(t, headers1, headers2)

	// test invalid Accept header
	body1, headers1 = getMetrics(t, h1, http.Header{"Accept": {"image/*"}})
	body2, headers2 = getMetrics(t, h2, http.Header{"Accept": {"image/*"}})
	assert.Equal(t, strings.Split(body1, "\n"), strings.Split(body2, "\n"))
	assert.Equal(t, headers1, headers2)
}

func getMetrics(t *testing.T, h http.Handler, requestHeaders http.Header) (responseBody string, responseHeaders http.Header) {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil)
	maps.Copy(r.Header, requestHeaders)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	resp := w.Result()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err.Error())
	}
	return string(buf), resp.Header
}
