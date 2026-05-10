# Release Workflow

Cypra server releases and Go SDK releases are tagged separately. Run release work only from a green `main` branch.

## Server Release

Server tags use `vX.Y.Z` for stable releases and `vX.Y.Z-rc.N` for prereleases.

1. Confirm `./bin/agent-ci run --quiet --all` passes locally.
2. Confirm `make lighthouse-baseline`, `make image-size`, and `make test-go-coverage` pass if they were not already included in the local CI run.
3. Update `CHANGELOG.md` with the version, date, and known limitations.
4. Dry-run the workflow from GitHub Actions with `workflow_dispatch`, for example tag `v0.1.0-rc.1` and `dry_run=true`.
5. Confirm the dry-run artifacts include the platform archives, `checksums.txt`, and the dry-run OCI image checksum.
6. Push the release tag, for example `git tag v0.1.0-rc.1 && git push origin v0.1.0-rc.1`.
7. Let `.github/workflows/release.yml` publish `ghcr.io/watzon/cypra:<version-without-v>`, plus `latest` only for stable non-prerelease tags.
8. Confirm the GitHub Release contains all binary archives, `checksums.txt`, and `image-digests.txt`.

The release workflow builds Linux and Darwin binaries for `amd64` and `arm64`, publishes a multi-arch GHCR image for `linux/amd64` and `linux/arm64`, and records the published image digest as the image verification artifact. Server git tags include the leading `v`; Docker image tags drop it, so `v0.1.0-rc.1` publishes `ghcr.io/watzon/cypra:0.1.0-rc.1`.

## Go SDK Release

The Go SDK module is `github.com/watzon/cypra/sdk/go`. SDK tags use the `sdk/go/vX.Y.Z` or `sdk/go/vX.Y.Z-rc.N` shape, but consumers install the module with the version only:

```sh
go get github.com/watzon/cypra/sdk/go@v0.1.0-rc.1
```

Before tagging, run:

```sh
cd sdk/go && go test ./...
```

The release workflow validates SDK tags by running the SDK test suite and priming the Go module proxy for `github.com/watzon/cypra/sdk/go@<version>`.

## Smoke Verification

`.github/workflows/smoke.yml` runs on a schedule and by manual dispatch. It no longer depends on a single-use setup-token secret. Instead, it creates disposable Docker Compose environments from the published image, mints a fresh bootstrap token inside that disposable stack, and tears the stack down after the run.

The disposable release smoke executes:

1. Canonical demo against a published-image instance.
2. `cypra export` from that instance.
3. `cypra import` into a second fresh instance.
4. Readiness and version checks on the restored instance.
5. Multi-instance-admin recovery by issuing a host-side recovery invite, redeeming it over HTTP, and verifying the recovered admin appears in `admin list-instance-admins --json`.

The SDK smoke compiles a third-machine Go program against the configured SDK version using `go get github.com/watzon/cypra/sdk/go@<version>`.

On failure, the workflow uploads the disposable stack logs and smoke artifacts, then opens an actionable GitHub issue pointing at the failed run.

You can run the disposable published-image smoke locally after installing dependencies and Docker:

```sh
CYPRA_SMOKE_IMAGE=ghcr.io/watzon/cypra:0.1.0-rc.1 bash scripts/smoke-disposable-release.sh
```
