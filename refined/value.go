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
type Value[V ValueType[V, T], T any] struct {
	value Option[T]
}

type Prevalue[V any, T any] struct {
	value Option[T]
}

func Build[V ValueType[V, T], T any](v Prevalue[V, T]) Value[V, T] {
	if v.value.IsNone() {
		panic("illegal use of zero-valued instance of refined.Prevalue")
	}
	return Value[V, T]{v.value}
}

func New[V ValueType[V, T], T any](value T) (V, error) {
	var builder V
	err := builder.MatchesValue(value)
	if err != nil {
		var empty V
		return empty, err
	}
	return builder.Build(Prevalue[V, T]{Some(value)}), nil
}

func Literal[V ValueType[V, T], T any](value T) V {
	result, err := New[V](value)
	if err != nil {
		panic(err.Error())
	}
	return result
}

func newValue[V ValueType[V, T], T any](value T) (Value[V, T], error) {
	var builder V
	err := builder.MatchesValue(value)
	if err != nil {
		return Value[V, T]{None[T]()}, err
	}
	return Value[V, T]{Some(value)}, nil
}

func (v Value[V, T]) Raw() T {
	return v.value.UnwrapOrPanic("illegal use of zero-valued instance of refined.Value")
}

func (v *Value[V, T]) UnmarshalJSON(buf []byte) error {
	var value T
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}
	*v, err = newValue[V, T](value)
	return err
}

func (v Value[V, T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Raw())
}
