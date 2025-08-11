<!--
SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
SPDX-License-Identifier: Apache-2.0
-->

# v1.2.0 (TBD)

Changes:

- Add package jsonmatch.

# v1.1.0 (2025-04-24)

Changes:

- Add `options.Max()` and `options.Min()`.

# v1.0.0 (2025-02-12)

Initial release. The Go version requirement is 1.24.0 because `type Option`
depends on support for `omitzero` in encoding/json for correct marshaling.
