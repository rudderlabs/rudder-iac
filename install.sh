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

# Messages go through printf, not echo: the echo of dash and of macOS sh rewrites
# backslash sequences such as \c, which a folder name can contain.
fail() {
  printf 'rudder-cli install: %s\n' "$*" >&2
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
  printf '%s\n' "${url##*/}"
}

# Prints $1 as one shell word that sh, bash and zsh read back as the same text:
# as is when it holds only characters no shell treats specially, otherwise in
# single quotes, with each ' written as '\''. Inside single quotes $, backticks,
# \ and ! are plain characters, so a folder name cannot run as code.
shell_word() {
  case "$1" in
    *[!A-Za-z0-9_./:@%+=,-]*) printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")" ;;
    *) printf '%s' "$1" ;;
  esac
}

# Removes the temp directory and, if set, the partial copy in the install directory.
cleanup() {
  rm -rf "$tmp"
  if [ -n "$partial" ]; then rm -f "$partial"; fi
}

main() {
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
  # A shell running under Rosetta reports x86_64 on an Apple silicon Mac, which
  # runs the arm64 build natively.
  if [ "$os" = "Darwin" ] && [ "$arch" = "x86_64" ] &&
    [ "$(sysctl -in hw.optional.arm64 2>/dev/null || true)" = "1" ]; then
    arch="arm64"
  fi

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

  printf '%s\n' "Installing $BIN $tag ($os $arch) into $dir"
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
  # mktemp gives each run its own copy, so two installs into one directory do not
  # write the same file. The EXIT trap removes the copy if a step fails or the
  # script is interrupted.
  partial=$(mktemp "$dir/.$BIN.XXXXXX" 2>/dev/null) ||
    fail "could not write to $dir; set RUDDER_CLI_INSTALL_DIR to a directory you can write to"
  { cp "$tmp/$BIN" "$partial" && chmod 755 "$partial" && mv -f "$partial" "$dir/$BIN"; } 2>/dev/null ||
    fail "could not write $dir/$BIN; set RUDDER_CLI_INSTALL_DIR to a directory you can write to"
  partial=""

  # rudder-cli prints startup errors, such as a missing HOME, on stdout.
  installed=$("$dir/$BIN" --version) || fail "installed $dir/$BIN but it failed to run${installed:+: $installed}"
  printf '%s\n' "Installed $installed at $dir/$BIN"

  # Which binary a plain 'rudder-cli' starts in this shell. Shells print it as
  # dir//rudder-cli for a PATH entry with a trailing slash, and as a relative or
  # dir/./ path for a relative entry such as ./bin, so resolve its directory the
  # same way as $dir above before comparing.
  found=$(command -v "$BIN" || true)
  other_dir=""
  if [ -n "$found" ]; then
    other_dir=$(CDPATH='' cd -- "$(dirname -- "$found")" 2>/dev/null && pwd) || other_dir=""
  fi
  other=""
  case "$other_dir" in
    "" | "$dir") ;;
    *)
      # </dev/null keeps the other copy from reading the rest of a piped script.
      other_version=$("$other_dir/$BIN" --version </dev/null 2>/dev/null) || other_version=""
      other="$other_dir/$BIN${other_version:+ ($other_version)}"
      ;;
  esac

  case "${SHELL:-}" in
    */zsh) profile=".zshrc" ;;
    */bash)
      profile=".bashrc"
      # Terminal on macOS starts bash as a login shell, which reads the first of
      # these files that exists and never reads .bashrc on its own.
      if [ "$os" = "Darwin" ]; then
        profile=".bash_profile"
        for f in .bash_profile .bash_login .profile; do
          if [ -f "${HOME:-}/$f" ]; then
            profile=$f
            break
          fi
        done
      fi
      ;;
    *) profile=".profile" ;;
  esac
  # The user pastes these lines into a shell, so the folder goes in as a shell word.
  qdir=$(shell_word "$dir")
  if [ "$qdir" = "$dir" ]; then
    path_line="export PATH=\"$dir:\$PATH\""
  else
    path_line="export PATH=$qdir:\"\$PATH\""
  fi
  run_line=$(shell_word "$dir/$BIN")

  case ":$PATH:" in
    *":$dir:"* | *":$dir/:"*)
      if [ -n "$other" ]; then
        echo
        printf '%s\n' "Warning: '$BIN' runs $other, not $dir/$BIN, because $other_dir comes before $dir on your PATH."
        printf '%s\n' "To put the new one first, add this line to the end of ~/$profile, then open a new terminal:"
        printf '  %s\n' "$path_line"
        printf 'To use it in this terminal, run the same line, or run %s directly.\n' "$run_line"
      fi
      ;;
    *)
      echo
      printf '%s\n' "$dir is not on your PATH. Add this line to the end of ~/$profile, then open a new terminal:"
      printf '  %s\n' "$path_line"
      if [ -n "$other" ]; then
        printf "Until then, '%s' runs %s. To use the new one in this terminal, run the same line, or run %s directly.\n" "$BIN" "$other" "$run_line"
      fi
      ;;
  esac
}

# Everything that downloads or installs runs inside main, which this last line
# calls. If curl | sh delivers only part of the script, main never runs.
main "$@"
