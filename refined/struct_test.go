// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined_test

import (
	"fmt"
	"math"
	"regexp"
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

type AccountID struct {
	refined.Scalar[AccountID, uint64]
}

func (AccountID) Refine(c refined.Challenge[AccountID, uint64]) (AccountID, error) {
	s, err := refined.RangeCheck(c, 1, math.MaxUint64)
	return AccountID{s}, err
}

type AccountName struct {
	refined.Scalar[AccountName, string]
}

var accountNameRx = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func (AccountName) Refine(c refined.Challenge[AccountName, string]) (AccountName, error) {
	s, err := refined.RegexpCheck(c, accountNameRx)
	return AccountName{s}, err
}

type AccountInfo = refined.Struct[AccountInfoPayload]

type AccountInfoPayload struct {
	ID   AccountID   `json:"id,omitzero"`
	Name AccountName `json:"name,omitzero"`
}

func (p AccountInfoPayload) ReadableName() string {
	return fmt.Sprintf("%s (ID %d)", p.Name.Raw(), p.ID.Raw())
}

func TestAccountInfo(t *testing.T) {
	var info AccountInfo //nolint:staticcheck
	info = refined.NewStruct(AccountInfoPayload{
		ID: refined.Literal[AccountID](53),
	})
	info.Has.Name = refined.Literal[AccountName]("hello")
	AssertEqual(t, info.Has.ReadableName(), "hello (ID 53)")
}
