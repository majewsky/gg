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

type Builder[T any, V any] interface {
	MatchesValue(T) error
	Build(T, Verification) V
}

type Verification interface {
	isVerification(verificationSeal)
}

type verification struct{}

type verificationSeal struct{}

func (verification) isVerification(_ verificationSeal) {}

// Building block for writing MatchesValue() implementations.
func RegexpMatch(rx *regexp.Regexp, value string) error {
	if !rx.MatchString(value) {
		return errors.New("TODO: error message")
	}
	return nil
}
