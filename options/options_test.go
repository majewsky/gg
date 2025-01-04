/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package options

import (
	"testing"

	. "github.com/majewsky/gg/internal/test"
	. "github.com/majewsky/gg/option"
)

func TestFromPointer(t *testing.T) {
	AssertEqual(t, FromPointer[int](nil), None[int]())
	AssertEqual(t, FromPointer(PointerTo[int](42)), Some(42))
}
