<!--
SPDX-FileCopyrightText: 2025 Stefan Majewsky <majewsky@gmx.net>
SPDX-License-Identifier: Apache-2.0
-->

# gg (Generic Groundwork)

My personal extension of the standard library.

## List of packages

### Foundational generic types

- [columnar](./columnar/): efficient JSON marshaling of lists of objects in a columnar format
- [is](./is/): binary operations that are expressed in a curried style, e.g. `is.LessThan(b)(a) == a < b`, for use with `Option.IsSomeAnd()` etc.
- [option](./option/): an Option type with strong isolation
- [options](./options/): additional functions for type Option

### Addons for net/http

- [assetembed](./assetembed/): HTTP handler for efficiently serving embedded assets using the cache-busting pattern

### Addons for testing

- [jsonmatch](./jsonmatch/): matching of encoded JSON payloads against fixed assertions

## How to contribute

This repository accepts contributions as follows:

- For new methods and functions, feel free to submit a PR right away.
- For entirely new types, or when adding library dependencies, please send a mail or open a GitHub issue first to discuss the design.
- Generally, we dislike library dependencies in this house because every dependency means extra work managing dependency upgrades.

Before sending a patch, please ensure that `make check` does not report any problems, and run `make benchmark` to check the performance impact of your changes.

To contribute to the primary repository at <https://git.xyrillian.de/go-gg>, please use `git format-patch` in the usual manner and send patches to the maintainer's mail address (which can be found in the copyright notice headers on each file).
Alternatively, if you are still using GitHub, you can submit issues and pull requests at the mirror repository <https://github.com/majewsky/gg>.
