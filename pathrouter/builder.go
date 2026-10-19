// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pathrouter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	. "go.xyrillian.de/gg/option"
)

// Build offers a Sinatra-style interface for building [Matcher] interfaces.
// The routing table can be defined inside the supplied build function, using the methods on type [Builder].
func Build(build func(Builder)) Matcher {
	b := builder{Node: choiceNode{}}
	build(&b)
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

func (b *builder) buildRoute(method, pathTemplate string, handler http.HandlerFunc) (builderNode, error) {
	if handler == nil && method != http.MethodHead {
		return nil, errors.New("handler must not be nil")
	}
	cut, ok := strings.CutPrefix(pathTemplate, "/")
	if !ok {
		return nil, errors.New("path template does not have a leading slash")
	}

	// TODO: phase 1: parse `pathTemplate` into a sequence of path nodes leading to a handler node
	panic("TODO")
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
	if rhs, ok := other.(pathNode); ok && n.Attributes == rhs.Attributes {
		if merged, ok := n.Inner.TryMerge(rhs.Inner).Unpack(); ok {
			return Some[builderNode](pathNode{merged, n.Attributes})
		}
	}
	return None[builderNode]()
}

// TryMerge implements the [node] interface.
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
	IsHere   bool
	Handlers map[string]http.HandlerFunc
}
