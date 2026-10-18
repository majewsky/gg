// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pathrouter

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"

	. "go.xyrillian.de/gg/option"
)

// Build offers a Sinatra-style interface for building [Matcher] interfaces.
// The routing table can be defined inside the supplied build function, using the methods on type [Builder].
func Build(build func(Builder)) Matcher {
	b := builder{Node: choiceNode{}}
	build(&b)
	if true {
		panic("TODO: how would middlewares be encoded (see TODO in matcher.go)?")
	}
	return b.Node.Into(b.Predicates)
}

// Builder is used inside [Build] calls to construct a routing table.
//
// The following rules apply to AddRoute calls:
//
//   - Matching the semantics of [Handlers], a handler for [http.MethodGet] will also match [http.MethodHead]
//     unless a handler for [http.MethodHead] is installed for the same path template.
//     Said handler can be set to nil to explicitly forbid HEAD requests.
//   - The path template must consist of a number of path elements separated by slashes, with a leading slash (e.g. "/v1/objects/:id").
//     Path elements must consist of alphanumeric ASCII characters, and otherwise use symbols only as detailed below.
//     [Element], [Variable] and [CatchAllVariable] matchers using non-alphanumeric or non-ASCII characters cannot be represented
//     and must use the tree-shaped constructor API instead.
//
// AddRoute accepts the following syntaxes for path elements:
//
//   - Path elements without symbols will be translated into [Element] matchers with the path element as the value.
//   - Path elements of the form ":name" will be translated into [Variable] matchers using the specified name.
//   - Path elements of the form "*name" will be translated into [CatchAllVariable] matchers using the specified name.
//   - Variable matchers can have a suffix of the form "?predicate" to restrict matches to values matching the predicate (as registered with AddPredicate).
//     Those path elements will be translated into [VariableIf] and [CatchAllVariableIf] matchers.
//   - Path templates that have a trailing slash or no trailing slash will match this property of the actual request path exactly.
//     To match requests with trailing slash or no trailing slash using the same path template, write the trailing slash in parentheses, as "(/)".
//
// For example, a path template of "/objects/*path?isValidPath/comments/:id(/)" is equivalent to
//
//	Element("objects", CatchAllVariableIf("path", isValidPath,
//		Element("comments", Variable("id",
//			Here(Handlers(...)),
//		)),
//	))
type Builder interface {
	// AddRoute adds an endpoint handler to the routing table.
	// Refer to the type docstring for details.
	AddRoute(method, pathTemplate string, handler http.HandlerFunc)
	// AddPredicate registers a predicate that can be referenced in path templates of AddRoute().
	AddPredicate(name string, predicate func(string) bool)
}

type builder struct {
	Predicates map[string]func(string) bool
	Node       builderNode
}

// AddRoute implements the [Builder] interface.
func (b *builder) AddRoute(method, pathTemplate string, handler http.HandlerFunc) {
	// compile arguments into a sequence of path nodes leading to a handler node
	node, err := b.buildRoute(method, pathTemplate, handler)
	if err != nil {
		panic(fmt.Sprintf("could not add route %q: %s", method+" "+pathTemplate, err.Error()))
	}

	// merge new route into the existing routing table, spawning choice nodes as needed
	b.Node = b.Node.TryMerge(node).UnwrapOrPanic("choiceNode.TryMerge() should never fail")
}

func (b *builder) buildRoute(method, pathTemplate string, handler http.HandlerFunc) (node builderNode, err error) {
	if handler == nil && method != http.MethodHead {
		return nil, errors.New("handler must not be nil")
	}

	// consume leading slash
	var ok bool
	pathTemplate, ok = strings.CutPrefix(pathTemplate, "/")
	if !ok {
		return nil, errors.New("path template does not have a leading slash")
	}

	// consume required trailing slash
	hn := handlerNode{
		Handlers: map[string]http.HandlerFunc{method: handler},
	}
	pathTemplate, ok = strings.CutSuffix(pathTemplate, "(/)")
	if ok {
		hn.Attributes.IsHere = true
	}

	// consume optional trailing slash if there was no required trailing slash
	node = hn
	if !ok {
		pathTemplate, ok = strings.CutSuffix(pathTemplate, "/")
		if ok {
			node = pathNode{
				Inner:      node,
				Attributes: pathNodeAttributes{Element: "/"},
			}
		}
	}

	// now only the bare path remains in pathTemplate, e.g. `/v1/foo/bar(/)` -> `v1/foo/bar`
	for _, elem := range slices.Backward(strings.Split(pathTemplate, "/")) {
		node, err = b.buildPathNode(elem, node)
		if err != nil {
			return nil, fmt.Errorf("invalid path element %q in path %q: %w", elem, pathTemplate, err)
		}
	}
	return node, nil
}

