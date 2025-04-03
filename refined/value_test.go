// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined_test

import (
	"testing"

	. "github.com/majewsky/gg/internal/test"
	"github.com/majewsky/gg/refined"
)

type DiceRoll struct {
	refined.Scalar[DiceRoll, int]
}

func (DiceRoll) Refine(c refined.Challenge[DiceRoll, int]) (DiceRoll, error) {
	s, err := refined.RangeCheck(c, 1, 6)
	return DiceRoll{s}, err
}

func TestDiceRoll(t *testing.T) {
	d := refined.Literal[DiceRoll](5)
	AssertEqual(t, d.Raw(), 5)

	var err error
	d, err = refined.New[DiceRoll](7)
	AssertEqual(t, d, DiceRoll{})
	AssertEqual(t, err.Error(), "TODO 3")
}
