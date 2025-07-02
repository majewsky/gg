// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import (
	"encoding/json"
	"errors"

	. "github.com/majewsky/gg/option"
)

type Scalar[S any, V any] struct {
	value Option[V]
}

func (s Scalar[S, V]) Raw() V {
	return s.value.UnwrapOrPanic("TODO 1")
}

func New[S IsAScalar[S, V], V any](value V) (S, error) {
	var empty S
	if empty.RefinedMatch(value) {
		return empty.RefinedBuild(PreScalar[S, V]{value: Some(value)}), nil
	} else {
		return empty, errors.New("TODO 2")
	}
}

func Literal[S IsAScalar[S, V], V any](value V) S {
	s, err := New[S, V](value)
	if err != nil {
		panic(err.Error())
	}
	return s
}

func (s Scalar[S, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Raw())
}

func (s *Scalar[S, V]) UnmarshalJSON(buf []byte) error {
	var value V
	err := json.Unmarshal(buf, &value)
	if err != nil {
		return err
	}

	// We cannot directly call `New[S, V](value)` here because we cannot prove statically
	// that `S` satisfies `IsAScalar[S, V]`.
	var empty S
	if r, ok := any(empty).(IsAScalar[S, V]); ok {
		if r.RefinedMatch(value) {
			*s = Scalar[S, V]{value: Some(value)}
			return nil
		} else {
			return errors.New("TODO 3")
		}
	} else {
		return errors.New("TODO 4")
	}
}

type IsAScalar[S any, V any] interface {
	// We need both steps separately. New() wants to have an S at the end, which goes through both steps.
	// But Unmarshal...() wants to obtain a Scalar[S, V], so it only wants to use the first step.
	RefinedMatch(V) bool
	RefinedBuild(PreScalar[S, V]) S
}

type PreScalar[S any, V any] struct {
	value Option[V]
}

func (p PreScalar[S, V]) Into() Scalar[S, V] {
	if p.value.IsNone() {
		panic("broken PreScalar object")
	}
	return Scalar[S, V](p)
}
