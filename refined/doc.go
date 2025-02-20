/*******************************************************************************
* Copyright 2025 Stefan Majewsky <majewsky@gmx.net>
* SPDX-License-Identifier: GPL-3.0-only
* Refer to the file "LICENSE" for details.
*******************************************************************************/

// Package refined implements refinement types. Those are types that are constrained by a condition.
// For example, consider the following type representing Go variable names:
//
//	var variableNameRx = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z_0-9]*$`)
//
//	type VariableName string
//
// If you want for this type to only ever contain string values that are valid variable names,
// you would need to ensure that the value is first checked for validity whenever a VariableName instance is created.
// Since programmers inevitably make mistakes, we should rather have the type checker do this work for us.
// We can do this by wrapping the string inside type VariableName into a refined.Value like so:
//
//	type VariableName struct {
//		refined.Value[VariableName, string]
//	}
//
//	func (VariableName) MatchesValue(value string) error { return refined.RegexpMatch(variableNameRx, value) }
//	func (VariableName) Build(v refined.Prevalue[VariableName, string]) VariableName { return VariableName{refined.Build(v)} }
//
// # Why do we need so much boilerplate to declare a refinement type?
//
// As far as I'm aware, this is genuinely the least possible amount of boilerplate.
// To see why, let's go back to the initial form:
//
//	type VariableName string
//
// A newtype like this is not type-safe since it can be constructed by anyone at any time, without checking the implied constraint:
//
//	var illegalVariableName = VariableName("what is this?") // value does not match variableNameRx!
//
// We need to fully seal the contained string value away and prevent outsiders from messing with it without going through a constraint check.
// The only way to do this is with a struct:
//
//	type VariableName struct {
//		...
//	}
//
// Next, we want for this library to provide premade implementations of interfaces like json.Unmarshaler or sql.Scanner to all refinement types.
// This ensures that values inserted into the VariableName type during unmarshaling are properly checked against the refinement type's constraint.
// The respective functions are declared on the refined.Value type and can be inherited by making refined.Value an embedded field:
//
//	type VariableName struct {
//		refined.Value
//	}
//
// The refined.Value type obviously needs a type argument to know which raw value type it's holding:
//
//	type VariableName struct {
//		refined.Value[string]
//	}
//
// But refined.Value also needs a second type argument: It needs to be able to reach the MatchesValue() function that contains the refinement type's constraint.
// This is technically just a single function, but Go does not support providing raw functions as type arguments;
// you need a type implementing the desired function through an interface.
// I decided to require that this interface be implemented on the value type itself, so you don't have to declare a second bogus type:
//
//	type VariableName struct {
//		refined.Value[VariableName, string]
//	}
//
//	func (VariableName) MatchesValue(value string) error { ... }
//
// This would technically be enough, but it would not be very ergonomic.
// Supposing that we have a refined.NewValue() function that constructs refined.Value instances, we could construct VariableName instances like so:
//
//	name := VariableName{Value: refined.NewValue[VariableName](rawName)}
//
// This is rather convoluted and repeats words twice within a single line.
// And that's before considering that refined.NewValue() really ought to be returning an error for when rawName is not a valid variable name:
//
//	nameValue, err := refined.NewValue[VariableName](rawName)
//	if err != nil {
//		...
//	}
//	name := VariableName{Value: nameValue}
//
// We could wrap this in helper functions near the declaration of the VariableName type:
//
//	func NewVariableName(rawName string) (VariableName, error) {
//		v, err := refined.NewValue[VariableName](rawName)
//		return VariableName{v}, err
//	}
//
//	func MustNewVariableName(rawName string) VariableName { // for literals
//		return VariableName{refined.MustNewValue[VariableName](rawName)}
//	}
//
// But I don't want to copy-paste this for every single refinement type. Instead, we add one more method to the interface that already contains MatchValue.
// Our full type declaration becomes:
//
//	type VariableName struct {
//		refined.Value[VariableName, string]
//	}
//
//	func (VariableName) MatchesValue(value string) error { ... }
//	func (VariableName) Build(v refined.Prevalue[VariableName, string]) VariableName { return VariableName{refined.Build(v)} }
//
// This allows us to use library functions with nice names to construct instances of refinement types:
//
//	name, err := refined.New[VariableName](rawName)
//	fooName := refined.Literal[VariableName]("foo")
//
// The library functions check the constraint, build a refined.Prevalue, and then use the Build() method on VariableName to wrap those into proper VariableName instances.
// It would be nice if we could use the actual refined.Value type in Build() instead of its weird sibling refined.Prevalue:
//
//	func (VariableName) Build(v refined.Value[VariableName, string]) VariableName { return VariableName{v} }
//
// But that results in a recursive type declaration in the library, which the Go compiler rejects.
package refined
