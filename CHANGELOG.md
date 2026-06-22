<!--
SPDX-FileCopyrightText: 2026 Stefan Majewsky <majewsky@gmx.net>
SPDX-License-Identifier: Apache-2.0
-->

# v1.10.2 (TBD)

Changes:

- In assert.Equal output, present string literals with backticks instead of quotes when it makes the output more readable.

# v1.10.1 (2026-06-20)

Changes:

- Fix a panic in assert.ErrEqual when the `expected` argument is an error whose underlying type is a struct type.

# v1.10.0 (2026-06-20)

Changes:

- Add packages assert and testcapture.
- The minimum Go version was increased from 1.24 to 1.26
  because `assert.TestingTB` covers methods added to `testing.TB` in Go 1.26.

# v1.9.1 (2026-06-17)

Changes:

- Fix columnar choking on structs with `json:"-"` fields.

# v1.9.0 (2026-06-04)

Changes:

- jsonmatch: Allow embedding custom Diffable instances within Object and Array.

# v1.8.1 (2026-06-02)

Changes:

- Fix package documentation for columnar missing from pkg.go.dev.

# v1.8.0 (2026-06-02)

Changes:

- Add package columnar.

# v1.7.0 (2026-05-03)

Changes:

- The library must now be imported from the new module path `go.xyrillian.de/gg`.

# v1.6.0 (2026-04-01)

Changes:

- Add `jsonmatch.Irrelevant()`.
- Fix recursion into `jsonmatch.Array` during `DiffAgainst()`.

# v1.5.0 (2025-11-26)

Changes:

- Add package is.

# v1.4.0 (2025-11-18)

Changes:

- Add `jsonmatch.Diff.String()`.

# v1.3.0 (2025-08-15)

Changes:

- Add package assetembed.

# v1.2.0 (2025-08-11)

Changes:

- Add package jsonmatch.

# v1.1.0 (2025-04-24)

Changes:

- Add `options.Max()` and `options.Min()`.

# v1.0.0 (2025-02-12)

Initial release. The Go version requirement is 1.24.0 because `type Option`
depends on support for `omitzero` in encoding/json for correct marshaling.
