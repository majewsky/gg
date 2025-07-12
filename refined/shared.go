// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

// A private type that appears in interfaces to make them unimplementable for types outside this package.
type seal struct{}

// This interface is currently only implemented by Scalar[S, V].
// But by having this interface as an intermediate, New() and Literal() can
// work with other classes of refinement types in the future.
type isARefinementType[S any, V any] interface {
	refinedSeal() seal
	refinedNew(V) (S, error)
}

// This interface is currently only implemented by Scalar[S, V].
// But by having this interface as an intermediate, ValidateUnmarshaled() can
// work with other classes of refinement types in the future.
//
// Because of how the reflection logic in ValidateUnmarshaled() is written,
// this interface is restricted to not have any type parameters.
type validationTarget interface {
	refinedSeal() seal
	IsValid() bool
}

// New checks if the provided value is acceptable for the refinement type S,
// and returns an instance of S if so, or an error if not.
// For example, given a refinement type like:
//
//	type AccountName struct {
//		refined.Scalar[AccountName, string]
//	}
//	// not shown: implementation of refined.IsAScalar[AccountName, string] interface on AccountName
//
// An instance of this refinement type can be constructed like so:
//
//	var userInput string
//	accountName, err := refined.New[AccountName](userInput)
func New[S isARefinementType[S, V], V any](value V) (S, error) {
	var empty S
	return empty.refinedNew(value)
}

// Literal is like New, but panics on error.
// This function is intended for dealing with literal values, where unexpected errors are not possible.
// For example:
//
//	receiverName, err := refined.New[NonEmptyString]("Beitragsservice")
//	handle(err)
//	postCode, err := refined.New[PostCode](50616)
//	handle(err)
//	town, err := refined.New[NonEmptyString]("Cologne")
//	handle(err)
//	testAddress := Address {
//		ReceiverName: receiverName,
//		PostCode:     postCode,
//		Town:         town,
//	}
//
// can be shortened to:
//
//	testAddress := Address {
//		ReceiverName: refined.Literal[NonEmptyString]("Beitragsservice"),
//		PostCode:     refined.Literal[PostCode](50616),
//		Town:         refined.Literal[NonEmptyString]("Cologne"),
//	}
func Literal[S isARefinementType[S, V], V any](value V) S {
	var empty S
	s, err := empty.refinedNew(value)
	if err != nil {
		panic(err.Error())
	}
	return s
}
