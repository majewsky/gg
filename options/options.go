/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

// Package options provides additional functions for type option.Option
// that cannot be expressed as methods on the Option type itself.
package options

import . "github.com/majewsky/gg/option"

// NOTE: Keep functions sorted by name.

// FromPointer converts a *T into an Option[T].
func FromPointer[T any](value *T) Option[T] {
	if value == nil {
		return None[T]()
	} else {
		return Some(*value)
	}
}
