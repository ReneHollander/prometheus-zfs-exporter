{
  description = "prometheus-zfs-exporter";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    systems.url = "github:nix-systems/default";
    treefmt-nix = {
      url = "github:numtide/treefmt-nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      systems,
      nixpkgs,
      treefmt-nix,
      ...
    }:
    let
      eachSystem = f: nixpkgs.lib.genAttrs (import systems) (system: f nixpkgs.legacyPackages.${system});

      treefmtEval = eachSystem (pkgs: treefmt-nix.lib.evalModule pkgs ./treefmt.nix);
    in
    {
      overlays.prometheus-zfs-exporter = import ./nixos/overlay.nix;
      nixosModules.prometheus-zfs-exporter = import ./nixos/module.nix;

      devShells = eachSystem (pkgs: {
        default = pkgs.mkShell (
          with pkgs;
          {
            buildInputs = [
              go
              gopls
              go-tools
              jq
            ];
          }
        );
      });

      apps = eachSystem (pkgs: {
        # nix run ".#dev-server"
        dev-server = {
          type = "app";
          program = toString (
            pkgs.writers.writeBash "dev-server" ''
              ${pkgs.nodemon}/bin/nodemon --watch './**/*.go' --signal SIGTERM --exec '${pkgs.go}/bin/go' run main.go -- $@
            ''
          );
        };
      });

      # Run `nix fmt [FILE_OR_DIR]...` to execute formatters configured in treefmt.nix.
      formatter = eachSystem (pkgs: treefmtEval.${pkgs.system}.config.build.wrapper);

      checks = eachSystem (
        pkgs:
        let
          checkArgs = {
            # reference to nixpkgs for the current system
            pkgs = pkgs;
            # this gives us a reference to our flake but also all flake inputs
            inherit self;
          };
        in
        {
          vm = import ./tests/vm.nix checkArgs;

          # Throws an error if any of the source files are not correctly formatted
          # when you run `nix flake check --print-build-logs`. Useful for CI
          treefmt = treefmtEval.${pkgs.system}.config.build.check self;
        }
      );
    };
}
