/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined

import (
	"encoding/json"

	. "github.com/majewsky/gg/option"
)

// TODO: how do we express a literal constructor, preferably in a generic way? e.g. `var demoAccountName = refined.Literal[AccountName]("demo")`

// NOTE: The zero value is illegal and will panic on use.
type Value[Self Condition[T], T any] struct {
	value Option[T]
}

func NewValue[C Condition[T], T any](value T) (Value[C, T], error) {
	var c C
	err := c.MatchesValue(value)
	if err != nil {
		return Value[C, T]{None[T]()}, err
	}
	return Value[C, T]{Some(value)}, nil
}

func (v Value[Self, T]) Get() T {
	return v.value.UnwrapOrPanic("illegal use of zero-valued instance of Refined type")
}

func (v *Value[Self, T]) UnmarshalJSON(buf []byte) error {
	var value T
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}
	*v, err = NewValue[Self](value)
	return err
}

func (v Value[Self, T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Get())
}
