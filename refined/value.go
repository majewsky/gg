// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import (
	"cmp"
	"errors"
	"regexp"

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
	return empty.Refine(Challenge[S, V]{Value: value, valid: true})
}

func Literal[S IsAScalar[S, V], V any](value V) S {
	var empty S
	s, err := empty.Refine(Challenge[S, V]{Value: value, valid: true})
	if err != nil {
		panic("TODO 2")
	}
	return s
}

type IsAScalar[S any, V any] interface {
	Refine(Challenge[S, V]) (S, error)
}

type Challenge[S any, V any] struct {
	Value V
	valid bool
}

func (c Challenge[S, V]) Accept() Scalar[S, V] {
	if !c.valid {
		panic("broken Challenge object")
	}
	return Scalar[S, V]{value: Some(c.Value)}
}

func RangeCheck[S any, V cmp.Ordered](c Challenge[S, V], minimum, maximum V) (Scalar[S, V], error) {
	if minimum <= maximum && c.Value >= minimum && c.Value <= maximum {
		return c.Accept(), nil
	}
	return Scalar[S, V]{}, errors.New("TODO 3")
}

func RegexpCheck[S any, V ~string](c Challenge[S, V], rx *regexp.Regexp) (Scalar[S, V], error) {
	if rx.MatchString(string(c.Value)) {
		return c.Accept(), nil
	}
	return Scalar[S, V]{}, errors.New("TODO 4")
}

func NotZeroCheck[S any, V comparable](c Challenge[S, V]) (Scalar[S, V], error) {
	var zero V
	if c.Value != zero {
		return c.Accept(), nil
	}
	return Scalar[S, V]{}, errors.New("TODO 5")
}
