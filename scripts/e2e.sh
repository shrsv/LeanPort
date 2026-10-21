#!/usr/bin/env bash
# End-to-end check for LeanPort: fresh scaffold, tutorial, failure path, rich types.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/.go/bin:${HOME}/.elan/bin:${PATH}"
export CGO_ENABLED=1

WORK="$(mktemp -d "${TMPDIR:-/tmp}/leanport-e2e.XXXXXX")"
trap 'rm -rf "$WORK"' EXIT

say() { printf '\n== %s\n' "$*"; }
fail() { printf 'FAIL: %s\n' "$*" >&2; exit 1; }

LP="$WORK/leanport"

say "build leanport"
(cd "$ROOT" && go build -o "$LP" ./cmd/leanport)
"$LP" version

say "fresh scaffold: init -> new -> check -> build -> run"
mkdir -p "$WORK/fresh"
cd "$WORK/fresh"
printf 'module example.com/fresh\n\ngo 1.24\n' > go.mod
"$LP" init --env local --ci none --name fresh >/dev/null
"$LP" new Decide >/dev/null
"$LP" check
"$LP" build
cat > main.go <<'EOF'
package main

import (
	"fmt"

	"example.com/fresh/internal/verified/decide"
)

func main() { fmt.Println(decide.Decide(true, 1)) }
EOF
[ "$(go run .)" = "true" ] || fail "fresh scaffold returned the wrong value"

say "tutorial: checked policy and runtime-free archive"
cd "$ROOT/tutorial/permctl"
"$LP" check
"$LP" build
go test -count=1 ./...
[ "$(go run ./cmd/permctl verify)" = "verified 39 checks" ] || fail "permctl verify disagreed"
[ "$(go run ./cmd/permctl decide --auth --owner --action update)" = "allow" ] || fail "owner/update should allow"
if go run ./cmd/permctl decide --auth --owner --action viewAudit >/dev/null 2>&1; then
	fail "owner/viewAudit should exit non-zero"
fi
archive="internal/verified/authz/native/linux_amd64/libAuthz.a"
[ -f "$archive" ] || fail "missing archive"
size=$(wc -c <"$archive")
[ "$size" -lt 1048576 ] || fail "runtime-free archive is unexpectedly large: $size bytes"
go build -o "$WORK/permctl" ./cmd/permctl
if ldd "$WORK/permctl" | grep -qi lean; then
	fail "permctl links a Lean shared library"
fi
printf '   archive: %s bytes, binary: %s bytes\n' "$size" "$(wc -c <"$WORK/permctl")"

say "check rejects an injected hole"
proof="lean/LeanPort/Authz/Proofs.lean"
cp "$proof" "$WORK/Proofs.lean.bak"
printf '\ntheorem injected_hole : True := by\n  sorry\n' >>"$proof"
if "$LP" check >/dev/null 2>"$WORK/sorry.err"; then
	cp "$WORK/Proofs.lean.bak" "$proof"
	fail "check accepted an unfinished proof"
fi
grep -q "Proofs.lean" "$WORK/sorry.err" || fail "hole error did not name the file"
cp "$WORK/Proofs.lean.bak" "$proof"
"$LP" check >/dev/null
echo "   rejected as expected"

say "multiple modules: add a second module and link both"
cp -a "$ROOT/tutorial/permctl" "$WORK/multi"
cd "$WORK/multi"
"$LP" new --package quota --go-name AllowQuota --export leanport_quota_allow Quota >/dev/null
"$LP" check
"$LP" build
mkdir -p cmd/quotademo
cat > cmd/quotademo/main.go <<'EOF'
package main

import (
	"fmt"

	"example.com/permctl/internal/verified/authz"
	"example.com/permctl/internal/verified/quota"
)

func main() {
	fmt.Println(authz.Authorize(true, false, true, authz.Update))
	fmt.Println(quota.AllowQuota(true, 7))
	fmt.Println(quota.AllowQuota(false, 7))
}
EOF
out="$(go run ./cmd/quotademo)"
[ "$out" = $'true\ntrue\nfalse' ] || fail "two-module program returned: $out"

say "rich types: strings and byte arrays through the runtime-linked archive"
cd "$ROOT/examples/rich-types"
"$LP" check
"$LP" build
go test -count=1 ./...
demo="$(go run ./cmd/richdemo)"
grep -q "Hello, LeanPort!" <<<"$demo" || fail "greet output missing"
grep -q "sum(\[1 2 3 250\]) = 256" <<<"$demo" || fail "byte sum output missing"
grep -q 'tropnael' <<<"$demo" || fail "byte reverse output missing"
rich_archive="internal/verified/rich/native/linux_amd64/libRich.a"
[ "$(wc -c <"$rich_archive")" -gt 1048576 ] || fail "linked archive should include the Lean runtime"
printf '%s\n' "$demo" | sed 's/^/   /'

say "docs"
"$LP" docs >/dev/null
[ -f lean/LeanPort/Rich/SPEC.md ] || fail "docs did not write SPEC.md"

printf '\n✓ e2e passed\n'
