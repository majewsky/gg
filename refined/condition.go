/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: Apache-2.0
* Refer to the file "LICENSE" for details.
*******************************************************************************/

package refined

import (
	"errors"
	"regexp"
)

type Condition[T any] interface {
	MatchesValue(T) error
}

// Building block for writing MatchesValue() implementations.
func RegexpMatch(rx *regexp.Regexp, value string) error {
	if !rx.MatchString(value) {
		return errors.New("TODO: error message")
	}
	return nil
}
