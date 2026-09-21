# @block65/oxlint-tsgolint

tsgolint with the four type-aware rules that @block65/oxlint exposes under
`typescript/`. The rules are only reachable through @block65/oxlint, whose
README describes them and declares the oldest build of this package they work
with. Linux x64 and arm64 only, the platforms Block65 runs. The package
version names the upstream release it is built on, as described below.

## Versioning

The patch is upstream's patch times 100 plus a build number. Upstream 7.0.2002
gives 7.0.200200, then 7.0.200201. The major and minor are upstream's, so
`oxlint-tsgolint` ranges keep resolving.
