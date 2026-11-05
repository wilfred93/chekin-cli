#!/bin/sh
# Builds the release archives and checksums.txt into dist/.
# Usage: scripts/build-release.sh v1.2.3
# The release workflow runs this on every v* tag; run it locally to try a release
# without publishing anything.
set -eu

version=${1:?usage: scripts/build-release.sh VERSION (e.g. v1.2.3)}
case $version in v[0-9]*) ;; *) echo "version must look like v1.2.3, got $version" >&2; exit 2 ;; esac

cd "$(dirname "$0")/.."
rm -rf dist
mkdir -p dist

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
	goos=${target%/*}
	goarch=${target#*/}
	name=chekin_${goos}_${goarch}
	bin=chekin
	[ "$goos" = windows ] && bin=chekin.exe

	work=dist/$name
	mkdir -p "$work"
	CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath \
		-ldflags "-s -w -X github.com/wilfred93/chekin-cli/internal/cli.Version=$version" \
		-o "$work/$bin" ./cmd/chekin
	cp README.md LICENSE "$work/"

	if [ "$goos" = windows ]; then
		(cd "$work" && zip -q "../$name.zip" "$bin" README.md LICENSE)
	else
		tar -czf "dist/$name.tar.gz" -C "$work" "$bin" README.md LICENSE
	fi
	rm -rf "$work"
done

cd dist
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum chekin_* > checksums.txt
else
	shasum -a 256 chekin_* > checksums.txt
fi
cat checksums.txt
