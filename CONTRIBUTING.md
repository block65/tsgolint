# Contributing

The fork's changes are the files listed in `NOTICE`, which is maintained by
hand: a fork commit that touches a file not yet listed adds it. Anything
outside that list belongs upstream at
[oxc-project/tsgolint](https://github.com/oxc-project/tsgolint). The four
added rules live in `internal/rules/` here, and
[block65/oxc](https://github.com/block65/oxc) carries their oxlint-side stubs
and describes them.

`main` is a stack of fork commits rebased onto upstream `main`, so it reads as
ahead of upstream and never behind:

```sh
git fetch https://github.com/oxc-project/tsgolint.git main
git rebase FETCH_HEAD
git push --force-with-lease
```

Files the fork deletes (`AGENTS.md`, upstream CI) that upstream has since
edited are resolved with `git rm`.

`main` holds `0.0.0` in `VERSION`. A release is the stack replayed onto the
upstream release tag plus one commit that sets `VERSION`:

```sh
git fetch https://github.com/oxc-project/tsgolint.git main
base=$(git merge-base main FETCH_HEAD)
git fetch https://github.com/oxc-project/tsgolint.git tag v[upstream]
git checkout --detach main
git rebase --onto v[upstream] "$base"
```

The version is upstream's patch times 100 plus a build number, as the README
describes. Tag the version commit `v[version]` and publish a GitHub release
against the tag, which runs `deploy.yml`. Release tags are not on `main`'s
history.

`just init` and `just build` set up and build a clean clone. `just test` also
runs upstream's e2e suite; the fork's own tests are:

```sh
go test ./internal/rules/define_messages_keys ./internal/rules/no_widening_alias ./internal/rules/no_widening_object_keys ./internal/rules/no_widening_return_type
```
