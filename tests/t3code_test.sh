#!/usr/bin/env bash

set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)"
test_root="$(mktemp -d "${TMPDIR:-/tmp}/t3code-test.XXXXXX")"
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

fail() {
  printf 'FAIL: %s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local expected="$2"
  if ! grep -Fq -- "$expected" "$file"; then
    printf 'Expected to find %s in %s:\n' "$expected" "$file" >&2
    sed -n '1,200p' "$file" >&2
    exit 1
  fi
}

fake_bin="${test_root}/bin"
fake_home="${test_root}/home"
mkdir -p "$fake_bin" "$fake_home"

cat >"${fake_bin}/systemctl" <<'SYSTEMCTL'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${T3CODE_TEST_COMMANDS}"
if [[ "$*" == *"is-active"* ]]; then
  exit 1
fi
SYSTEMCTL

cat >"${fake_bin}/launchctl" <<'LAUNCHCTL'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${T3CODE_TEST_COMMANDS}"
if [[ "${1:-}" == "print" ]]; then
  exit 1
fi
LAUNCHCTL

cat >"${fake_bin}/npm" <<'NPM'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${T3CODE_TEST_COMMANDS}"
if [[ "${1:-}" == "view" ]]; then
  printf '"0.0.99-nightly.test"\n'
  exit 0
fi
if [[ "${1:-}" == "install" ]]; then
  if [[ "${T3CODE_TEST_NPM_FAIL:-false}" == "true" ]]; then
    exit 42
  fi
  prefix=""
  while (($# > 0)); do
    if [[ "$1" == "--prefix" ]]; then
      prefix="$2"
      break
    fi
    shift
  done
  mkdir -p "${prefix}/node_modules/.bin"
  cat >"${prefix}/node_modules/.bin/t3" <<'T3'
#!/usr/bin/env bash
if [[ "${1:-}" == "--version" ]]; then
  printf '0.0.99-nightly.test\n'
  exit 0
fi
printf '%s\n' "$@" >"${T3CODE_TEST_T3_ARGS}"
T3
  chmod +x "${prefix}/node_modules/.bin/t3"
fi
NPM

cat >"${fake_bin}/loginctl" <<'LOGINCTL'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${T3CODE_TEST_COMMANDS}"
if [[ "$*" == *"show-user"* ]]; then
  printf 'yes\n'
fi
LOGINCTL

chmod +x "${fake_bin}/systemctl" "${fake_bin}/launchctl" "${fake_bin}/npm" "${fake_bin}/loginctl"

export HOME="$fake_home"
export PATH="${fake_bin}:${PATH}"
export T3CODE_TEST_COMMANDS="${test_root}/commands"
export T3CODE_TEST_T3_ARGS="${test_root}/t3-args"
: >"$T3CODE_TEST_COMMANDS"

"${repo_dir}/t3code" help >"${test_root}/help"
assert_contains "${test_root}/help" "start"
assert_contains "${test_root}/help" "update"
assert_contains "${test_root}/help" "restart"
assert_contains "${test_root}/help" "pair"
assert_contains "${test_root}/help" "session"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" install --no-start >"${test_root}/install"
assert_contains "${test_root}/install" "Installed T3 Code 0.0.99-nightly.test."
assert_contains "$T3CODE_TEST_COMMANDS" "t3@0.0.99-nightly.test"
assert_contains "$T3CODE_TEST_COMMANDS" "rebuild"
assert_contains "${fake_home}/.config/systemd/user/t3code.service" 'ExecStart="%h/.local/bin/t3code" _serve'
assert_contains "${fake_home}/.local/share/t3code/package.json" '"node-pty": true'

if T3CODE_TEST_NPM_FAIL=true T3CODE_TEST_OS=Linux "${repo_dir}/t3code" update >"${test_root}/failed-update" 2>&1; then
  fail "update should report an npm installation failure"
fi
assert_contains "${test_root}/failed-update" "Failed to install t3@0.0.99-nightly.test."

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" _serve
assert_contains "$T3CODE_TEST_T3_ARGS" "serve"
assert_contains "$T3CODE_TEST_T3_ARGS" "--host"
assert_contains "$T3CODE_TEST_T3_ARGS" "127.0.0.1"
assert_contains "$T3CODE_TEST_T3_ARGS" "3773"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" pair --ttl 15m --label laptop --json
assert_contains "$T3CODE_TEST_T3_ARGS" "auth"
assert_contains "$T3CODE_TEST_T3_ARGS" "pairing"
assert_contains "$T3CODE_TEST_T3_ARGS" "create"
assert_contains "$T3CODE_TEST_T3_ARGS" "--base-url"
assert_contains "$T3CODE_TEST_T3_ARGS" "http://127.0.0.1:3773"
assert_contains "$T3CODE_TEST_T3_ARGS" "--ttl"
assert_contains "$T3CODE_TEST_T3_ARGS" "15m"
assert_contains "$T3CODE_TEST_T3_ARGS" "--label"
assert_contains "$T3CODE_TEST_T3_ARGS" "laptop"
assert_contains "$T3CODE_TEST_T3_ARGS" "--json"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" pair list --json
assert_contains "$T3CODE_TEST_T3_ARGS" "list"
assert_contains "$T3CODE_TEST_T3_ARGS" "--json"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" pair revoke test-pair-id
assert_contains "$T3CODE_TEST_T3_ARGS" "revoke"
assert_contains "$T3CODE_TEST_T3_ARGS" "test-pair-id"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" session list --json
assert_contains "$T3CODE_TEST_T3_ARGS" "auth"
assert_contains "$T3CODE_TEST_T3_ARGS" "session"
assert_contains "$T3CODE_TEST_T3_ARGS" "list"
assert_contains "$T3CODE_TEST_T3_ARGS" "--json"

T3CODE_TEST_OS=Linux "${repo_dir}/t3code" session revoke test-session-id
assert_contains "$T3CODE_TEST_T3_ARGS" "session"
assert_contains "$T3CODE_TEST_T3_ARGS" "revoke"
assert_contains "$T3CODE_TEST_T3_ARGS" "test-session-id"

T3CODE_TEST_OS=Darwin "${repo_dir}/t3code" start >"${test_root}/mac-start"
assert_contains "${fake_home}/Library/LaunchAgents/dev.devsetup.t3code.plist" "<string>_serve</string>"
assert_contains "$T3CODE_TEST_COMMANDS" "bootstrap gui/"

"${repo_dir}/install-t3code" --dry-run >"${test_root}/dry-run"
assert_contains "${test_root}/dry-run" "mac-mini"
assert_contains "${test_root}/dry-run" "omarchy-desktop"

if T3CODE_TEST_OS=Linux "${repo_dir}/t3code" status >"${test_root}/status"; then
  fail "status should be non-zero for a stopped service"
else
  status_code=$?
fi
[[ "$status_code" == 3 ]] || fail "stopped status should exit 3, got ${status_code}"
assert_contains "${test_root}/status" "State:      stopped (registered)"
assert_contains "${test_root}/status" "Version:    0.0.99-nightly.test"

printf 'PASS: t3code manager tests\n'
