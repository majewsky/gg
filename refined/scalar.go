// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	. "github.com/majewsky/gg/option"
)

// Scalar provides support for refinement types derived from scalar base types (individual numbers or strings).
// This type is used to declare a refinement type, by placing a reflect.Scalar type inside a struct type as an embedded field:
//
//	type AccountName struct {
//		refined.Scalar[AccountName, string]
//	}
//
// Like in this example, the first type argument of Scalar (S) is always the outer struct type, and the second type argument (V) is the base type.
// The construction using an embedded field on a struct type allows you to define additional convenience methods in the same way that you would on a newtype,
// while also allowing the struct type to expose method implementations provided by type Scalar (e.g. MarshalJSON):
//
//	// using a newtype
//	type AccountName string
//	func (n AccountName) IsReserved() bool {
//		return strings.HasPrefix(string(n), "__")
//	}
//
//	// using a refinement type instead
//	type AccountName struct {
//		refined.Scalar[AccountName, string]
//	}
//	func (n AccountName) IsReserved() bool {
//		// retrieves the underlying value using the Unpack method of type Scalar
//		return strings.HasPrefix(n.Unpack(), "__")
//	}
//
// To make the refinement type work, type Scalar needs to know the predicate that applies to this type.
// This is why the struct type is given to type Scalar as the type argument S.
// The struct type must implement the IsAScalar interface. Continuing the example:
//
//	var accountNameRx = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
//
//	func (AccountName) RefinedMatch(value string) bool {
//		// This function decides which values are acceptable for the refinement type.
//		return accountNameRx.MatchString(value)
//	}
//	func (AccountName) RefinedBuild(s refined.PreScalar[AccountName, string]) AccountName {
//		// This function allows the library to cast a bare Scalar instance into the full struct type.
//		return AccountName{refined.PromoteScalar(s)}
//	}
//
// Marshaling into and from YAML using https://github.com/go-yaml/yaml (or one of its many forks) is supported.
// The "omitempty" flag works as expected.
//
// Marshaling into and from JSON using encoding/json is supported, but the "omitempty" flag does not work.
// You must use the "omitzero" flag to get the same effect.
type Scalar[S IsAScalar[S, V], V ScalarValue] struct {
	value Option[V]
}

// ScalarValue is an interface that is implemented by all scalar types that do not allow interior mutation.
// That is to say, any operation on a type in this interface must always yield a fresh value without editing the inside of the existing value.
// For example, []byte is not in this interface because edits made on a copy of a []byte value can affect the original:
//
//	original := []byte("foo")
//	copied := original
//	copied[1] = 'x'
//	fmt.Println(string(original)) // prints "fxo", not "foo"!
type ScalarValue interface {
	~bool |
		~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~complex64 | ~complex128 |
		~string
}

// IsAScalar is an interface that must be implemented by user-defined struct types holding an instance of type Scalar.
// See documentation on type Scalar for an explanation and example of how these two types are interconnected.
type IsAScalar[S any, V ScalarValue] interface {
	// RefinedMatch implements the predicate for the refinement type.
	// Given a value of the scalar base type, this function shall return whether that value is acceptable for the refinement type.
	// If false is returned, package refined will ensure that no instance of Scalar[S, V] with that value is ever constructed.
	//
	// For technical reasons, this function is a method on type S, but the implementation shall not use the receiver value.
	// Package refined will always call this method on an instance of S that is invalid and not ready to use other than for calling this method.
	RefinedMatch(V) bool

	// RefinedBuild allows package refined to cast a bare Scalar instance into the full user-defined struct type.
	// Instead of a Scalar instance, this method receives
	//
	// For technical reasons, this function is a method on type S, but the implementation shall not use the receiver value.
	// Package refined will always call this method on an instance of S that is invalid and not ready to use other than for calling this method.
	// The implementation shall always return a freshly constructed instance of type S that is not related to the receiver value.
	RefinedBuild(PreScalar[S, V]) S
}

