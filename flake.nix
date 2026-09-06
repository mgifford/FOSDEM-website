{
  description = "fosdem.org Hugo site";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        python = pkgs.python3.withPackages (ps: [ ps.icalendar ]);

        validate-ics = pkgs.writeShellApplication {
          name = "validate-ics";
          runtimeInputs = [ python ];
          text = ''
            python3 ${./scripts/validate-ics.py} "$@"
          '';
        };
      in
      {
        packages.validate-ics = validate-ics;

        apps.validate-ics = {
          type = "app";
          program = "${validate-ics}/bin/validate-ics";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.go
            pkgs.hugo
            pkgs.pagefind
            pkgs.woff2
            validate-ics
          ];
        };
      }
    );
}
