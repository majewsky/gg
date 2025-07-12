// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined_test

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

////////////////////////////////////////////////////////////////////////////////
// example refinement types

type AccountName struct {
	refined.Scalar[AccountName, string]
}

var accountNameRx = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (AccountName) RefinedMatch(value string) bool {
	return accountNameRx.MatchString(value)
}
func (AccountName) RefinedBuild(s refined.PreScalar[AccountName, string]) AccountName {
	return AccountName{refined.PromoteScalar(s)}
}

type DiceRoll struct {
	refined.Scalar[DiceRoll, uint8]
}

func (DiceRoll) RefinedMatch(value uint8) bool {
	return value >= 1 && value <= 6
}
func (DiceRoll) RefinedBuild(s refined.PreScalar[DiceRoll, uint8]) DiceRoll {
	return DiceRoll{refined.PromoteScalar(s)}
}

// We need this for a test that specifically uses complex numbers.
type PurelyImaginary struct {
	refined.Scalar[PurelyImaginary, complex128]
}

func (PurelyImaginary) RefinedMatch(value complex128) bool {
	return real(value) == 0
}
func (PurelyImaginary) RefinedBuild(s refined.PreScalar[PurelyImaginary, complex128]) PurelyImaginary {
	return PurelyImaginary{refined.PromoteScalar(s)}
}

// We need this for a test involving serialization of zero values to JSON.
type EmptyString struct {
	refined.Scalar[EmptyString, string]
}

func (EmptyString) RefinedMatch(value string) bool {
	return value == ""
}
func (EmptyString) RefinedBuild(s refined.PreScalar[EmptyString, string]) EmptyString {
	return EmptyString{refined.PromoteScalar(s)}
}

////////////////////////////////////////////////////////////////////////////////
// static assertions that refined.Scalar implements the intended interfaces
// (The YAML interfaces are not checked because we don't want to add 3rd-party lib deps here.)

var (
	_ fmt.Formatter    = AccountName{}
	_ driver.Valuer    = AccountName{}
	_ sql.Scanner      = &AccountName{}
	_ json.Marshaler   = AccountName{}
	_ json.Unmarshaler = &AccountName{}
)

////////////////////////////////////////////////////////////////////////////////
// basic API

func TestNewOnScalar(t *testing.T) {
	value, err := refined.New[AccountName]("")
	AssertErrorEqual(t, err, `value "" is not acceptable for refined_test.AccountName`)
	AssertEqual(t, value.IsValid(), false)

	value, err = refined.New[AccountName]("a+b")
	AssertErrorEqual(t, err, `value "a+b" is not acceptable for refined_test.AccountName`)
	AssertEqual(t, value.IsValid(), false)

	value, err = refined.New[AccountName]("ab")
	AssertEqual(t, err, nil)
	AssertEqual(t, value.IsValid(), true)
	AssertEqual(t, value.Unpack(), "ab")
}

func TestLiteralOnScalar(t *testing.T) {
	message := AssertPanics(t, func() { _ = refined.Literal[AccountName]("") })
	AssertEqual(t, message, `value "" is not acceptable for refined_test.AccountName`)

	message = AssertPanics(t, func() { _ = refined.Literal[AccountName]("a+b") })
	AssertEqual(t, message, `value "a+b" is not acceptable for refined_test.AccountName`)

	value := refined.Literal[AccountName]("ab")
	AssertEqual(t, value.IsValid(), true)
	AssertEqual(t, value.Unpack(), "ab")
}

func TestIllegalScalarOperations(t *testing.T) {
	var emptyScalar AccountName
	AssertEqual(t, emptyScalar.IsValid(), false)
	AssertPanics(t, func() { emptyScalar.Unpack() })

	var emptyPreScalar refined.PreScalar[AccountName, string]
	AssertPanics(t, func() { _ = refined.PromoteScalar(emptyPreScalar) })
}