// PreScalar is like Scalar, but with a weaker type bound.
//
// This type is only needed in one place, in the declaration of RefinedBuild(),
// to break a dependency cycle between the definition of type Scalar and type IsAScalar.
//
// It is not proper to construct instances of this type outside of package refined.
// Attempting to use a zero-initialized value of this type will result in a panic.
type PreScalar[S any, V ScalarValue] struct {
	value Option[V]
}

// PromoteScalar converts a PreScalar into a Scalar, thus strengthening its type bound.
//
// This function is only needed in one place, in implementations of RefinedBuild(),
// to break a dependency cycle between the definition of type Scalar and type IsAScalar.
func PromoteScalar[S IsAScalar[S, V], V ScalarValue](p PreScalar[S, V]) Scalar[S, V] {
	if p.value.IsNone() {
		// `value = None()` only occurs when user code outside of package refined
		// creates a zero-valued instance of PreScalar like this:
		//
		//	var ps refined.PreScalar[S, V]
		//	s := refined.PromoteScalar(ps)
		//
		// or through reflection. This is illegal. Only PreScalar instances
		// constructed within package refined are legal to use because package
		// refined will ensure that the predicate of S is upheld.
		panic("PromoteScalar received an illegally constructed PreScalar instance")
	}
	return Scalar[S, V](p)
}

////////////////////////////////////////////////////////////////////////////////
// generic methods on Scalar[S, V]

// refinedSeal implements the isARefinementType interface.
func (s Scalar[S, V]) refinedSeal() seal {
	// This method is never called. It's only part of the interface to make it unimplementable outside this package.
	return seal{}
}

// refinedNew implements the isARefinementType interface.
func (s Scalar[S, V]) refinedNew(value V) (S, error) {
	var empty S
	err := checkScalarValue[S, V](value)
	if err == nil {
		return empty.RefinedBuild(PreScalar[S, V]{Some(value)}), nil
	} else {
		return empty, err
	}
}

func checkScalarValue[S IsAScalar[S, V], V ScalarValue](value V) error {
	var empty S
	if empty.RefinedMatch(value) {
		return nil
	} else {
		return fmt.Errorf("value %#v is not acceptable for %T", value, empty)
	}
}

// IsValid returns whether the scalar holds a valid value.
// This will only ever be false for zero-valued Scalar instances:
//
//	type AccountName struct {
//		refined.Scalar[AccountName, string]
//	}
//
//	name1, err := refined.New[AccountName]("example")
//	if err == nil {
//		fmt.Println(name1.IsValid()) // prints true
//	}
//
//	var name2 AccountName        // not initialized to a valid value!
//	fmt.Println(name2.IsValid()) // prints false
//
//	type AccountData struct {
//		ID   int64
//		Name AccountName
//	}
//	data := AccountData {
//		ID: 42,
//		// Name is not initialized!
//	}
//	fmt.Println(data.Name.IsValid()) // prints false
//
// Most functions handling refinement types should not have to call this method.
// Access the refinement type's value directly through Unpack() or any other method on Scalar.
// If that panics, you should be catching that in a test.
//
// The most common situation where IsValid() truly needs to be used is when unmarshaling data structures from JSON or similar formats.
// Unmarshalers often leave struct fields unfilled if they are not mentioned in the data, and package refined cannot intervene in that
// because technically nothing gets unmarshaled into the Scalar value and so no code is executed. For example:
//
//	var accountData struct {
//		ID   int64
//		Name AccountName
//	}
//	input := `{"ID":42}`                               // "Name" key is missing!
//	err := json.Unmarshal([]byte(input), &accountData) // will succeed (err == nil)
//	fmt.Println(accountData.Name.IsValid())            // prints false
//
// When unmarshaling data structures that contain refinement type values, use func ValidateUnmarshaled instead of this method.
func (s Scalar[S, V]) IsValid() bool {
	return s.value.IsSome()
}

// Unpack returns a copy of the raw value inside this scalar.
// Panics if called on a zero value; see func IsValid for details.
func (s Scalar[S, V]) Unpack() V {
	return s.value.UnwrapOrPanic("Unpack() called on an illegally constructed Scalar instance")
}

