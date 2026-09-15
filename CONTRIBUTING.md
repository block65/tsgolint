# Contributing

This fork carries only the Block65 patch set listed in `NOTICE` on top of
upstream tsgolint 7.0.2001. Changes to tsgolint itself belong upstream at
[oxc-project/tsgolint](https://github.com/oxc-project/tsgolint); please do not
open them here. The four added rules are maintained in the
[block65 oxlint plugin](https://github.com/block65/oxlint-plugin) repository
under `native/patches`, which is the source of truth for this branch; the
rebase procedure for the next upstream bump lives there too.

To build from a clean clone:

```sh
git submodule update --init
(cd typescript-go && git am --3way --no-gpg-sign ../patches/*.patch)
mkdir -p internal/collections
find ./typescript-go/internal/collections -type f ! -name '*_test.go' -exec cp {} internal/collections/ \;
go build -ldflags="-s -w" -trimpath -o tsgolint ./cmd/tsgolint
go test ./internal/rules/define_messages_keys ./internal/rules/no_widening_alias ./internal/rules/no_widening_object_keys ./internal/rules/no_widening_return_type
```
