// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

type Struct[P any] struct {
	Has P
}

func NewStruct[P any](payload P) Struct[P] {
	return Struct[P]{Has: payload}
}