////////////////////////////////////////////////////////////////////////////////
// formatting/marshalling support for Scalar[S, V]

// Format implements the fmt.Formatter interface.
//
// For most verbs, the contained value will be formatted as if it was given directly.
// For %v, the %v representation of the contained value is wrapped in "TypeName[]".
// For %#v, a refined.Literal() invocation is formatted.
func (s Scalar[S, V]) Format(f fmt.State, verb rune) {
	val := s.Unpack()
	if verb != 'v' {
		fmt.Fprintf(f, fmt.FormatString(f, verb), val)
		return
	}

	var empty S
	sName := fmt.Sprintf("%T", empty)
	inner := fmt.Sprintf(fmt.FormatString(f, verb), val)
	if f.Flag('#') {
		fmt.Fprintf(f, "refined.Literal[%s](%s)", sName, inner)
	} else {
		fmt.Fprintf(f, "%s[%s]", sName, inner)
	}
}

// Value implements the database/sql/driver.Valuer interface.
//
// If you want to get the contained value, use Unpack().
// This name is unfortunately taken by an interface from the standard library.
func (s Scalar[S, V]) Value() (driver.Value, error) {
	return driver.DefaultParameterConverter.ConvertValue(s.Unpack())
}

// Scan implements the database/sql.Scanner interface.
func (s *Scalar[S, V]) Scan(src any) error {
	// We cannot scan `src` into V directly because the required function (database/sql.convertAssign) is private.
	// sql.Null[V].Scan() is the next best thing, but it allows `src = nil` even though no possible choice for V does,
	// so we need to catch this case ourselves.
	var (
		data  sql.Null[V]
		empty S
	)
	err := data.Scan(src)
	if err != nil {
		return err
	}

	if !data.Valid {
		// this mimics the error message that database/sql would generate when scanning `src = nil` into V directly
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, empty)
	}

	err = checkScalarValue[S, V](data.V)
	if err == nil {
		*s = Scalar[S, V]{Some(data.V)}
	}
	return err
}

// IsZero implements the IsZeroer interface as understood by encoding/json and github.com/go-yaml/yaml.
// It returns whether the underlying value is zero.
func (s Scalar[S, V]) IsZero() bool {
	value := s.Unpack()
	var zero V
	return value == zero
}

// MarshalJSON implements the encoding/json.Marshaler interface.
func (s Scalar[S, V]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Unpack())
}

// UnmarshalJSON implements the encoding/json.Unmarshaler interface.
func (s *Scalar[S, V]) UnmarshalJSON(buf []byte) error {
	var raw V
	err := json.Unmarshal(buf, &raw)
	if err != nil {
		return err
	}
	err = checkScalarValue[S, V](raw)
	if err == nil {
		*s = Scalar[S, V]{Some(raw)}
	}
	return err
}

type yamlMarshaler interface {
	MarshalYAML() (any, error)
}

// MarshalYAML implements the yaml.Marshaler interface from gopkg.in/yaml.v2 and v3 as well as their forks.
func (s Scalar[S, V]) MarshalYAML() (any, error) {
	value := s.Unpack()
	// If we just return `value` directly here, MarshalYAML will not be called
	// on the value even if it exists. For this one specific case, we have to
	// take care ourselves.
	if m, ok := any(value).(yamlMarshaler); ok {
		return m.MarshalYAML()
	} else {
		return value, nil
	}
}

// UnmarshalYAML implements the yaml.Unmarshaler interface from gopkg.in/yaml.v2.
//
// gopkg.in/yaml.v3 and its forks support this interface via backwards-compatibility,
// so we intentionally do not use the v3-only signature that refers to the yaml.Node type.
func (s *Scalar[S, V]) UnmarshalYAML(unmarshal func(any) error) error {
	var raw V
	err := unmarshal(&raw)
	if err != nil {
		return err
	}
	err = checkScalarValue[S, V](raw)
	if err == nil {
		*s = Scalar[S, V]{Some(raw)}
	}
	return err
}

// TODO: func ValidateUnmarshaled()
