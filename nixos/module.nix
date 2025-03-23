{
  config,
  pkgs,
  lib,
  ...
}:

with lib;
let
  cfg = config.services.prometheus-zfs-exporter;
in
{
  options = {
    services.prometheus-zfs-exporter = {
      enable = mkEnableOption "prometheus-zfs-exporter";

      listenAddress = lib.mkOption {
        type = lib.types.str;
        default = "127.0.0.1";
        description = lib.mdDoc ''
          Address to listen on for the HTTP server.
        '';
      };

      port = lib.mkOption {
        type = lib.types.port;
        default = 9901;
        description = lib.mdDoc ''
          Port to listen on for the HTTP server.
        '';
      };
    };
  };

  config = {
    nixpkgs.overlays = [ (import ./overlay.nix) ];

    systemd.services.prometheus-zfs-exporter = mkIf cfg.enable {
      description = "Exporter to send ZFS metrics to Prometheus.";
      after = [
        "network.target"
        "zfs.target"
      ];
      wantedBy = [ "network.target" ];
      serviceConfig = {
        DynamicUser = "false";
        ExecStart = "${pkgs.prometheus-zfs-exporter}/bin/prometheus-zfs-exporter --listen-addr ${cfg.listenAddress}:${builtins.toString cfg.port}";
        Restart = "always";
        RestartSec = "5";
      };
    };
  };
}
