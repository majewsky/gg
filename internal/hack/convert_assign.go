/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package hack

import (
	_ "database/sql"
	_ "unsafe"
)

//go:linkname ConvertAssign database/sql.convertAssign
func ConvertAssign(dest, src any) error

// ^ NOTE: This is needed because of <https://github.com/golang/go/issues/62146>
