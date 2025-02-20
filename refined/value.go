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

// NOTE: The zero value is illegal and will panic on use.
type Value[B Builder[T, V], V any, T any] struct {
	value Option[T]
}

func Build[B Builder[T, V], V any, T any](value T, _ Verification) Value[B, V, T] {
	return Value[B, V, T]{Some(value)}
}

func New[B Builder[T, V], V any, T any](value T) (V, error) {
	var b B
	err := b.MatchesValue(value)
	if err != nil {
		var empty V
		return empty, err
	}
	return b.Build(value, verification{}), nil
}

func Literal[B Builder[T, V], V any, T any](value T) V {
	result, err := New[B](value)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func newValue[B Builder[T, V], V any, T any](value T) (Value[B, V, T], error) {
	var b B
	err := b.MatchesValue(value)
	if err != nil {
		return Value[B, V, T]{None[T]()}, err
	}
	return Value[B, V, T]{Some(value)}, nil
}

func (v Value[B, V, T]) Raw() T {
	return v.value.UnwrapOrPanic("illegal use of zero-valued instance of Refined type")
}

func (v *Value[B, V, T]) UnmarshalJSON(buf []byte) error {
	var value T
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}
	*v, err = newValue[B, V, T](value)
	return err
}

func (v Value[B, V, T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Raw())
}
