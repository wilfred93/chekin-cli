#!/bin/sh
# Installs the latest chekin release on macOS or Linux:
#
#   curl -fsSL https://wilfred93.github.io/chekin-cli/install.sh | sh
#
# CHEKIN_INSTALL_DIR  where to put the binary (default ~/.local/bin)
# CHEKIN_VERSION      release to install, e.g. v0.2.0 (default: latest)
set -eu

repo=https://github.com/wilfred93/chekin-cli
dir=${CHEKIN_INSTALL_DIR:-$HOME/.local/bin}
version=${CHEKIN_VERSION:-latest}

fail() { echo "install.sh: $*" >&2; exit 1; }

case $(uname -s) in
	Darwin) os=darwin ;;
	Linux) os=linux ;;
	*) fail "unsupported OS $(uname -s); download a build from $repo/releases" ;;
esac
case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	arm64 | aarch64) arch=arm64 ;;
	*) fail "unsupported CPU $(uname -m); download a build from $repo/releases" ;;
esac

if [ "$version" = latest ]; then
	url=$repo/releases/latest/download
else
	url=$repo/releases/download/$version
fi
archive=chekin_${os}_${arch}.tar.gz

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ($version)..." >&2
curl -fsSL -o "$tmp/$archive" "$url/$archive" || fail "download failed: $url/$archive"
curl -fsSL -o "$tmp/checksums.txt" "$url/checksums.txt" || fail "download failed: $url/checksums.txt"

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "$archive is not listed in checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/$archive" | awk '{ print $1 }')
else
	got=$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')
fi
[ "$got" = "$want" ] || fail "checksum mismatch for $archive; not installing"

tar -xzf "$tmp/$archive" -C "$tmp" chekin
mkdir -p "$dir"
# Install through a temp file and rename, so a running chekin isn't clobbered.
cp "$tmp/chekin" "$dir/.chekin.new"
chmod 755 "$dir/.chekin.new"
mv -f "$dir/.chekin.new" "$dir/chekin"

echo "Installed $("$dir/chekin" version) to $dir/chekin" >&2
case ":$PATH:" in
	*":$dir:"*) ;;
	*)
		echo >&2
		echo "$dir is not on your PATH. Add it with:" >&2
		echo "  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.$(basename "${SHELL:-sh}")rc" >&2
		;;
esac
