/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* refined.Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined_test

import (
	"encoding/json"
	"regexp"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

var accountNameRx = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// Full demonstration of a refinement type for the test.
type AccountName = refined.Value[string, accountNameCondition]

type accountNameCondition struct{}

func (accountNameCondition) MatchesValue(value string) error {
	return refined.RegexpMatch(accountNameRx, value)
}

// Demonstration of a struct containing a refinement type.
type AccountData struct {
	Name AccountName
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
		// TODO: yuck
		foo = AccountName(refined.LiteralValue[string, accountNameCondition]("foo"))
		bar = AccountName(refined.LiteralValue[string, accountNameCondition]("bar"))
	)
	m := map[AccountName]int{
		foo: 3,
		bar: 1,
	}
	// TODO: AccountName is not an ordered type; we might need stuff like slices.Sorted() in package refined
	// AssertEqual(slices.Sorted(maps.Keys(m)), []AccountName{bar, foo})
	_ = m
}
