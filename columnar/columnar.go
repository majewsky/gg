// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

// Package columnarjson provides efficient encoding of lists of objects in a columnar JSON format.
//
// The standard way of encoding a list of objects in JSON looks like this:
//
//	[
//		{ "id": 1, "first_name": "Alice", "last_name": "Allison", "married": false },
//		{ "id": 2, "first_name": "Bob", "last_name": "Burger", "married": true },
//		{ "id": 3, "first_name": "Carol", "last_name": "Callagher", "married": true }
//	]
//
// Encoding the same list in a columnar fashion results in this:
//
//	{
//		"id": [1, 2, 3],
//		"first_name": ["Alice", "Bob", "Carol"],
//		"last_name": ["Allison", "Burger", "Callagher"],
//		"married": [false, true, true]
//	}
//
// In this example, changing the encoding from row-wise to columnar reduced the
// (minified) size of the JSON encoding from 202 to 124 bytes.
//
// This package eliminates the boilerplate code that would be associated with
// converting a list of objects into the respective columnar form before
// marshaling, and vice versa after unmarshaling.

package columnar

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
)

// NOTE: naming convention for variables
//
// single-letter = original type (t = reflect.Type, v = reflect.Value, f = reflect.StructField)
// with "c" prefix = columnar type (ct = reflect.Type, cv = reflect.Value, cf = reflect.StructField)

// prove interface implementations
var _ interface {
	json.Marshaler
	json.Unmarshaler
} = &List[bool]{}

// cache for auto-generated columnar struct types
var columnarListTypes = map[reflect.Type]reflect.Type{}

// List provides columnar marshaling for lists of objects.
// T must be a struct type or a pointer to one, otherwise all methods on this type will panic.
//
// Please refer to the package docstring for how this type is marshaled.
type List[T any] []T

func foreachRelevantField(t reflect.Type, action func(f reflect.StructField)) {
	for idx := range t.NumField() {
		f := t.Field(idx)
		if f.PkgPath == "" {
			action(f)
		}
	}
}

func getColumnarType(t reflect.Type) reflect.Type {
	if t.Kind() != reflect.Struct {
		zero := reflect.New(t).Elem().Interface()
		panic(fmt.Sprintf("type %T is not a struct or pointer to a struct", zero))
	}

	result, ok := columnarListTypes[t]
	if ok {
		return result
	}

	var fields []reflect.StructField
	foreachRelevantField(t, func(f reflect.StructField) {
		fields = append(fields, reflect.StructField{
			Name: f.Name,
			Type: reflect.SliceOf(f.Type),
			Tag:  f.Tag,
		})
	})

	result = reflect.StructOf(fields)
	columnarListTypes[t] = result
	return result
}

// MarshalJSON implements the [json.Marshaler] interface.
func (l List[T]) MarshalJSON() ([]byte, error) {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	ct := getColumnarType(t)
	cv := reflect.New(ct).Elem()

	columns := make(map[string]reflect.Value, t.NumField())
	foreachRelevantField(t, func(f reflect.StructField) {
		column := reflect.MakeSlice(reflect.SliceOf(f.Type), len(l), len(l))
		cv.FieldByName(f.Name).Set(column)
		columns[f.Name] = column
	})
	if len(columns) == 0 {
		zero := reflect.New(t).Elem().Interface()
		return nil, fmt.Errorf("%[1]T has no exported fields", zero)
	}

	for idx, elem := range l {
		v := reflect.ValueOf(elem)
		for v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		foreachRelevantField(t, func(f reflect.StructField) {
			columns[f.Name].Index(idx).Set(v.FieldByIndex(f.Index))
		})
	}

	return json.Marshal(cv.Interface())
}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (l *List[T]) UnmarshalJSON(buf []byte) error {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	ct := getColumnarType(t)
	cv := reflect.New(ct)

	err := json.Unmarshal(buf, cv.Interface())
	if err != nil {
		return err
	}
	cv = cv.Elem()

	columns := make(map[string]reflect.Value, t.NumField())
	lengths := make(map[int]int)
	foreachRelevantField(t, func(f reflect.StructField) {
		column := cv.FieldByName(f.Name)
		columns[f.Name] = column
		lengths[column.Len()]++
	})

	switch len(lengths) {
	case 0:
		zero := reflect.New(t).Elem().Interface()
		return fmt.Errorf("%[1]T has no exported fields", zero)
	case 1:
		for length := range lengths {
			*l = make(List[T], length)
			break
		}
	default:
		return fmt.Errorf("cannot unmarshal from columns with inconsistent lengths %v", slices.Sorted(maps.Keys(lengths)))
	}

	for idx := range *l {
		v := reflect.ValueOf(&(*l)[idx]).Elem()
		for v.Kind() == reflect.Pointer {
			v.Set(reflect.New(v.Type().Elem()))
			v = v.Elem()
		}
		foreachRelevantField(t, func(f reflect.StructField) {
			v.FieldByIndex(f.Index).Set(columns[f.Name].Index(idx))
		})
	}
	return nil
}
