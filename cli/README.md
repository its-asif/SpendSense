# SpendSense CLI

This module builds the `spendsense` command.

## Local build

From repository root:

```bash
make cli-build
./bin/spendsense --help
```

Or from this folder:

```bash
go build -o ../bin/spendsense .
```

## GitHub release (downloadable binaries)

The workflow at [.github/workflows/release-cli.yml](../.github/workflows/release-cli.yml) publishes release assets for Linux, macOS, and Windows.

1. Commit and push your changes.
2. Create and push a version tag in this format:

```bash
git tag spendsense-cli/v0.1.0
git push origin spendsense-cli/v0.1.0
```

3. GitHub Actions builds and uploads binaries to the tag release.

Produced asset examples:

- `spendsense-cli_v0.1.0_linux_amd64.tar.gz`
- `spendsense-cli_v0.1.0_darwin_arm64.tar.gz`
- `spendsense-cli_v0.1.0_windows_amd64.zip`

## End-user install

After download, users place `spendsense` (or `spendsense.exe`) in their `PATH`.

Then they can run:

```bash
spendsense auth login
```