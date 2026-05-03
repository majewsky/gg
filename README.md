<!--
SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
SPDX-License-Identifier: Apache-2.0
-->

# gg (Generic Generics)

My personal extension of the standard library, with foundational generic types and `net/http` addons.

## List of packages

- [assetembed](./assetembed/): HTTP handler for efficiently serving embedded assets using the cache-busting pattern
- [jsonmatch](./jsonmatch/): matching of encoded JSON payloads against fixed assertions
- [is](./is/): binary operations that are expressed in a curried style, e.g. `is.LessThan(b)(a) == a < b`, for use with `Option.IsSomeAnd()` etc.
- [option](./option/): an Option type with strong isolation
- [options](./options/): additional functions for type Option

## Future developments

I may add additional types (e.g. `Result`, `Either` or `Pair`) if:

- there is a compelling usecase for myself, and
- I find an API that is ergonomic in practice (this is the biggest reason why `Result` might never happen).

## How to contribute

This repository accepts contributions as follows:

- For new methods and functions, feel free to submit a PR right away.
- For entirely new types, or when adding library dependencies, please open an issue first to discuss the design.
- Generally, we dislike library dependencies in this house because every dependency means extra work managing dependency upgrades.

Before sending a patch, please ensure that `make check` does not report any problems, and run `make benchmark` to check the performance impact of your changes.

To contribute to the primary repository at <https://git.xyrillian.de/go-gg>, please use `git format-patch` in the usual manner and send patches to the maintainer's mail address (which can be found in the copyright notice headers on each file).
Alternatively, if you are still using GitHub, you can submit issues and pull requests at the mirror repository <https://github.com/majewsky/gg>.
