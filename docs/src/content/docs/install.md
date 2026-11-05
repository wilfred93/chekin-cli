---
title: Install and update
description: Install chekin with the one-line installer, on Windows, or with go install; update it with chekin update; uninstall it.
---

`chekin` is a single binary with no runtime dependencies. Releases are built for macOS, Linux and Windows,
on amd64 and arm64.

## macOS and Linux

```sh
curl -fsSL https://wilfred93.github.io/chekin-cli/install.sh | sh
chekin version
```

The script picks the release archive for your OS and CPU and checks it against the release's
`checksums.txt`. Then it installs `chekin` into `~/.local/bin`. If that directory isn't on your `PATH`, the
script prints the line to add to your shell profile.

If the docs site is down, the same script is at
`https://raw.githubusercontent.com/wilfred93/chekin-cli/master/docs/public/install.sh`.

| Variable | Effect |
| --- | --- |
| `CHEKIN_INSTALL_DIR=/usr/local/bin` | Install somewhere else |
| `CHEKIN_VERSION=v0.1.0` | Install a specific release instead of the latest |

```sh
curl -fsSL https://wilfred93.github.io/chekin-cli/install.sh | CHEKIN_VERSION=v0.1.0 sh
```

## Windows

Download `chekin_windows_amd64.zip` (or `chekin_windows_arm64.zip`) from the
[releases page](https://github.com/wilfred93/chekin-cli/releases). Unzip it and put `chekin.exe` in a folder
on your `PATH`.

## With Go

You need [Go](https://go.dev/dl/) 1.26 or newer.

```sh
go install github.com/wilfred93/chekin-cli/cmd/chekin@latest
```

This puts `chekin` in `$(go env GOPATH)/bin` (usually `~/go/bin`). If your shell can't find it, add that
directory to your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

To build from a checkout instead:

```sh
git clone https://github.com/wilfred93/chekin-cli.git
cd chekin-cli
go build -o bin/chekin ./cmd/chekin
```

## Check the version

```sh
chekin version
```

```text
chekin v0.1.0
```

Releases and `go install ...@version` copies print their version. A build from a checkout prints the Go
pseudo-version of its commit (for example `v0.1.1-0.20261104085500-c210519abcde+dirty`), or `dev` when Go
couldn't read it.

The changes in each version are listed in its notes on the
[releases page](https://github.com/wilfred93/chekin-cli/releases).

## Update

```sh
chekin update --check   # only report whether a newer release exists
chekin update           # download, verify and replace; asks first
chekin update --yes     # don't ask (needed when not run from a terminal)
```

`chekin update` downloads the archive for your platform and checks its SHA-256 against the release's
`checksums.txt` before touching anything. Then it replaces the binary with a rename, so a failed update leaves
the old one in place. If the binary is in a directory you can't write to (for example `/usr/local/bin`), run
it with `sudo`.

- A copy installed with `go install` isn't replaced. `chekin update` prints the `go install` command to run instead.
- A build from a checkout is never replaced.
- Pre-releases (tags like `v0.2.0-rc.1`) are skipped, by both `chekin update` and the install script.

Running the install script again also updates to the latest release.

## Uninstall

Delete your saved login, then the binary:

```sh
chekin logout
rm ~/.local/bin/chekin            # or wherever you installed it: command -v chekin
rm -r ~/.config/chekin            # anything else chekin saved
```

Making a release is described in [Contributing](/chekin-cli/contributing/).

Next: [log in](/chekin-cli/log-in/).
