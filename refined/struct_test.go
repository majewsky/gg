// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

type AccountID struct {
	refined.Scalar[AccountID, uint64]
}

func (AccountID) RefinedMatch(value uint64) bool { return value > 0 }
func (AccountID) RefinedBuild(p refined.PreScalar[AccountID, uint64]) AccountID {
	return AccountID{p.Into()}
}

type AccountName struct {
	refined.Scalar[AccountName, string]
}

var accountNameRx = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (AccountName) RefinedMatch(value string) bool { return accountNameRx.MatchString(value) }
func (AccountName) RefinedBuild(p refined.PreScalar[AccountName, string]) AccountName {
	return AccountName{p.Into()}
}

type AccountInfo = refined.Struct[AccountInfoAttributes]
type AccountInfoAttributes struct {
	ID   AccountID   `json:"id"`
	Name AccountName `json:"name"`
}

func (p AccountInfoAttributes) ReadableName() string {
	return fmt.Sprintf("%s (ID %d)", p.Name.Raw(), p.ID.Raw())
}

func TestAccountInfo(t *testing.T) {
	var info AccountInfo
	err := json.Unmarshal([]byte(`{"id":42,"name":"foo"}`), &info)
	AssertEqual(t, err, nil)
	AssertEqual(t, info.Has.ID.Raw(), 42)
	AssertEqual(t, info.Has.Name.Raw(), "foo")

	info = refined.NewStruct(AccountInfoAttributes{
		ID: refined.Literal[AccountID](53),
	})
	info.Has.Name = refined.Literal[AccountName]("hello")
	AssertEqual(t, info.Has.ReadableName(), "hello (ID 53)")

	buf, err := json.Marshal(info)
	AssertEqual(t, err, nil)
	AssertEqual(t, string(buf), `{"id":53,"name":"hello"}`)
}