func (b *builder) buildPathNode(template string, inner builderNode) (_ builderNode, err error) {
	// try to parse a variable declaration
	var (
		isVariable = false
		isCatchAll = false
		ok         bool
	)
	if template, ok = strings.CutPrefix(template, ":"); ok {
		isVariable = true
	} else if template, ok = strings.CutPrefix(template, ":"); ok {
		isVariable = true
		isCatchAll = true
	}
	if isVariable {
		var (
			attrs        = pathNodeAttributes{IsCatchAll: isCatchAll}
			hasPredicate bool
		)
		attrs.VariableName, attrs.PredicateName, hasPredicate = strings.Cut(template, "?")
		if !isASCIIAlphanumeric(attrs.VariableName) {
			return nil, fmt.Errorf("variable name %q contains invalid characters", attrs.VariableName)
		}
		if hasPredicate {
			if attrs.PredicateName == "" {
				return nil, errors.New("predicate name may not be empty")
			}
			if !isASCIIAlphanumeric(attrs.PredicateName) {
				return nil, fmt.Errorf("predicate name %q contains invalid characters", attrs.PredicateName)
			}
			if _, ok := b.Predicates[attrs.PredicateName]; !ok {
				return nil, fmt.Errorf("predicate %q is not defined", attrs.PredicateName)
			}
		}
		return pathNode{inner, attrs}, nil
	}

	// try to parse an element declaration
	if !isASCIIAlphanumeric(template) {
		return nil, fmt.Errorf("element %q contains invalid characters", template)
	}
	attrs := pathNodeAttributes{Element: template}
	return pathNode{inner, attrs}, nil
}

func isASCIIAlphanumeric(in string) bool {
	for _, r := range in {
		if r > unicode.MaxASCII {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// AddPredicate implements the [Builder] interface.
func (b *builder) AddPredicate(name string, predicate func(string) bool) {
	_, ok := b.Predicates[name]
	if ok {
		panic(fmt.Sprintf("AddPredicate() called multiple times for name = %q", name))
	}
	b.Predicates[name] = predicate
}

// builderNode is an intermediate representation for the routing table that is being assembled by type builder.
type builderNode interface {
	// Merge the given node into this one if the head nodes are compatible,
	// returning the merged node on success or None on structural mismatch.
	TryMerge(route builderNode) Option[builderNode]
	// Convert this node into a real Matcher.
	Into(predicates map[string]func(string) bool) Matcher
}

type choiceNode []builderNode

// TryMerge implements the [node] interface.
func (n choiceNode) TryMerge(other builderNode) Option[builderNode] {
	for idx, node := range n {
		if merged, ok := node.TryMerge(other).Unpack(); ok {
			n[idx] = merged
			return Some[builderNode](n)
		}
	}
	return Some[builderNode](append(n, other))
}

// Into implements the [node] interface.
func (n choiceNode) Into(predicates map[string]func(string) bool) Matcher {
	if len(n) == 1 {
		return n[0].Into(predicates)
	}

	matchers := make([]Matcher, len(n))
	for idx, node := range n {
		matchers[idx] = node.Into(predicates)
	}
	return Choice(matchers...)
}

type pathNode struct {
	Inner      builderNode
	Attributes pathNodeAttributes
}

type pathNodeAttributes struct {
	// Element()
	Element string
	// Variable(), CatchAllVariable(), ...If()
	VariableName  string
	IsCatchAll    bool
	PredicateName string
}

// TryMerge implements the [node] interface.
func (n pathNode) TryMerge(other builderNode) Option[builderNode] {
	rhs, ok := other.(pathNode)
	if !ok || n.Attributes != rhs.Attributes {
		return None[builderNode]()
	}

	// if both sides have identical path nodes, we will always force a merge,
	// possibly by spawning a choice node inside
	merged, ok := n.Inner.TryMerge(rhs.Inner).Unpack()
	if !ok {
		merged = choiceNode{n.Inner, rhs.Inner}
	}
	return Some[builderNode](pathNode{merged, n.Attributes})
}

// Into implements the [node] interface.
func (n pathNode) Into(predicates map[string]func(string) bool) Matcher {
	var (
		inner = n.Inner.Into(predicates)
		attrs = n.Attributes
	)
	switch {
	case attrs.Element != "":
		return Element(attrs.Element, inner)
	case attrs.IsCatchAll:
		return CatchAllVariableIf(attrs.VariableName, predicates[attrs.PredicateName], inner)
	default:
		return VariableIf(attrs.VariableName, predicates[attrs.PredicateName], inner)
	}
}

type handlerNode struct {
	Handlers   map[string]http.HandlerFunc
	Attributes handlerNodeAttributes
}

type handlerNodeAttributes struct {
	IsHere bool
}

// TryMerge implements the [node] interface.
func (n handlerNode) TryMerge(other builderNode) Option[builderNode] {
	if rhs, ok := other.(handlerNode); ok && n.Attributes == rhs.Attributes {
		maps.Copy(n.Handlers, rhs.Handlers)
		return Some[builderNode](n)
	} else {
		return None[builderNode]()
	}
}

// Into implements the [node] interface.
func (n handlerNode) Into(predicates map[string]func(string) bool) Matcher {
	result := Handlers(n.Handlers)
	if n.Attributes.IsHere {
		result = Here(result)
	}
	return result
}