func TestComparisonOnScalar(t *testing.T) {
	n1 := refined.Literal[AccountName]("same")
	n2 := refined.Literal[AccountName]("different")
	n3 := refined.Literal[AccountName]("same")
	AssertEqual(t, n1 == n2, false)
	AssertEqual(t, n1 == n3, true)
}

////////////////////////////////////////////////////////////////////////////////
// interface implementations

func TestFormatterOnScalar(t *testing.T) {
	name := refined.Literal[AccountName]("abc")
	AssertEqual(t, fmt.Sprintf("%s", name), `abc`)
	AssertEqual(t, fmt.Sprintf("%q", name), `"abc"`)
	AssertEqual(t, fmt.Sprintf("%v", name), `refined_test.AccountName[abc]`)
	AssertEqual(t, fmt.Sprintf("%#v", name), `refined.Literal[refined_test.AccountName]("abc")`)

	roll := refined.Literal[DiceRoll](4)
	AssertEqual(t, fmt.Sprintf("%d", roll), `4`)
	AssertEqual(t, fmt.Sprintf("%02d", roll), `04`)
	AssertEqual(t, fmt.Sprintf("%v", roll), `refined_test.DiceRoll[4]`)
	AssertEqual(t, fmt.Sprintf("%#v", roll), `refined.Literal[refined_test.DiceRoll](0x4)`)
}

func TestMarshalSQLOnScalar(t *testing.T) {
	value, err := refined.Literal[AccountName]("hello").Value()
	AssertEqual(t, err, nil)
	AssertEqual(t, value, "hello")

	// complex64 and complex128 are the only types matching ScalarValue that can fail conversion into an SQL value
	_, err = refined.Literal[PurelyImaginary](5i).Value()
	AssertErrorEqual(t, err, `unsupported type complex128, a complex128`)
}

func TestUnmarshalSQLOnScalar(t *testing.T) {
	var n1 AccountName
	err := n1.Scan("example")
	AssertEqual(t, err, nil)
	AssertEqual(t, n1.Unpack(), "example")

	var n2 AccountName
	err = n2.Scan("a+b")
	AssertErrorEqual(t, err, `value "a+b" is not acceptable for refined_test.AccountName`)
	AssertEqual(t, n2.IsValid(), false)

	var n3 AccountName
	err = n3.Scan(nil)
	AssertErrorEqual(t, err, `unsupported Scan, storing driver.Value type <nil> into type refined_test.AccountName`)
	AssertEqual(t, n3.IsValid(), false)
}

func TestMarshalJSONOnScalar(t *testing.T) {
	data := struct {
		// test for serializing non-zero values
		N AccountName `json:"n"`
		// tests for serializing zero values
		E1 EmptyString `json:"e1"`
		E2 EmptyString `json:"e2,omitempty"`
		E3 EmptyString `json:"e3,omitzero"`
	}{
		N:  refined.Literal[AccountName]("foo"),
		E1: refined.Literal[EmptyString](""),
		E2: refined.Literal[EmptyString](""),
		E3: refined.Literal[EmptyString](""),
	}
	buf, err := json.Marshal(data)
	AssertEqual(t, err, nil)
	AssertEqual(t, string(buf), `{"n":"foo","e1":"","e2":""}`)
}

func TestUnmarshalJSONOnScalar(t *testing.T) {
	type payload struct {
		Name AccountName `json:"n"`
	}

	var p1 payload
	err := json.Unmarshal([]byte(`{"n":"foo"}`), &p1)
	AssertEqual(t, err, nil)
	AssertEqual(t, p1.Name.Unpack(), "foo")

	var p2 payload
	err = json.Unmarshal([]byte(`{"n":"a+b"}`), &p2)
	AssertErrorEqual(t, err, `value "a+b" is not acceptable for refined_test.AccountName`)

	// This is the problematic case. If "n" is not mentioned in the JSON payload,
	// UnmarshalJSON will never be called.
	var p3 payload
	err = json.Unmarshal([]byte(`{}`), &p3)
	AssertEqual(t, err, nil)
	AssertEqual(t, p3.Name.IsValid(), false)

	// TODO: demonstrate basic use of ValidateUnmarshaled() here
}
