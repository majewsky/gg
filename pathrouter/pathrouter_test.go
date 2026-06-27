// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pathrouter_test

import (
	"cmp"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	pr "go.xyrillian.de/gg/pathrouter"
	"go.xyrillian.de/gg/testcapture"
)

func TestRouting(t *testing.T) {
	// helper methods
	varRx := regexp.MustCompile(`\$\w+`)
	h := func(status int, msgBase string) func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
		return func(w http.ResponseWriter, r *http.Request, vars map[string]string) {
			msg := varRx.ReplaceAllStringFunc(msgBase, func(match string) string {
				return vars[strings.TrimPrefix(match, "$")]
			})
			if r.Method == http.MethodHead {
				w.WriteHeader(status)
			} else {
				http.Error(w, msg, status)
			}
		}
	}
	var m pr.Matcher
	check := func(method, path, expected string, expectedHeaders map[string]string) {
		t.Helper()

		req := httptest.NewRequest(method, "http://localhost"+path, http.NoBody)
		rec := httptest.NewRecorder()
		m.ServeHTTP(rec, req)

		body := strings.TrimSuffix(rec.Body.String(), "\n")
		actual := fmt.Sprintf("%d: %s", rec.Code, cmp.Or(body, "<none>"))
		assert.Equal(t, actual, expected)

		actualHeaders := make(map[string]string)
		for key, value := range rec.Header() {
			actualHeaders[key] = value[0]
		}
		if actualHeaders["Content-Type"] == "text/plain; charset=utf-8" {
			delete(actualHeaders, "Content-Type")
		}
		if actualHeaders["X-Content-Type-Options"] == "nosniff" {
			delete(actualHeaders, "X-Content-Type-Options")
		}
		if len(actualHeaders) == 0 {
			actualHeaders = nil
		}
		assert.Equal(t, actualHeaders, expectedHeaders)
	}

	// test router matching only "/" path
	m = pr.Element("/", pr.Handlers(pr.ByMethod{
		http.MethodGet: h(http.StatusOK, "root"),
	}))
	check("GET", "/", "200: root", nil)
	check("PUT", "/", "405: Method Not Allowed", map[string]string{"Allow": "GET, HEAD"})
	check("OPTIONS", "/", "200: <none>", map[string]string{"Allow": "GET, HEAD"})
	check("GET", "///", "200: root", nil)
	check("PUT", "///", "405: Method Not Allowed", map[string]string{"Allow": "GET, HEAD"})
	check("OPTIONS", "///", "200: <none>", map[string]string{"Allow": "GET, HEAD"})
	check("GET", "/foo", "404: 404 page not found", nil)
	check("PUT", "/foo", "404: 404 page not found", nil)
	check("OPTIONS", "/foo", "404: 404 page not found", nil)

	// test alternative representation for router matching only "/" path
	m = pr.Here(pr.Handlers(pr.ByMethod{
		http.MethodGet: h(http.StatusOK, "hello"),
	}))
	check("GET", "/", "200: hello", nil)
	check("GET", "/foo", "404: 404 page not found", nil)

	// check Element()
	m = pr.Element("foo", pr.Element("bar", pr.Handlers(pr.ByMethod{
		http.MethodGet:    h(http.StatusOK, "that's me"),
		http.MethodPut:    h(http.StatusCreated, "here I am"),
		http.MethodDelete: h(http.StatusAccepted, "goodbye"),
	})))
	check("GET", "/", "404: 404 page not found", nil)
	check("GET", "/foo/", "404: 404 page not found", nil)
	check("GET", "/foo", "404: 404 page not found", nil)
	check("GET", "/bar/", "404: 404 page not found", nil)
	check("GET", "/bar", "404: 404 page not found", nil)
	check("GET", "/foo/bar/", "404: 404 page not found", nil)
	check("GET", "/foo/bar", "200: that's me", nil)
	check("PUT", "/foo/bar", "201: here I am", nil)
	check("DELETE", "/foo/bar", "202: goodbye", nil)

	// check Choice(), different behavior for MethodHead in Handlers()
	m = pr.Choice(
		pr.Element("implied", pr.Handlers(pr.ByMethod{
			http.MethodGet: h(http.StatusOK, "implied"),
		})),
		pr.Element("separate", pr.Handlers(pr.ByMethod{
			http.MethodGet:  h(http.StatusOK, "separate"),
			http.MethodHead: h(http.StatusNoContent, ""),
		})),
		pr.Element("deleted", pr.Handlers(pr.ByMethod{
			http.MethodGet:  h(http.StatusOK, "deleted"),
			http.MethodHead: nil,
		})),
	)
	check("GET", "/implied", "200: implied", nil)
	check("HEAD", "/implied", "200: <none>", nil)
	check("GET", "/separate", "200: separate", nil)
	check("HEAD", "/separate", "204: <none>", nil)
	check("GET", "/deleted", "200: deleted", nil)
	check("HEAD", "/deleted", "405: Method Not Allowed", map[string]string{"Allow": "GET"})

	// check CatchAllVariable(), esp. with a variable number of acceptable path length in the inner matcher
	m = pr.CatchAllVariable("path", pr.Choice(
		pr.Element("one", pr.Handlers(pr.ByMethod{
			http.MethodGet: h(http.StatusOK, "one $path"),
		})),
		pr.Element("two", pr.Element("two", pr.Handlers(pr.ByMethod{
			http.MethodGet: h(http.StatusOK, "two $path"),
		}))),
		pr.Element("three", pr.Element("three", pr.Element("three", pr.Handlers(pr.ByMethod{
			http.MethodGet: h(http.StatusOK, "three $path"),
		})))),
	))

	check("GET", "/foo/bar/one", "200: one foo/bar", nil)
	check("GET", "/foo/bar/two/two", "200: two foo/bar", nil)
	check("GET", "/foo/bar/three/three/three", "200: three foo/bar", nil)

	check("GET", "/foo%2Fbar/one", "200: one foo/bar", nil)
	check("GET", "/foo%2Fbar/two%2Ftwo", "404: 404 page not found", nil)

	check("GET", "/long/path/but/no/match", "404: 404 page not found", nil)
	check("GET", "/shortpathbutnomatch", "404: 404 page not found", nil)

	// check Variable()
	m = pr.Element("nice", pr.Element("objects", pr.Variable("id", pr.Here(pr.Handlers(pr.ByMethod{
		http.MethodPut: h(http.StatusCreated, "created object $id"),
	})))))

	check("PUT", "/nice/objects", "404: 404 page not found", nil)
	check("PUT", "/nice/objects/", "404: 404 page not found", nil)
	check("PUT", "/nice/objects/42", "201: created object 42", nil)
	check("PUT", "/nice/objects/4/2", "404: 404 page not found", nil) // variable can onlt catch one element
	check("PUT", "/nice/objects/4%2F2", "201: created object 4/2", nil)
	check("PUT", "/nice/objects/4%2F2/", "201: created object 4/2", nil)
}

func TestPanics(t *testing.T) {
	check := func(expected string, action func()) {
		t.Helper()
		result := testcapture.Capture(t.Context(), t.Name(), func(t assert.TestingTB) { action() })
		assert.Equal(t, result, testcapture.Result{
			Outcome: testcapture.OutcomePanicked,
			Panic:   expected,
		})
	}

	check(`matcher within CatchAllVariable() may not accept unlimited path lengths`, func() {
		pr.CatchAllVariable("first", pr.Choice(pr.CatchAllVariable("second", pr.Handlers(nil))))
	})

	check(`Choice() called without any matchers`, func() {
		pr.Element("", pr.Choice())
	})

	check(`Element() called with value = ""`, func() {
		pr.Element("", pr.Handlers(nil))
	})

	check(`matcher within Here() must be Handlers()`, func() {
		pr.Here(pr.Element("foo", pr.Handlers(nil)))
	})

	check(`handler is nil for method "POST"`, func() {
		pr.Handlers(pr.ByMethod{
			http.MethodPost: nil,
		})
	})
}
