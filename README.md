# @block65/oxlint-tsgolint

A Block65 patched build of [tsgolint](https://github.com/oxc-project/tsgolint)
7.0.2001, the type-aware backend for oxlint. It is not the oxc project. The
source is [block65/tsgolint](https://github.com/block65/tsgolint), branch
`block65/v7.0.2001`, which is the upstream tag plus the patch set described in
`NOTICE`. Upstream's LICENSE and copyright apply unchanged.

## What it adds

Four rules, available under the `typescript/` namespace when paired with the
matching [`@block65/oxlint`](https://github.com/block65/oxc) build:

- `typescript/define-messages-keys`
- `typescript/no-widening-alias`
- `typescript/no-widening-object-keys`
- `typescript/no-widening-return-type`

Everything else is upstream tsgolint 7.0.2001, unchanged.

## Installation

Keep the plain `oxlint` and `oxlint-tsgolint` dependencies in `package.json`
and route them to the patched builds with a pnpm override, so no command or
config names the fork:

```yaml
# pnpm-workspace.yaml
overrides:
  oxlint: npm:@block65/oxlint@1.82.0
  oxlint-tsgolint: npm:@block65/oxlint-tsgolint@7.0.2001
```

The package keeps upstream's layout and the `tsgolint` bin name, so
`oxlint --type-aware` finds it exactly as it finds the upstream package.
Supported platforms: Linux x64 and Linux arm64 only.

## Documentation

Upstream's documentation covers everything but the four rules:
[oxc-project/tsgolint](https://github.com/oxc-project/tsgolint) and the
[oxlint type-aware guide](https://oxc.rs/docs/guide/usage/linter/type-aware.html).
The four rules are documented in the
[block65 oxlint plugin](https://github.com/block65/oxlint-plugin).
