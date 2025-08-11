<!--
SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
SPDX-License-Identifier: Apache-2.0
-->

# gg (Generic Generics)

My personal extension of the standard library, mostly containing foundational generic types.

## List of packages

- [jsonmatch](./jsonmatch/): matching of encoded JSON payloads against fixed assertions
- [option](./option/): an Option type with strong isolation
- [options](./options/): additional functions for type Option

## Future developments

I may add additional types (e.g. `Result`, `Either` or `Pair`) if:

- there is a compelling usecase for myself, and
- I find an API that is ergonomic in practice (this is the biggest reason why `Result` might never happen).
