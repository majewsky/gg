/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined

import (
	"fmt"
	"regexp"
)

// Building block for writing MatchesValue() implementations.
func RegexpMatch(rx *regexp.Regexp, value string) error {
	if !rx.MatchString(value) {
		return fmt.Errorf("provided value %q does not match expected pattern %q", value, rx.String())
	}
	return nil
}
