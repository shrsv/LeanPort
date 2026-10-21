package app

import (
	"bytes"
	"text/template"
)

func render(tpl string, data any) (string, error) {
	t, err := template.New("tpl").Parse(tpl)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

const agentsTemplate = `# Agent guide

This project uses [LeanPort](https://github.com/shrsv/LeanPort): verified logic
is written in Lean and exposed to Go as an ordinary package.

## The one rule

Edit ` + "`lean/`" + `. Never edit generated output under ` + "`internal/verified/`" + `.
If generated code is wrong, fix the Lean source or ` + "`leanport.toml`" + ` and run
` + "`leanport build`" + `.

## Loop

    leanport check     # type-check proofs; rejects sorry/admit and missing exports
    leanport build     # Lean -> C -> static archive -> generated Go binding
    go test ./...      # verify the Go wiring
    leanport docs      # refresh SPEC.md files

## Where a change belongs

| Change | Edit |
|---|---|
| A policy rule | ` + "`lean/LeanPort/<Module>/Policy.lean`" + ` |
| A guarantee | ` + "`lean/LeanPort/<Module>/Proofs.lean`" + ` |
| What crosses the boundary | ` + "`lean/LeanPort/<Module>/Boundary.lean`" + ` + the ` + "`[[binding]]`" + ` entry |
| Application wiring | Go code outside ` + "`internal/verified/`" + ` |

## Boundary

Exports are either scalar-only (runtime-free archive) or use ` + "`String`" + `/` + "`ByteArray`" + `
(runtime-linked archive). ` + "`leanport build`" + ` chooses automatically. Records,
arrays and ` + "`Option`" + ` are ABI-validated but not generated yet. Boundaries must
be total: invalid input returns the deny/zero result, never panics.

## Failure meanings

- **sorry/admit found** — a proof obligation is open; close the proof.
- **export X not found** — a missing or misspelled ` + "`@[export X]`" + `, or the module did not build.
- **Lean build failed** — the definition or theorem does not type-check.

Never add ` + "`sorry`" + ` to reach a green build. Never hand-edit generated files.
`

const flakeTemplate = `{
  description = "LeanPort project: Lean 4 + Go + C toolchain";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      devShells = forAll (pkgs: {
        default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gcc pkgs.gnumake pkgs.curl pkgs.git pkgs.pkg-config ];
          shellHook = ''
            # Lean is pinned by lean/lean-toolchain and fetched by elan.
            # Keeping elan project-local avoids depending on a system install.
            if ! command -v lean >/dev/null 2>&1; then
              export ELAN_HOME="$PWD/.elan"
              curl -fsSL https://elan.lean-lang.org/elan-init.sh \
                | sh -s -- -y --default-toolchain none
              export PATH="$ELAN_HOME/bin:$PATH"
            fi
          '';
        };
      });
    };
}
`

const ciTemplate = `name: verify

on:
  push:
  pull_request:

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.24"
      - name: Install Lean via elan
        run: |
          curl -fsSL https://elan.lean-lang.org/elan-init.sh | sh -s -- -y
          echo "$HOME/.elan/bin" >> "$GITHUB_PATH"
      - name: Install leanport
        run: |
          go install github.com/shrsv/LeanPort/cmd/leanport@v0.1.1
          echo "$(go env GOPATH)/bin" >> "$GITHUB_PATH"
      - name: Proofs, bindings, Go tests
        run: |
          leanport check
          leanport build
          go test ./...
`

const policyTemplate = `namespace LeanPort.{{.Module}}

/-- A total decision function.

    This scaffold is deliberately small. Replace ` + "`decide`" + ` with your
    specification and rules, and keep it total: every input has a result. -/
def decide (enabled : Bool) (code : UInt8) : Bool :=
  enabled && code != 0

end LeanPort.{{.Module}}
`

const proofsTemplate = `import LeanPort.{{.Module}}.Policy

namespace LeanPort.{{.Module}}

/-- A worked example. Replace it with the guarantees your policy needs.

    ` + "`leanport check`" + ` fails if any theorem does not elaborate or if the
    project contains an unfinished proof. -/
theorem disabled_is_denied (code : UInt8) : decide false code = false := by
  simp [decide]

end LeanPort.{{.Module}}
`

const boundaryTemplate = `import LeanPort.{{.Module}}.Policy

namespace LeanPort.{{.Module}}

/-- The C-facing boundary.

    Keep this total and scalar-only while the module is runtime-free:
    invalid input fails closed (returns 0), never panics. If you move to
    strings/byte arrays/records, set ` + "`runtime = \"linked\"`" + ` on the binding. -/
@[export {{.Export}}]
def {{.GoName}}ForC (enabled : UInt8) (code : UInt8) : UInt8 :=
  if decide (enabled == 1) code then 1 else 0

end LeanPort.{{.Module}}
`

const specTemplate = `# {{.Module}} specification

> Generated by ` + "`leanport docs`" + `. Edit ` + "`Policy.lean`" + ` and ` + "`Proofs.lean`" + `,
> not this file.

## Export

| C symbol | Go | Arguments | Result |
|---|---|---|---|
| ` + "`{{.Export}}`" + ` | ` + "`{{.Pkg}}.{{.GoName}}`" + ` | ` + "`bool, uint8`" + ` | ` + "`bool`" + ` |

## Guarantees

Replace this section with the properties your ` + "`Proofs.lean`" + ` establishes.
`
