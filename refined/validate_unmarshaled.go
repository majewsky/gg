// SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package refined

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	. "github.com/majewsky/gg/option"
)

// ValidateUnmarshaled calls IsValid() on each refinement type instance contained somewhere within the given data structure.
// For each instance of a refinement type where IsValid() returns false, an error is returned.
// Each returned error will be of type InvalidInstanceError.
//
// Invalid instances of refinement types can occur when unmarshaling data structures from JSON or similar formats.
// See documentation on func Scalar.IsValid() for a detailed explanation.
// After unmarshaling, the owner of the obtained data structure therefore needs to walk through it and check IsValid() on each instance of a refinement type.
//
// Because writing this check by hand is tedious, ValidateUnmarshaled() automates it.
// Call this function after your Unmarshal() call.
// For example with JSON:
//
//	var payload SomeComplexStructWithNestedArraysMapsAndSubstructs
//	err := json.Unmarshal(buf, &payload)
//	handle(err)
//	errs := refined.ValidateUnmarshaled(payload, 1)
//	handle(errs)
//	// if there were no errors, it is now guaranteed that
//	// each refined.Scalar etc. contained within `payload` will be valid
//
// This function can handle inputs containing cyclic references (as can occur when unmarshaling YAML that contains anchor backreferences),
// but it is not equipped to search in unexported struct fields.
func ValidateUnmarshaled(input any) []error {
	// As we traverse through the input, we need to keep track of where we are to report error locations correctly.
	// But in the happy case of no errors, we want to do this tracking in a way that minimizes allocations.
	// This is why we do all our tracking in this one path slice.
	// When we recurse down, we add additional elements to the slice.
	// And when we recurse back up, the previous function will still have the shorter slice and,
	// by extending it again on the next call, overwrite the previous subpath.
	path := make([]pathElement, 0, 16)

	// We could have validateRecursively() return a list of errors, but then we would have to do
	// useless extra allocations when building and then concatenating those lists of errors.
	// It is cheaper to have a single error slice that everyone appends into.
	var errs []error

	// And as one final piece of setup, our cycle detection needs a place to remember visited objects in.
	// We will only make entries here for heap-allocated objects that can actually participate in cycles.
	// The design of this type is adapted from type visit used by reflect.DeepEqual().
	vm := make(visitedMap)

	validateRecursively(reflect.ValueOf(input), path, vm, &errs)
	return errs
}

type pathElement struct {
	// If this is set, this path element is an index into an array.
	Index Option[int]
	// If this is set, this path element is a field within a struct.
	Field string
	// Otherwise, this path element is a key within a map using a non-string type.
	MapKey reflect.Value
}

type visitedKey struct {
	Ptr  unsafe.Pointer
	Type reflect.Type
}

type visitedMap map[visitedKey]bool

// IsVisited returns false exactly once for each value, and then true for each successive call.
func (m visitedMap) IsVisited(v reflect.Value) bool {
	k := visitedKey{v.UnsafePointer(), v.Type()}
	if m[k] {
		return true
	}
	m[k] = true // make next call to IsVisited return true instead
	return false
}

// InvalidInstanceError is the error type returned by ValidateUnmarshaled().
type InvalidInstanceError struct {
	// A user-readable description of where the invalid instance was found within the input.
	// Currently, this uses the JSON Pointer format of RFC 6901, but future versions may change this.
	Path string
	// The type of the invalid instance.
	Type reflect.Type
}

func newInvalidInstanceError(path []pathElement, type_ reflect.Type) InvalidInstanceError {
	// convert path into a JSON pointer [RFC6901]
	tokens := make([]string, len(path)+1)
	tokens[0] = "" // ensure leading slash
	for idx, elem := range path {
		var token string
		switch {
		case elem.Index.IsSome():
			token = strconv.FormatInt(int64(elem.Index.UnwrapOr(0)), 10)
		case elem.Field != "":
			token = elem.Field
		case elem.MapKey.Kind() == reflect.String:
			token = elem.MapKey.String()
		default:
			token = fmt.Sprintf("%v", elem.MapKey)
		}
		token = strings.ReplaceAll(token, "~", "~0")
		token = strings.ReplaceAll(token, "/", "~1")
		tokens[idx+1] = token
	}
	pathStr := strings.Join(tokens, "/")
	return InvalidInstanceError{pathStr, type_}
}

// Error implements the builtin/error interface.
func (e InvalidInstanceError) Error() string {
	return fmt.Sprintf("found an invalid %s at %s", e.Type.String(), e.Path)
}

func validateRecursively(v reflect.Value, path []pathElement, vm visitedMap, errs *[]error) {
	// This switch writes out each kind specifically, because we want to have
	// a loud error in the default case if later Go versions add new kinds.
	switch v.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.String:
		// All these types can absolutely not hold any refined values inside of them, so no work is necessary.

	case reflect.Interface, reflect.Pointer:
		if !v.IsNil() && !vm.IsVisited(v) {
			validateRecursively(v.Elem(), path, vm, errs)
		}

	case reflect.Slice:
		if v.IsNil() || vm.IsVisited(v) {
			return
		}
		fallthrough
	case reflect.Array:
		for idx := range v.Len() {
			subpath := append(path, pathElement{Index: Some(idx)})
			validateRecursively(v.Index(idx), subpath, vm, errs)
		}

	case reflect.Map:
		if !v.IsNil() && !vm.IsVisited(v) {
			iter := v.MapRange()
			for iter.Next() {
				subpath := append(path, pathElement{MapKey: iter.Key()})
				validateRecursively(iter.Value(), subpath, vm, errs)
			}
		}

	case reflect.Struct:
		if vt, ok := v.Interface().(validationTarget); ok {
			if !vt.IsValid() {
				*errs = append(*errs, newInvalidInstanceError(path, v.Type()))
			}
		} else {
			for idx := range v.NumField() {
				f := v.Field(idx)
				// We cannot recurse into unexported fields; otherwise v.Interface() above will panic when recursing into another struct.
				if f.CanInterface() {
					subpath := append(path, pathElement{Field: v.Type().Field(idx).Name})
					validateRecursively(f, subpath, vm, errs)
				}
			}
		}

	case reflect.UnsafePointer:
		// For this type, we do not have any method of looking inside, so we cannot do anything.

	case reflect.Invalid:
		fallthrough
	default:
		panic(fmt.Sprintf("do not know how to handle value %#v of kind %s", v, v.Kind().String()))
	}
}
