// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package assetembed_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/majewsky/gg/assetembed"
	. "github.com/majewsky/gg/internal/test"
	. "github.com/majewsky/gg/option"
)

func TestAssetembed(t *testing.T) {
	const (
		fooTxt    = "hello world"
		fooSHA384 = "/b2OdaZ/KfcBpOBAOF4uI5hjA+oQI5IRr5B/y7g1eLPkF8txzmRu/QgZ3YwIjeG9"
		barJS     = "alert('hello')"
		barSHA384 = "nK45OZX/RRKGmhPEj7lSXYQ3NDNNqiBUgbymhjhJ/jpCg0eAyZQ5UuzakE/UFcBd"
	)
	h, err := assetembed.NewHandler(fstest.MapFS{
		"foo.txt": &fstest.MapFile{
			Data:    []byte(fooTxt),
			Mode:    0644,
			ModTime: time.Unix(1, 0),
		},
		"res/assets/bar.js": &fstest.MapFile{
			Data:    []byte(barJS),
			Mode:    0644,
			ModTime: time.Unix(2, 0),
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	// test Handler.AssetPath()
	urlSafe := func(base64str string) string {
		base64str = strings.ReplaceAll(base64str, "+", "-")
		base64str = strings.ReplaceAll(base64str, "/", "_")
		return base64str
	}
	fooTxtPath := fmt.Sprintf("foo-%s.txt", urlSafe(fooSHA384))
	AssertEqual(t, h.AssetPath("foo.txt"), Some(fooTxtPath))
	barJSPath := fmt.Sprintf("res/assets/bar-%s.js", urlSafe(barSHA384))
	AssertEqual(t, h.AssetPath("res/assets/bar.js"), Some(barJSPath))
	AssertEqual(t, h.AssetPath("res/assets/unknown.css"), None[string]())

	// test Handler.SubresourceIntegrity()
	AssertEqual(t, h.SubresourceIntegrity("foo.txt"), Some("sha384-"+fooSHA384))
	AssertEqual(t, h.SubresourceIntegrity("res/assets/bar.js"), Some("sha384-"+barSHA384))
	AssertEqual(t, h.SubresourceIntegrity("res/assets/unknown.css"), None[string]())

	// test HTTP handler
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		t.Logf("testing with method = %q", method)
		probe := func(path string) *httptest.ResponseRecorder {
			req := httptest.NewRequest(method, path, http.NoBody)
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			return resp
		}
		ifGet := func(payload string) string {
			if method == http.MethodGet {
				return payload
			} else {
				return ""
			}
		}

		resp := probe("/" + fooTxtPath)
		AssertEqual(t, resp.Code, http.StatusOK)
		AssertEqual(t, resp.Header().Get("Content-Type"), "text/plain; charset=utf-8")
		AssertEqual(t, resp.Header().Get("Content-Length"), strconv.Itoa(len(fooTxt)))
		AssertEqual(t, resp.Body.String(), ifGet(fooTxt))

		resp = probe("/" + barJSPath)
		AssertEqual(t, resp.Code, http.StatusOK)
		AssertEqual(t, resp.Header().Get("Content-Type"), "text/javascript; charset=utf-8")
		AssertEqual(t, resp.Header().Get("Content-Length"), strconv.Itoa(len(barJS)))
		AssertEqual(t, resp.Body.String(), ifGet(barJS))

		resp = probe("/foo.txt") // without digest!
		AssertEqual(t, resp.Code, http.StatusNotFound)

		resp = probe("/res/assets/unknown.css") // entirely unknown file
		AssertEqual(t, resp.Code, http.StatusNotFound)
	}

	req := httptest.NewRequest(http.MethodPut, "/"+fooTxtPath, http.NoBody)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	AssertEqual(t, resp.Code, http.StatusMethodNotAllowed)
}
