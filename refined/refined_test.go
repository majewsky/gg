/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: GPL-3.0-only
* refined.Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

var accountNameRx = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Full demonstration of a refinement type for the test.
type AccountName struct {
	refined.Value[AccountName, string]
}

// Demonstration of a struct containing a refinement type.
type AccountData struct {
	Name AccountName
}

// MatchesValue implements the refined.Builder interface.
func (AccountName) MatchesValue(value string) error {
	return refined.RegexpMatch(accountNameRx, value)
}

// Build implements the refined.Builder interface.
func (AccountName) Build(v refined.Prevalue[AccountName, string]) AccountName {
	return AccountName{refined.Build(v)}
}

// Example for how to access the contained value in computations.
func (n AccountName) ContainerName() string {
	return fmt.Sprintf("container-for-%s", n.Raw())
}

func TestAccountName(t *testing.T) {
	buf1 := []byte(`{"Name":"foo"}`)
	var d1 AccountData
	err := json.Unmarshal(buf1, &d1)
	AssertEqual(t, err, error(nil))
	AssertEqual(t, d1.Name.Raw(), "foo")

	// TODO: fails because we need specialized unmarshaling logic on type AccountData
	buf2 := []byte(`{}`)
	var d2 AccountData
	err = json.Unmarshal(buf2, &d2)
	AssertEqual(t, err.Error(), "foo")
}

func TestRefinedMapKeys(t *testing.T) {
	var (
		foo = refined.Literal[AccountName]("foo")
		bar = refined.Literal[AccountName]("bar")
	)
	m := map[AccountName]int{
		foo: 3,
		bar: 1,
	}
	// TODO: AccountName is not an ordered type; we might need stuff like slices.Sorted() in package refined
	// AssertEqual(slices.Sorted(maps.Keys(m)), []AccountName{bar, foo})
	_ = m
}
