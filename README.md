# @block65/oxlint-tsgolint

Fork of tsgolint 7.0.2002 with additional type-aware rules, used by @block65/oxlint. Block65 internal.

## Use

```yaml
# pnpm-workspace.yaml
overrides:
  oxlint: npm:@block65/oxlint@1.83.0
  oxlint-tsgolint: npm:@block65/oxlint-tsgolint@7.0.200200
```

## Versioning

Our patch is upstream's patch times 100, plus our build number (00-99).
Upstream 7.0.2002 gives `7.0.200200` for build 00 and `7.0.200201` for build
01. Upstream's major and minor are left alone, so the `oxlint-tsgolint` ranges
in consuming repos stay satisfied.
