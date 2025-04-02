/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined

import (
	"encoding/json"

	//nolint:staticcheck // this dot import is fine (ST1001)
	. "github.com/majewsky/gg/option"
)

// NOTE: The zero value is illegal and will panic on use.
type Value[V any, C Condition[V]] struct {
	value Option[V]
}

type Condition[V any] interface {
	MatchesValue(V) error
}

func NewValue[V any, C Condition[V]](value V) (Value[V, C], error) {
	var cond C
	err := cond.MatchesValue(value)
	if err == nil {
		return Value[V, C]{value: Some(value)}, nil
	} else {
		return Value[V, C]{}, err
	}
}

func LiteralValue[V any, C Condition[V]](value V) Value[V, C] {
	var cond C
	err := cond.MatchesValue(value)
	if err == nil {
		return Value[V, C]{value: Some(value)}
	} else {
		panic(err.Error())
	}
}

func (v Value[V, C]) Raw() V {
	return v.value.UnwrapOrPanic("illegal use of zero-valued instance of refined.Value")
}

func (v *Value[V, C]) UnmarshalJSON(buf []byte) error {
	var value V
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}
	*v, err = NewValue[V, C](value)
	return err
}

func (v Value[V, C]) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Raw())
}
