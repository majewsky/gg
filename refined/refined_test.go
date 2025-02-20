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

func NewAccountName(value string) (AccountName, error) {
	v, err := refined.NewValue[AccountName](value)
	return AccountName{v}, err
}

// MatchesValue implements the refined.Condition interface.
func (AccountName) MatchesValue(value string) error {
	return refined.RegexpMatch(accountNameRx, value)
}

// Example for how to access the contained value in computations.
func (n AccountName) ContainerName() string {
	return fmt.Sprintf("container-for-%s", n.Get())
}

func TestAccountName(t *testing.T) {
	buf1 := []byte(`{"Name":"foo"}`)
	var d1 AccountData
	err := json.Unmarshal(buf1, &d1)
	AssertEqual(t, err, error(nil))
	AssertEqual(t, d1.Name.Get(), "foo")

	// TODO: fails because we need specialized unmarshaling logic on type AccountData
	buf2 := []byte(`{}`)
	var d2 AccountData
	err = json.Unmarshal(buf2, &d2)
	AssertEqual(t, err.Error(), "foo")
}
