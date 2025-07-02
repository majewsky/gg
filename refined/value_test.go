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

func (DiceRoll) RefinedMatch(value int) bool                              { return value >= 1 && value <= 6 }
func (DiceRoll) RefinedBuild(p refined.PreScalar[DiceRoll, int]) DiceRoll { return DiceRoll{p.Into()} }

func TestDiceRoll(t *testing.T) {
	d := refined.Literal[DiceRoll](5)
	AssertEqual(t, d.Raw(), 5)

	var err error
	d, err = refined.New[DiceRoll](7)
	AssertEqual(t, d, DiceRoll{})
	AssertEqual(t, err.Error(), "TODO 2")
}
