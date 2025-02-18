/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined

import (
	"encoding/json"
	"errors"
	"regexp"

	. "github.com/majewsky/gg/option"
)

// NOTE: The zero value is illegal and will panic on use.
type Refined[Self Condition[T], T any] struct {
	value Option[T]
}

type Condition[T any] interface {
	MatchesValue(T) error
}

func Refine[C Condition[T], T any](value T) (Refined[C, T], error) {
	var c C
	err := c.MatchesValue(value)
	if err != nil {
		return Refined[C, T]{None[T]()}, err
	}
	return Refined[C, T]{Some(value)}, nil
}

func (r Refined[Self, T]) GetValue() T {
	return r.value.UnwrapOrPanicf("illegal use of zero-valued instance of Refined type")
}

func (r *Refined[Self, T]) UnmarshalJSON(buf []byte) error {
	var value T
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}
	*r, err = Refine[Self](value)
	return err
}

func (r Refined[Self, T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.GetValue())
}

// Building block for writing MatchesValue() implementations.
func RegexpMatch(rx *regexp.Regexp, value string) error {
	if !rx.MatchString(value) {
		return errors.New("TODO: error message")
	}
	return nil
}
