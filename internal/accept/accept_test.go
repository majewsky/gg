// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package accept_test

import (
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/internal/accept"
	. "go.xyrillian.de/gg/option"
)

func TestAcceptWithHeader(t *testing.T) {
	// asking for a wide range of formats, including wildcard matches
	h := accept.ParseHeader([]string{"text/*;q=0.3, text/plain;format=flowed, text/plain;format=fixed;q=0.4, */*;q=0.5"})

	assert.Equal(t, h.Negotiate(
		"image/png",                // matches with q=0.5
		"text/plain; format=fixed", // matches with q=0.4
	), Some("image/png"))

	assert.Equal(t, h.Negotiate(
		"image/png",                 // matches with q=0.5
		"text/plain; format=flowed", // matches with q=1.0
	), Some("text/plain; format=flowed"))

	assert.Equal(t, h.Negotiate(
		"text/plain",                // matches with q=0.7
		"text/plain; format=flowed", // matches with q=1.0
	), Some("text/plain; format=flowed"))

	assert.Equal(t, h.Negotiate(
		"text/plain",               // matches with q=0.7
		"text/plain; format=other", // matches with q=0.3
	), Some("text/plain"))

	assert.Equal(t, h.Negotiate(
		"text/markdown", // matches with q=0.3
		"text/plain",    // matches with q=0.3 (but first wins)
	), Some("text/markdown"))

	// asking for specific formats only
	h = accept.ParseHeader([]string{"image/png, image/jpeg"})

	assert.Equal(t, h.Negotiate(
		"text/plain",
		"image/png",
	), Some("image/png"))

	assert.Equal(t, h.Negotiate(
		"text/plain",
	), None[string]())
}

func TestAcceptWithoutHeader(t *testing.T) {
	// Negotiate() will always pick the first option
	h := accept.ParseHeader(nil)

	assert.Equal(t, h.Negotiate(
		"image/png",
		"image/jpeg",
	), Some("image/png"))

	assert.Equal(t, h.Negotiate(nil...), None[string]())

	// malformed media types are ignored
	assert.Equal(t, h.Negotiate(
		"image/png/foo",
		"image/jpeg",
	), Some("image/jpeg"))

	assert.Equal(t, h.Negotiate(
		"image/png/foo",
		"image/jpeg/foo",
	), None[string]())
}

func TestAcceptWithMalformedHeader(t *testing.T) {
	for _, brokenHeader := range []string{
		"text/plain, text/markdown/foo",  // malformed media type
		"text/plain, image/png; q=high",  // malformed q-value
		"text/plain, image/jpeg; q=1.25", // q-value out of range
	} {
		h := accept.ParseHeader([]string{brokenHeader})

		// broken headers are ignored completely, so the first option wins by default
		assert.Equal(t, h.Negotiate(
			"image/png",
			"image/jpeg",
			"text/plain",
		), Some("image/png"))
	}
}
