// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import . "github.com/majewsky/gg/option"

type Scalar struct {
	value Option[any]
}

// TODO: func ValidateUnmarshaled()
// TODO: constrain base type of Scalar to be a scalar via an explicit type enum
