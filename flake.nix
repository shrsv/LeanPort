{
  description = "LeanPort — Lean 4 proofs behind an ordinary Go API";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in {
      devShells = forAll (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go
            pkgs.gcc
            pkgs.gnumake
            pkgs.pkg-config
            pkgs.curl
            pkgs.cacert
            pkgs.git
            pkgs.gh
          ];

          shellHook = ''
            # Lean itself is pinned by lean/lean-toolchain and fetched by elan.
            # Keep elan project-local so the shell does not depend on a global install.
            if [ -x "$PWD/.elan/bin/lean" ]; then
              export ELAN_HOME="$PWD/.elan"
              export PATH="$ELAN_HOME/bin:$PATH"
            elif ! command -v lean >/dev/null 2>&1; then
              export ELAN_HOME="$PWD/.elan"
              curl -fsSL https://elan.lean-lang.org/elan-init.sh \
                | sh -s -- -y --default-toolchain none
              export PATH="$ELAN_HOME/bin:$PATH"
            fi
            export CGO_ENABLED=1
          '';
        };
      });

      formatter = forAll (pkgs: pkgs.nixfmt-rfc-style);
    };
}
