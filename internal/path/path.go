// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package path

import (
	"encoding/json"
	"strconv"
	"strings"

	. "go.xyrillian.de/gg/option"
)

// Path is used to identify the current location within a nested data structure
// while recursing through it. For example, when comparing
//
//	actual = { "foo": { "bar": [ 5, 23 ] } }
//	expected = { "foo": { "bar": [ 5, 42 ] } }
//
// we would generate a diff at the path {"foo", "bar", 1}.
// Since diffs are usually rare, we only build Pointer strings
// out of these paths when we really need them.
// During recursion, Path holds a sequence of path elements,
// most of which are constants to keep allocations to a minimum.
//
// # Warning
//
// Because the same Path slice is heavily reused across nested function calls,
// it is not safe to store references to the Path slice during such a recursion.
type Path []Element

// Path preallocates a decently sized path buffer for use in a recursion.
func NewPath() Path {
	return make([]Element, 0, 32)
}

// Element occurs in type [Path]. Only one of both fields is set per instance.
type Element struct {
	Key   Option[string]
	Index int
}

// KeyElement is a shorthand for constructing an Element with the Key field set.
func KeyElement(key string) Element { return Element{Some(key), 0} }

// IndexElement is a shorthand for constructing an Element with the Index field set.
func IndexElement(idx int) Element { return Element{None[string](), idx} }

// AsJSONPointer serializes p as a JSON pointer (RFC 6901).
func (p Path) AsJSONPointer() string {
	if len(p) == 0 {
		return ""
	}
	fragments := make([]string, len(p)+1)
	fragments[0] = ""
	for idx, elem := range p {
		if key, ok := elem.Key.Unpack(); ok {
			fragments[idx+1] = keyIntoPointerFragment(key)
		} else {
			fragments[idx+1] = strconv.Itoa(elem.Index)
		}
	}
	return strings.Join(fragments, "/")
}

func keyIntoPointerFragment(key string) string {
	buf, _ := json.Marshal(key)
	s := string(buf)
	s = strings.TrimPrefix(s, "\"")
	s = strings.TrimSuffix(s, "\"")
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}
