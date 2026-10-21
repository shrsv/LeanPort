package main

import (
	"bytes"
	"strings"
	"testing"

	"example.com/permctl/internal/verified/authz"
)

func TestMatrixMatchesSpecification(t *testing.T) {
	for _, l := range levels {
		for _, a := range actions {
			got := allowed(l.auth, l.adm, l.own, a.code)
			want := expected[l.name][a.name]
			if got != want {
				t.Errorf("%s/%s: got %v, want %v", l.name, a.name, got, want)
			}
		}
	}
}

func TestVerifyCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"verify"}, &out, &errOut); code != 0 {
		t.Fatalf("verify exited %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "verified") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestDecideExitCodes(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"decide", "--auth", "--owner", "--action", "update"}, &out, &errOut); code != 0 {
		t.Fatalf("owner/update should allow, got %d (%s)", code, out.String())
	}
	if !strings.Contains(out.String(), "allow") {
		t.Fatalf("expected allow, got %q", out.String())
	}
	out.Reset()
	if code := run([]string{"decide", "--auth", "--owner", "--action", "viewAudit"}, &out, &errOut); code != 1 {
		t.Fatalf("owner/viewAudit should deny with exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), "deny") {
		t.Fatalf("expected deny, got %q", out.String())
	}
}

func TestUnknownActionIsUsageError(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"decide", "--auth", "--action", "drop-tables"}, &out, &errOut); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestInvalidWireCodeFailsClosed(t *testing.T) {
	for _, code := range []authz.Action{5, 255} {
		if allowed(true, true, true, code) {
			t.Errorf("invalid action code %d must be denied", code)
		}
	}
}
