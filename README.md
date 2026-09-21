# @block65/oxlint-tsgolint

tsgolint 7.0.2002 implementing the type-aware rules that @block65/oxlint
exposes under `typescript/`. Unreachable without that build, which carries the
rule list and the override pair. Linux x64 and arm64 only.

Internal to Block65, on a public registry only because a pnpm override has to
resolve from one. Not supported for outside use, and the repo takes no issues.

## Versioning

Patch is upstream's patch times 100 plus a build number: upstream 7.0.2002
gives 7.0.200200, then 7.0.200201. Major and minor are upstream's, so
`oxlint-tsgolint` ranges keep resolving.
