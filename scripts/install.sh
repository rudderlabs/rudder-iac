#!/bin/sh
# Installs rudder-cli from a GitHub release.
#
#   curl -fsSL <host>/cli | sh
#
# Environment variables
#   RUDDER_CLI_VERSION      Release to install, with or without a leading "v". Default is the latest release.
#   RUDDER_CLI_INSTALL_DIR  Where to put the binary. Default is $HOME/.local/bin.
#   RUDDER_CLI_DOWNLOAD_URL Base URL of the release assets, laid out as <base>/v<version>/<asset>.
#                           Default is https://github.com/rudderlabs/rudder-iac/releases/download.
#
# The script never uses sudo. It verifies the archive against the release checksum file
# before it installs anything.

set -eu

REPO="rudderlabs/rudder-iac"
BINARY="rudder-cli"
DEFAULT_DOWNLOAD_URL="https://github.com/${REPO}/releases/download"
LATEST_URL="https://github.com/${REPO}/releases/latest"

say() {
	printf '%s\n' "$*"
}

die() {
	printf 'install.sh: %s\n' "$*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "$1 is required but was not found on PATH"
}

detect_os() {
	case "$(uname -s)" in
	Darwin) echo "Darwin" ;;
	Linux) echo "Linux" ;;
	*) die "unsupported operating system: $(uname -s). This script supports macOS and Linux." ;;
	esac
}

detect_arch() {
	os="$1"
	case "$(uname -m)" in
	x86_64 | amd64) arch="x86_64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) die "unsupported architecture: $(uname -m). This script supports amd64 and arm64." ;;
	esac

	# A shell running under Rosetta reports x86_64 on an Apple silicon Mac.
	if [ "$os" = "Darwin" ] && [ "$arch" = "x86_64" ] &&
		[ "$(sysctl -in hw.optional.arm64 2>/dev/null || true)" = "1" ]; then
		arch="arm64"
	fi

	echo "$arch"
}

# Prints the version without a leading "v".
resolve_version() {
	if [ -n "${RUDDER_CLI_VERSION:-}" ]; then
		echo "${RUDDER_CLI_VERSION#v}"
		return
	fi

	# The latest-release URL redirects to /releases/tag/<tag>. Reading the redirect
	# avoids the GitHub API and its unauthenticated rate limit.
	final_url="$(curl -fsSL -I -o /dev/null -w '%{url_effective}' "$LATEST_URL")" ||
		die "could not resolve the latest release from $LATEST_URL. Set RUDDER_CLI_VERSION to skip the lookup."

	case "$final_url" in
	*/releases/tag/v*) echo "${final_url##*/releases/tag/v}" ;;
	*) die "unexpected redirect while resolving the latest release: $final_url" ;;
	esac
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		die "sha256sum or shasum is required to verify the download"
	fi
}

installed_version() {
	# Output is "rudder-cli version <version>".
	"$1" --version 2>/dev/null | awk '{print $NF}' || true
}

main() {
	need curl
	need tar
	need uname

	os="$(detect_os)"
	arch="$(detect_arch "$os")"
	version="$(resolve_version)"
	[ -n "$version" ] || die "could not determine the version to install"

	install_dir="${RUDDER_CLI_INSTALL_DIR:-$HOME/.local/bin}"
	dest="$install_dir/$BINARY"

	if [ -x "$dest" ] && [ "$(installed_version "$dest")" = "$version" ]; then
		say "$BINARY $version is already installed at $dest"
		warn_if_not_on_path "$install_dir"
		return
	fi

	base="${RUDDER_CLI_DOWNLOAD_URL:-$DEFAULT_DOWNLOAD_URL}"
	base="${base%/}/v${version}"
	archive="${BINARY}_${os}_${arch}.tar.gz"
	checksums="${BINARY}_${version}_checksums.txt"

	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT INT TERM

	say "Installing $BINARY $version for $os/$arch"

	curl -fsSL -o "$tmp/$archive" "$base/$archive" ||
		die "could not download $base/$archive. Check that version $version exists and has a $os/$arch build."
	curl -fsSL -o "$tmp/$checksums" "$base/$checksums" ||
		die "could not download $base/$checksums"

	expected="$(awk -v name="$archive" '$2 == name {print $1}' "$tmp/$checksums")"
	[ -n "$expected" ] || die "$archive is not listed in $checksums"

	actual="$(sha256_of "$tmp/$archive")"
	if [ "$expected" != "$actual" ]; then
		die "checksum mismatch for $archive: expected $expected, got $actual. Nothing was installed."
	fi

	tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY" || die "$archive does not contain $BINARY"

	mkdir -p "$install_dir" 2>/dev/null ||
		die "cannot create $install_dir. Set RUDDER_CLI_INSTALL_DIR to a directory you can write to."
	[ -w "$install_dir" ] ||
		die "$install_dir is not writable. Set RUDDER_CLI_INSTALL_DIR to a directory you can write to."

	# Copy then rename, so a running binary is replaced atomically and a failed
	# copy never leaves a half-written file at the destination.
	cp "$tmp/$BINARY" "$dest.new.$$"
	chmod 755 "$dest.new.$$"
	mv -f "$dest.new.$$" "$dest"

	say "Installed $("$dest" --version) at $dest"
	warn_if_not_on_path "$install_dir"
}

warn_if_not_on_path() {
	case ":${PATH:-}:" in
	*":$1:"*) ;;
	*)
		say ""
		say "$1 is not on your PATH. Add it to your shell profile, for example:"
		say "  export PATH=\"$1:\$PATH\""
		;;
	esac
}

main "$@"
