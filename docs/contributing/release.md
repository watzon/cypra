# Release Workflow

v0.1 release work is deployment-gated and should only run after CI is green on `main`.

## Server Release

1. Confirm `./bin/agent-ci run --quiet --all` passes locally.
2. Confirm `make lighthouse-baseline`, `make image-size`, and `make test-go-coverage` pass.
3. Update `CHANGELOG.md` with the version, date, and known limitations.
4. Tag the release with `v0.1.0`.
5. Let `.github/workflows/release.yml` build and publish the multi-arch image and GitHub Release.

## Go SDK Release

The Go SDK is versioned separately under `sdk/go`. SDK tags use the `sdk/go/vX.Y.Z` shape. Before tagging, run:

```sh
cd sdk/go && go test ./...
```

## Deployment Verification

After the image is published, verify a fresh deploy with the canonical demo, Go SDK third-machine smoke, export/import round trip, and instance-admin recovery smoke. Those checks are intentionally deployment-phase tasks because they require a real install URL and published image.
