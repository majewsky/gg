// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import "encoding/json"

type Struct[T any] struct {
	Has T
}

func NewStruct[T any](attributes T) Struct[T] {
	return Struct[T]{Has: attributes}
}

func (s Struct[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Has)
}

func (s *Struct[T]) UnmarshalJSON(buf []byte) error {
	err := json.Unmarshal(buf, &s.Has)
	if err != nil {
		return err
	}

	// TODO reflect on the fields of s.Has; if any are refined.Value that are not occupied, attempt to fill the zero value through Refine()
	return nil
}
