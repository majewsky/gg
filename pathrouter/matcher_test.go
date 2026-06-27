// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pathrouter // This file contains tests that access unexported helper functions.

import (
	"net/url"
	"testing"

	"go.xyrillian.de/gg/assert"
)

func TestExtractPath(t *testing.T) {
	extract := func(rawURL string) []string {
		t.Helper()
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		return extractPath(parsedURL)
	}

	assert.Equal(t, extract("http://localhost/"), []string{""})
	assert.Equal(t, extract("http://localhost///"), []string{""})
	assert.Equal(t, extract("http://localhost/foo"), []string{"foo"})
	assert.Equal(t, extract("http://localhost///foo"), []string{"foo"})
	assert.Equal(t, extract("http://localhost/foo/"), []string{"foo", ""})
	assert.Equal(t, extract("http://localhost///foo///"), []string{"foo", ""})
	assert.Equal(t, extract("http://localhost/foo/%2Fbar%2F"), []string{"foo", "%2Fbar%2F"})
	assert.Equal(t, extract("http://localhost///foo///%2Fbar%2F"), []string{"foo", "%2Fbar%2F"})
	assert.Equal(t, extract("http://localhost/foo/%2Fbar%2F/"), []string{"foo", "%2Fbar%2F", ""})
	assert.Equal(t, extract("http://localhost///foo///%2Fbar%2F///"), []string{"foo", "%2Fbar%2F", ""})
}
