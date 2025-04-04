/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined_test

import (
	"math"
	"testing"

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

// TODO: AccountName.Refine()

type AccountInfo refined.Struct[struct {
	ID   AccountID   `json:"id,omitzero"`
	Name AccountName `json:"name,omitzero"`
}]

func TestAccountInfo(t *testing.T) {
	info := AccountInfo{}
	info.Has.ID = refined.Literal[AccountID](53)
}
