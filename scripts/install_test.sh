#!/bin/sh
# Tests install.sh against a stubbed curl and uname. Nothing touches the network.
#
#   sh scripts/install_test.sh

set -eu

SCRIPT="$(cd "$(dirname "$0")" && pwd)/install.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

FAILED=0
PASSED=0

pass() {
	PASSED=$((PASSED + 1))
	printf 'ok   %s\n' "$1"
}

fail() {
	FAILED=$((FAILED + 1))
	printf 'FAIL %s\n' "$1"
	[ -z "${2:-}" ] || printf '     %s\n' "$2"
}

assert_contains() { # name haystack needle
	case "$2" in
	*"$3"*) pass "$1" ;;
	*) fail "$1" "expected output to contain: $3 (got: $2)" ;;
	esac
}

assert_not_contains() { # name haystack needle
	case "$2" in
	*"$3"*) fail "$1" "expected output not to contain: $3 (got: $2)" ;;
	*) pass "$1" ;;
	esac
}

assert_file() { # name path
	if [ -x "$2" ]; then pass "$1"; else fail "$1" "missing executable $2"; fi
}

assert_no_file() { # name path
	if [ ! -e "$2" ]; then pass "$1"; else fail "$1" "unexpected file $2"; fi
}

hash_file() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	else
		shasum -a 256 "$1" | cut -d ' ' -f 1
	fi
}

# A release fixture for one version. The archive holds a fake rudder-cli that
# prints the version the way the real one does.
make_release() { # version
	version="$1"
	dir="$WORK/release/v$version"
	mkdir -p "$dir" "$WORK/pkg"

	cat >"$WORK/pkg/rudder-cli" <<BIN
#!/bin/sh
echo "rudder-cli version $version"
BIN
	chmod 755 "$WORK/pkg/rudder-cli"

	: >"$dir/rudder-cli_${version}_checksums.txt"
	for name in Darwin_arm64 Darwin_x86_64 Linux_arm64 Linux_x86_64; do
		archive="rudder-cli_${name}.tar.gz"
		tar -czf "$dir/$archive" -C "$WORK/pkg" rudder-cli
		printf '%s  %s\n' "$(hash_file "$dir/$archive")" "$archive" >>"$dir/rudder-cli_${version}_checksums.txt"
	done
}

# Stubs go first on PATH. curl serves files from $FAKE_RELEASE_ROOT and answers
# the latest-release lookup with $FAKE_LATEST_TAG. It logs every URL it is asked for.
make_stubs() {
	STUBS="$WORK/stubs"
	mkdir -p "$STUBS"

	cat >"$STUBS/curl" <<'STUB'
#!/bin/sh
out=""
want_effective=""
url=""
while [ $# -gt 0 ]; do
	case "$1" in
	-o) out="$2"; shift ;;
	-w) want_effective="$2"; shift ;;
	-*) ;;
	*) url="$1" ;;
	esac
	shift
done
printf '%s\n' "$url" >>"$FAKE_CURL_LOG"

if [ -n "$want_effective" ]; then
	printf '%s' "https://github.com/rudderlabs/rudder-iac/releases/tag/$FAKE_LATEST_TAG"
	exit 0
fi

rel="${url#*/v}"
file="$FAKE_RELEASE_ROOT/v$rel"
if [ ! -f "$file" ]; then
	echo "curl: (22) The requested URL returned error: 404" >&2
	exit 22
fi
cp "$file" "$out"
STUB

	cat >"$STUBS/uname" <<'STUB'
#!/bin/sh
case "$1" in
-s) echo "${FAKE_OS:-Linux}" ;;
-m) echo "${FAKE_ARCH:-x86_64}" ;;
*) echo "${FAKE_OS:-Linux}" ;;
esac
STUB
	chmod 755 "$STUBS/curl" "$STUBS/uname"
}

# Callers pass extra environment variables as prefix assignments.
run_install() {
	PATH="$STUBS:$PATH" HOME="$WORK/home" sh "$SCRIPT" 2>&1
}

new_case() {
	rm -rf "$WORK/dest" "$WORK/curl.log"
	: >"$WORK/curl.log"
}

make_stubs
make_release 1.2.3
make_release 1.2.4
export FAKE_LATEST_TAG="v1.2.4"
export FAKE_RELEASE_ROOT="$WORK/release"
export FAKE_CURL_LOG="$WORK/curl.log"
mkdir -p "$WORK/home"

