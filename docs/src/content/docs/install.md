---
title: Install and update
description: Install chekin with go install or build it from source, update it, and check which version you have.
---

`chekin` is a single Go binary with no runtime dependencies. You need [Go](https://go.dev/dl/) 1.26 or newer
to install it.

## Install with `go install`

```sh
go install github.com/wilfred93/chekin-cli/cmd/chekin@latest
```

This puts `chekin` in `$(go env GOPATH)/bin` (usually `~/go/bin`). If your shell can't find it, add that
directory to your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Build from source

```sh
git clone https://github.com/wilfred93/chekin-cli.git
cd chekin-cli
go build -o bin/chekin ./cmd/chekin
./bin/chekin help
```

To stamp a version into the binary, set it at build time:

```sh
go build -ldflags "-X github.com/wilfred93/chekin-cli/internal/cli.Version=v0.1.0" -o bin/chekin ./cmd/chekin
```

## Update

Run the same `go install` command again. `@latest` fetches the newest version.

```sh
go install github.com/wilfred93/chekin-cli/cmd/chekin@latest
```

## Check the version

```sh
chekin version
```

```text
chekin dev
```

Builds that don't set a version at build time, including `go install` and plain `go build`, print `dev`.

## Uninstall

Delete the binary and your saved login:

```sh
chekin logout
rm "$(go env GOPATH)/bin/chekin"
```

Next: [log in](/chekin-cli/log-in/).
