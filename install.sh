#!/bin/sh
# Installs rudder-cli from the GitHub releases of rudderlabs/rudder-iac.
#
#   curl -fsSL https://raw.githubusercontent.com/rudderlabs/rudder-iac/main/install.sh | sh
#
# Environment variables:
#   RUDDER_CLI_VERSION      release to install, such as 0.28.0 or v0.28.0 (default: latest)
#   RUDDER_CLI_INSTALL_DIR  directory to install into (default: $HOME/.local/bin)
#
# The script checks the archive against the release's checksums file and refuses
# to install on a mismatch. It never uses sudo.

set -eu

REPO="rudderlabs/rudder-iac"
BIN="rudder-cli"
RELEASES="https://github.com/$REPO/releases"

fail() {
  echo "rudder-cli install: $*" >&2
  exit 1
}

has() {
  command -v "$1" >/dev/null 2>&1
}

# download URL FILE
download() {
  if has curl; then
    curl -fsL --retry 3 -o "$2" "$1"
  else
    wget -q -O "$2" "$1" 2>/dev/null
  fi
}

# exists URL: succeeds when URL answers with a success status.
exists() {
  if has curl; then
    curl -fsLI -o /dev/null "$1"
  else
    wget -q --spider "$1" 2>/dev/null
  fi
}

# The release-please workflow publishes a release before goreleaser uploads its
# assets, so for a few minutes after a release, the release exists without them.
not_uploaded() {
  fail "release $tag has no asset $1 yet (it may still be uploading); try again in a few minutes"
}

# Prints the tag that releases/latest redirects to, such as v0.28.0.
latest_tag() {
  if has curl; then
    url=$(curl -fsLI -o /dev/null -w '%{url_effective}' "$RELEASES/latest") || return 1
  else
    url=$(wget -S --spider "$RELEASES/latest" 2>&1 | tr -d '\r' | sed -n 's/^ *[Ll]ocation: *\([^ ]*\).*/\1/p' | tail -n 1)
  fi
  echo "${url##*/}"
}

has curl || has wget || fail "curl or wget is required"

# The archive names follow name_template in cli/.goreleaser.yaml.
os=$(uname -s)
case "$os" in
  Darwin | Linux) ;;
  MINGW* | MSYS* | CYGWIN* | Windows_NT)
    fail "this script does not support Windows; download the .zip for your CPU from $RELEASES/latest"
    ;;
  *) fail "unsupported operating system: $os" ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64 | amd64) arch="x86_64" ;;
  arm64 | aarch64) arch="arm64" ;;
  i386 | i486 | i586 | i686) arch="i386" ;;
  *) fail "unsupported CPU architecture: $arch" ;;
esac

version="${RUDDER_CLI_VERSION:-latest}"
if [ "$version" = "latest" ]; then
  tag=$(latest_tag) || tag=""
  case "$tag" in
    v[0-9]*) ;;
    *) fail "could not resolve the latest release from $RELEASES/latest" ;;
  esac
else
  tag="v${version#v}"
fi

archive="${BIN}_${os}_${arch}.tar.gz"
checksums="${BIN}_${tag#v}_checksums.txt"
dir="${RUDDER_CLI_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is not set; set RUDDER_CLI_INSTALL_DIR to the directory to install into"
  dir="$HOME/.local/bin"
fi

tmp=$(mktemp -d) || fail "could not create a temporary directory"
partial=""
cleanup() {
  rm -rf "$tmp"
  if [ -n "$partial" ]; then rm -f "$partial"; fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

# Check the release before touching the install directory, so a version that
# does not exist leaves nothing behind.
download "$RELEASES/download/$tag/$checksums" "$tmp/$checksums" || {
  exists "$RELEASES/tag/$tag" && not_uploaded "$checksums"
  fail "could not download $RELEASES/download/$tag/$checksums; check that release $tag exists at $RELEASES"
}

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/$checksums")
[ -n "$want" ] || fail "release $tag has no $archive for $os $arch; see $RELEASES/tag/$tag"

{ mkdir -p "$dir" && [ -w "$dir" ]; } 2>/dev/null ||
  fail "cannot write to $dir; set RUDDER_CLI_INSTALL_DIR to a directory you can write to"
# An absolute path, without a trailing slash, for the PATH check and the PATH line below.
dir=$(CDPATH='' cd -- "$dir" && pwd) ||
  fail "cannot write to $dir; set RUDDER_CLI_INSTALL_DIR to a directory you can write to"

echo "Installing $BIN $tag ($os $arch) into $dir"
download "$RELEASES/download/$tag/$archive" "$tmp/$archive" || {
  exists "$RELEASES/download/$tag/$archive" || not_uploaded "$archive"
  fail "could not download $RELEASES/download/$tag/$archive"
}

if has sha256sum; then
  got=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
elif has shasum; then
  got=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
else
  fail "sha256sum or shasum is required to verify the download"
fi
[ "$got" = "$want" ] || fail "checksum mismatch for $archive (expected $want, got $got); not installing"

tar -xzf "$tmp/$archive" -C "$tmp" "$BIN" 2>/dev/null || fail "could not extract $BIN from $archive"

# Copy next to the target first so the final rename never leaves a partial binary.
# The EXIT trap removes the copy if a step fails or the script is interrupted.
partial="$dir/.$BIN.tmp"
{ cp "$tmp/$BIN" "$partial" && chmod 755 "$partial" && mv -f "$partial" "$dir/$BIN"; } 2>/dev/null ||
  fail "could not write $dir/$BIN; set RUDDER_CLI_INSTALL_DIR to a directory you can write to"
partial=""

# rudder-cli prints startup errors, such as a missing HOME, on stdout.
installed=$("$dir/$BIN" --version) || fail "installed $dir/$BIN but it failed to run${installed:+: $installed}"
echo "Installed $installed at $dir/$BIN"

# Which binary a plain 'rudder-cli' starts in this shell. A PATH entry may end
# in a slash, and then some shells print the binary as dir//rudder-cli.
found=$(command -v "$BIN" || true)
other=""
case "$found" in
  "$dir/$BIN" | "$dir//$BIN") ;;
  /*)
    other_dir=${found%/*}
    other_dir=${other_dir%/}
    # </dev/null keeps the other copy from reading the rest of a piped script.
    other_version=$("$found" --version </dev/null 2>/dev/null) || other_version=""
    other="$other_dir/$BIN${other_version:+ ($other_version)}"
    ;;
esac

case "${SHELL:-}" in
  */zsh) profile=".zshrc" ;;
  */bash) profile=".bashrc" ;;
  *) profile=".profile" ;;
esac
path_line="export PATH=\"$dir:\$PATH\""

case ":$PATH:" in
  *":$dir:"* | *":$dir/:"*)
    if [ -n "$other" ]; then
      echo
      echo "Warning: '$BIN' runs $other, not $dir/$BIN, because $other_dir comes before $dir on your PATH."
      echo "To put the new one first, add this line to the end of ~/$profile, then open a new terminal:"
      echo "  $path_line"
      echo "To use it in this terminal, run the same line, or run $dir/$BIN directly."
    fi
    ;;
  *)
    echo
    echo "$dir is not on your PATH. Add this line to the end of ~/$profile, then open a new terminal:"
    echo "  $path_line"
    if [ -n "$other" ]; then
      echo "Until then, '$BIN' runs $other. To use the new one in this terminal, run the same line, or run $dir/$BIN directly."
    fi
    ;;
esac
