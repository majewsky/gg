// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package refined provides refinement types for Go.
//
// A refinement type is built on top of a base type by adding a predicate that accepts or rejects values of the base type.
// An instance of the refinement type can only be constructed if the predicate accepts the proposed value.
// For example, one can use the base type "string" to derive the refinement type "GoIdentifier", using a predicate that only accepts string values that are valid identifiers in Go code.
//
// Refinement types based on this library make use of the type system to make invalid values unrepresentable.
// For example, if a piece of code obtains an instance of the aforementioned GoIdentifier type, the code shall be able to assume that the contained string value is a valid Go identifier, because it is not possible to construct a GoIdentifier instance holding a string value that is not.
// Creating and updating instances of refinement types is only possible through functions and methods in this package that check the proper predicate and throw an error or panic if the value in question is rejected by it.
//
// Because of how the Go language works, there are some significant restrictions on what refinement types can do.
//
// # Restriction #1: Base types must allow deep cloning
//
// When deriving a refinement type from a complex base type, any pointers hiding inside that base type would allow interior mutation that the refinement type cannot block.
// For example, if the refinement type's base type is a struct that contains a []byte somewhere, getting a shallow copy of the held value would allow us to overwrite elements of that slice without needing to uphold the predicate of the refinement type.
//
// To guard against this, we only allow base types that are scalars (single numbers or string values), through type Scalar, or struct types that can deep-clone themselves, through type Struct.
// Refinement types based on structs will refuse to give out shallow copies, which may impose a significant performance penalty for structs containing complex data structures.
// Other structured base types like slices or maps may be added in the future if we can devise an API that is both safe and sufficiently ergonomic.
//
// # Restriction #2: Unmarshaling
//
// Go does a lot of useful things through reflection, the most significant of them being marshalling to/from wire formats like JSON or SQL.
// Reflection sometimes constructs zero-valued instances of arbitrary types without giving the type a chance to intervene.
// To make sure that programs can never use instances of refinement types that did not check their predicate on construction, attempting to use a zero-valued instance of any refinement type results in a panic.
// When receiving data structures containing refinement types from a reflect-based generator like json.Unmarshal(), you need to use func ValidateUnmarshaled to check that all refinement type instances therein are intact.
package refined
