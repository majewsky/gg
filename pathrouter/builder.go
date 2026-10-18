// SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
// SPDX-License-Identifier: Apache-2.0

package pathrouter

import "net/http"

// Build offers a Sinatra-style interface for building [Matcher] interfaces.
// The routing table can be defined inside the supplied build function, using the methods on type [Builder].
func Build(build func(Builder)) Matcher {
	panic("TODO")
}

// Builder is used inside [Build] calls to construct a routing table.
//
// The AddHandler method adds an endpoint handler to the routine table:
//
//   - Matching the semantics of [Handlers], a handler for [http.MethodGet] will also match [http.MethodHead]
//     unless a handler for [http.MethodHead] is installed for the same path template.
//     Said handler can be set to nil to explicitly forbid HEAD requests.
//   - The path template must consist of a number of path elements separated by slashes, with a leading slash (e.g. "/v1/objects/:id").
//     Path elements must consist of alphanumeric ASCII characters, and otherwise use symbols only as detailed below.
//     [Element], [Variable] and [CatchAllVariable] matchers using non-alphanumeric or non-ASCII characters cannot be represented
//     and must use the tree-shaped constructor API instead.
//
// AddHandler accepts the following syntaxes for path elements:
//
//   - Path elements without symbols will be translated into [Element] matchers with the path element as the value.
//   - Path elements of the form ":name" will be translated into [Variable] matchers using the specified name.
//   - Path elements of the form "*name" will be translated into [CatchAllVariable] matchers using the specified name.
//
// For example, a path template of "/objects/*path/comments/:id" is equivalent to
//
//	Element("objects", CatchAllVariable("path", Element("comments", Variable("id", http.Handlers(...)))))
type Builder interface {
	AddHandler(method, pathTemplate string, handler http.HandlerFunc)
	// TODO: register predicates for VariableIf, CatchAllVariableIf; syntax: /objects/*path?isObjectPath/...
	// TODO: syntax for Here()
}

type builder struct {
}

type builderRoute struct {
	Path string
}
