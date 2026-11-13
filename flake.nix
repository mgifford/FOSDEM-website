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

        check-links = pkgs.writeShellApplication {
          name = "check-links";
          runtimeInputs = [ pkgs.python3 ];
          text = ''
            python3 ${./scripts/check-links.py} "$@"
          '';
        };

        fetch-pretalx = pkgs.writeShellApplication {
          name = "fetch-pretalx";
          runtimeInputs = [ pkgs.python3 ];
          text = ''
            python3 ${./scripts/fetch-pretalx.py} "$@"
          '';
        };
      in
      {
        packages.validate-ics = validate-ics;
        packages.check-links = check-links;
        packages.fetch-pretalx = fetch-pretalx;

        apps.validate-ics = {
          type = "app";
          program = "${validate-ics}/bin/validate-ics";
        };

        apps.check-links = {
          type = "app";
          program = "${check-links}/bin/check-links";
        };

        apps.fetch-pretalx = {
          type = "app";
          program = "${fetch-pretalx}/bin/fetch-pretalx";
        };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            pkgs.git
            pkgs.hugo
            pkgs.just
            pkgs.nodejs
            pkgs.openssh_gssapi
            pkgs.rsync
            pkgs.woff2
            check-links
            fetch-pretalx
            validate-ics
          ];
        };
      }
    );
}
