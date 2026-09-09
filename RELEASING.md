# Releasing

Releases are cut by pushing a tag. Nothing is automatic before that: pushing
to `main` runs the tests but publishes nothing.

## Cut a release

1. Make sure `main` is green and you are level with the remote.
2. Tag and push:

   ```
   git tag -a v0.2.0 -m "v0.2.0"
   git push origin v0.2.0
   ```

3. The `release` workflow runs GoReleaser, which builds macOS and Linux
   binaries for amd64 and arm64, writes `checksums.txt`, and publishes a
   GitHub release with a changelog grouped from the commit messages.

To see what a release would contain without publishing anything:

```
goreleaser release --snapshot --clean --skip=publish
```

## Versioning

Semver, currently pre-1.0, so breaking changes bump the minor. The commit
types drive the changelog groups: `feat` and `fix` are listed, `chore` and
`ci` are filtered out.

## What a tag changes

With no tags in the repository, `go install ...@latest` resolves to the
newest commit on `main`. Once a tag exists, `@latest` resolves to the newest
**tag** instead, so anything merged after it is not picked up until the next
release. Tag when you want users to get the change.

`flywheel version` reports the tag on a release binary because GoReleaser
stamps it via `-ldflags`. A `go install` build has no stamp and falls back to
the Go toolchain's build info, reporting a pseudo-version and the revision it
was built from. Either way the revision is there, which is what makes a stale
binary diagnosable.