# 1. Latest release resolves from the redirect and installs.
new_case
out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "latest install exits 0" "$out"
assert_file "latest: binary installed" "$WORK/dest/rudder-cli"
assert_contains "latest: prints installed version" "$out" "rudder-cli version 1.2.4"
assert_contains "latest: downloads the Linux x86_64 archive" "$(cat "$WORK/curl.log")" "v1.2.4/rudder-cli_Linux_x86_64.tar.gz"
assert_contains "latest: downloads the checksum file" "$(cat "$WORK/curl.log")" "v1.2.4/rudder-cli_1.2.4_checksums.txt"

# 2. Second run is a no-op and downloads nothing.
: >"$WORK/curl.log"
out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "idempotent run exits 0" "$out"
assert_contains "idempotent: reports already installed" "$out" "already installed"
assert_not_contains "idempotent: downloads no archive" "$(cat "$WORK/curl.log")" ".tar.gz"

# 3. A pinned version, with and without the v prefix, replaces the binary.
: >"$WORK/curl.log"
out="$(RUDDER_CLI_VERSION=v1.2.3 RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "pinned install exits 0" "$out"
assert_contains "pinned: v prefix accepted" "$out" "rudder-cli version 1.2.3"
assert_not_contains "pinned: no latest lookup" "$(cat "$WORK/curl.log")" "releases/latest"

# 4. darwin arm64 picks the Darwin_arm64 archive.
new_case
out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Darwin FAKE_ARCH=arm64 run_install)" || fail "darwin arm64 exits 0" "$out"
assert_contains "darwin arm64: archive name" "$(cat "$WORK/curl.log")" "rudder-cli_Darwin_arm64.tar.gz"

# 5. aarch64 maps to arm64 on Linux.
new_case
out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=aarch64 run_install)" || fail "linux aarch64 exits 0" "$out"
assert_contains "linux aarch64: archive name" "$(cat "$WORK/curl.log")" "rudder-cli_Linux_arm64.tar.gz"

# 6. A checksum mismatch installs nothing and exits non-zero.
new_case
cp -R "$WORK/release" "$WORK/release.bad"
printf 'tampered' >>"$WORK/release.bad/v1.2.4/rudder-cli_Linux_x86_64.tar.gz"
if out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_RELEASE_ROOT="$WORK/release.bad" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)"; then
	fail "checksum mismatch exits non-zero" "$out"
else
	pass "checksum mismatch exits non-zero"
fi
assert_contains "checksum mismatch: says so" "$out" "checksum mismatch"
assert_no_file "checksum mismatch: nothing installed" "$WORK/dest/rudder-cli"

# 7. An unsupported OS and an unsupported arch fail with a message.
new_case
if out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=FreeBSD run_install)"; then
	fail "unsupported OS exits non-zero" "$out"
else
	assert_contains "unsupported OS: message" "$out" "unsupported operating system"
fi
if out="$(RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=riscv64 run_install)"; then
	fail "unsupported arch exits non-zero" "$out"
else
	assert_contains "unsupported arch: message" "$out" "unsupported architecture"
fi

# 8. A missing release fails with a message and installs nothing.
new_case
if out="$(RUDDER_CLI_VERSION=9.9.9 RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)"; then
	fail "missing release exits non-zero" "$out"
else
	assert_contains "missing release: message" "$out" "could not download"
fi
assert_no_file "missing release: nothing installed" "$WORK/dest/rudder-cli"

# 9. The default install dir is $HOME/.local/bin, and a dir off PATH is reported.
new_case
out="$(FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "default dir exits 0" "$out"
assert_file "default dir: installed under HOME/.local/bin" "$WORK/home/.local/bin/rudder-cli"
assert_contains "default dir: PATH warning" "$out" "is not on your PATH"

# 10. No PATH warning when the dir is already on PATH.
new_case
out="$(PATH="$WORK/dest:$PATH" RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "on-PATH run exits 0" "$out"
assert_not_contains "on PATH: no warning" "$out" "is not on your PATH"

# 11. The download base can be overridden.
new_case
out="$(RUDDER_CLI_DOWNLOAD_URL=https://mirror.example.com/rudder RUDDER_CLI_INSTALL_DIR="$WORK/dest" FAKE_OS=Linux FAKE_ARCH=x86_64 run_install)" || fail "mirror run exits 0" "$out"
assert_contains "mirror: uses override base" "$(cat "$WORK/curl.log")" "https://mirror.example.com/rudder/v1.2.4/rudder-cli_Linux_x86_64.tar.gz"

# 12. The script never calls sudo.
if grep -v '^[[:space:]]*#' "$SCRIPT" | grep -q 'sudo'; then
	fail "script does not call sudo"
else
	pass "script does not call sudo"
fi

printf '\n%d passed, %d failed\n' "$PASSED" "$FAILED"
[ "$FAILED" -eq 0 ]
