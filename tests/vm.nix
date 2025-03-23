(import ./lib.nix) {

  name = "prometheus-zfs-exporter";
  nodes = {
    machine =
      {
        self,
        pkgs,
        ...
      }:
      {
        imports = [ self.nixosModules.prometheus-zfs-exporter ];

        virtualisation.graphics = false;

        virtualisation.emptyDiskImages = [ 4096 ];
        networking.hostId = "deadbeef";
        boot.supportedFilesystems = [ "zfs" ];
        environment.systemPackages = [
          pkgs.parted
          pkgs.curl
        ];

        services.prometheus-zfs-exporter = {
          enable = true;
        };
      };
  };

  testScript = builtins.readFile ./test.py;
}
