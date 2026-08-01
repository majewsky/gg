// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pgruntime_test

import (
	"net/url"
	"strings"
	"testing"

	"go.xyrillian.de/gg/assert"
	"go.xyrillian.de/gg/pgruntime"
)

func TestParseConnectionTargetSuccess(t *testing.T) {
	testCases := map[string]pgruntime.ConnectionTarget{
		// minimal case: just the required fields are set
		`postgresql://alice@localhost/bookstore`: {
			HostName:     "localhost",
			UserName:     "alice",
			DatabaseName: "bookstore",
		},
		// maximal case: all required fields are set
		`postgres://alice:swordfish@db.example.com:5432/bookstore?sslmode=prefer&application_name=frontdesk`: {
			HostName:          "db.example.com",
			Port:              "5432",
			UserName:          "alice",
			Password:          "swordfish",
			DatabaseName:      "bookstore",
			ConnectionOptions: "sslmode=prefer&application_name=frontdesk",
		},
	}

	for input, expected := range testCases {
		t.Run(input, func(t *testing.T) {
			u, err := url.Parse(input)
			if !assert.ErrEqual(t, err, nil) {
				t.FailNow()
			}
			parsed, err := pgruntime.ParseConnectionTargetFromURL(u)
			if !assert.ErrEqual(t, err, nil) {
				t.FailNow()
			}
			assert.Equal(t, parsed, expected)
			serialized, err := parsed.IntoURL()
			if !assert.ErrEqual(t, err, nil) {
				t.FailNow()
			}
			// IntoURL() always uses postgres://, so this is not always an exact roundtrip
			assert.Equal(t, serialized.String(), strings.ReplaceAll(input, "postgresql:", "postgres:"))
		})
	}
}

func TestParseConnectionTargetFailure(t *testing.T) {
	testCases := map[string]string{
		`https://alice@db.example.com/foo?sslmode=prefer`:            `expected scheme "postgres" or "postgresql", but got "https"`,
		`postgres:foo?sslmode=prefer`:                                `expected authority component, but got rootless path "foo"`,
		`postgres://alice@db.example.com/foo#sslmode=prefer`:         `unexpected fragment "sslmode=prefer"`,
		`postgres://alice@db.example.com::5432/foo?sslmode=prefer`:   `malformed host "db.example.com::5432"`,
		`postgres://alice@db.example.com/foo/bar?sslmode=prefer`:     `malformed database name "foo/bar"`,
		`postgres://alice@db.example.com/foo?sslmode=prefer;foo=bar`: `malformed connection options: invalid semicolon separator in query`,
	}
	for input, expected := range testCases {
		t.Run(input, func(t *testing.T) {
			u, err := url.Parse(input)
			if !assert.ErrEqual(t, err, nil) {
				t.FailNow()
			}
			_, err = pgruntime.ParseConnectionTargetFromURL(u)
			assert.ErrEqual(t, err, "in ParseConnectionTargetFromURL: "+expected)
		})
	}
}

func TestSerializeConnectionOptions(t *testing.T) {
	ct := pgruntime.ConnectionTarget{
		HostName:     "localhost",
		UserName:     "alice",
		DatabaseName: "bookstore",
	}

	// cannot merge ConnectionOptions if the string part is malformed
	ct.ConnectionOptions = `sslmode=prefer;foo=bar`
	ct.ExtraConnectionOptions = url.Values{"application_name": {"frontdesk"}}
	_, err := ct.IntoURL()
	assert.ErrEqual(t, err, `in ConnectionTarget.IntoURL: malformed connection options: invalid semicolon separator in query`)

	// successful trivial merges
	ct.ConnectionOptions = `sslmode=prefer`
	ct.ExtraConnectionOptions = nil
	u, err := ct.IntoURL()
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, u.RawQuery, `sslmode=prefer`)
	}

	ct.ConnectionOptions = ""
	ct.ExtraConnectionOptions = url.Values{"application_name": {"frontdesk"}}
	u, err = ct.IntoURL()
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, u.RawQuery, `application_name=frontdesk`)
	}

	// successful complex merge
	ct.ConnectionOptions = `sslmode=prefer`
	ct.ExtraConnectionOptions = url.Values{"application_name": {"frontdesk"}}
	u, err = ct.IntoURL()
	if assert.ErrEqual(t, err, nil) {
		assert.Equal(t, u.RawQuery, `application_name=frontdesk&sslmode=prefer`)
	}
}
