// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package microprom_test

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/microprom"
)

func TestHandlerBasic(t *testing.T) {
	// NOTE: Most happy path coverage is in `./testing/microprom`.
	//       This only covers the SortOutput = false case.

	h := microprom.Handler{
		Families: map[microprom.MetricFamilyName]microprom.MetricFamilyInfo{
			"process": {
				Type: microprom.MetricTypeInfo,
				Help: "Information about this process.",
			},
			"foo": {
				Type: microprom.MetricTypeGauge,
				Help: "This metric family will not have any collected metrics and thus go unreported.",
			},
		},
		Collect: func(ctx context.Context, ms *microprom.MetricSet) error {
			names := microprom.NewLabelNames("version")
			labels := ms.FormatLabels(names, "1.2.3")
			ms.Add("process", labels, 1.0)
			return nil
		},
	}

	// test normal behavior
	status, body, headers := getMetrics(t, h, nil)
	assert.Equal(t, status, http.StatusOK)
	assert.Equal(t, headers, http.Header{
		"Content-Type": {"text/plain; version=0.0.4; charset=utf-8; escaping=underscores"},
	})
	assert.Equal(t, body, strings.TrimSpace(`
# HELP process_info Information about this process.
# TYPE process_info info
process_info{version="1.2.3"} 1
	`)+"\n")
}

func TestHandlerErrors(t *testing.T) {
	h := microprom.Handler{
		Families: map[microprom.MetricFamilyName]microprom.MetricFamilyInfo{
			"process": {
				Type: microprom.MetricTypeInfo,
				Help: "Information about this process.",
			},
		},
		Collect: func(ctx context.Context, ms *microprom.MetricSet) error {
			return errors.New("kaboom")
		},
	}

	// test unacceptable content negotiation
	status, body, headers := getMetrics(t, h, http.Header{"Accept": {"application/json"}})
	assert.Equal(t, status, http.StatusNotAcceptable)
	assert.Equal(t, headers.Get("Content-Type"), "text/plain; charset=utf-8")
	assert.Equal(t, body, "supported formats are text/plain and application/openmetrics-text\n")

	// test error during h.Collect()
	status, body, headers = getMetrics(t, h, nil)
	assert.Equal(t, status, http.StatusInternalServerError)
	assert.Equal(t, headers.Get("Content-Type"), "text/plain; charset=utf-8")
	assert.Equal(t, body, "kaboom\n")

	// test panic from invalid metric family name
	h.Families["what is this?"] = microprom.MetricFamilyInfo{
		Type: microprom.MetricTypeGauge,
		Help: "invalid metric family name",
	}
	msg := assert.PanicsWith[string](t, func() { getMetrics(t, h, nil) })
	assert.Equal(t, msg, `in family "what is this?": invalid family name (does not match /^[a-zA-Z_:][a-zA-Z0-9_:]*$/)`)
	delete(h.Families, "what is this?")

	// test panic from invalid metric type
	h.Families["invalid"] = microprom.MetricFamilyInfo{
		Type: 100,
		Help: "invalid metric type",
	}
	msg = assert.PanicsWith[string](t, func() { getMetrics(t, h, nil) })
	assert.Equal(t, msg, `in family "invalid": invalid value for microprom.MetricType: 100`)
	delete(h.Families, "invalid")

	// test panic from invalid label name
	h.Collect = func(ctx context.Context, ms *microprom.MetricSet) error {
		names := microprom.NewLabelNames("app:version")
		labels := ms.FormatLabels(names, "1.2.3")
		ms.Add("process", labels, 1.0)
		return nil
	}
	msg = assert.PanicsWith[string](t, func() { getMetrics(t, h, nil) })
	assert.Equal(t, msg, `invalid label name: "app:version"`)

	// test panic from wrong number of label values
	h.Collect = func(ctx context.Context, ms *microprom.MetricSet) error {
		names := microprom.NewLabelNames("version", "build_date")
		labels := ms.FormatLabels(names, "1.2.3") // forgot build_date
		ms.Add("process", labels, 1.0)
		return nil
	}
	msg = assert.PanicsWith[string](t, func() { getMetrics(t, h, nil) })
	assert.Equal(t, msg, `expected 2 label values, but got 1`)

	// test panic from using an undeclared metric family
	h.Collect = func(ctx context.Context, ms *microprom.MetricSet) error {
		ms.Add("invalid", "", 1.0)
		return nil
	}
	msg = assert.PanicsWith[string](t, func() { getMetrics(t, h, nil) })
	assert.Equal(t, msg, `no such family: invalid`)
}

func getMetrics(t *testing.T, h http.Handler, requestHeaders http.Header) (status int, responseBody string, responseHeaders http.Header) {
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
	return resp.StatusCode, string(buf), resp.Header
}
