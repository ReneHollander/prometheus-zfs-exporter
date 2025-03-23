# prometheus-zfs-exporter

Exporter to send ZFS metrics to Prometheus.

Directly calls ZFS ioctl and reads kstats exported by ZFS to `/proc` without any dependencies (no native libraries etc).

ioctl and nvlist parsing code largely based on https://github.com/lorenz/go-zfs.

## Installation

Assuming you use NixOS:

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    docker-zfs-plugin.url = "github:ReneHollander/prometheus-zfs-exporter";

    docker-zfs-plugin.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, ... }@inputs: {
    nixosConfigurations."hostname" = nixpkgs.lib.nixosSystem rec {
      system = "x86_64-linux";
      modules = [
        inputs.prometheus-zfs-exporter.nixosModule
      ];
    };
  };
}
```

Configure the service:

```nix
{
  services.prometheus-zfs-exporter = {
    enable = true;
  };
}
```

Metrics exported at http://127.0.0.1:9901/metrics.

## Development

Run the exporter as a auto restarting dev server:

```sh
nix run ".#dev-server"
```

Run curl every second to fetch the metrics for fast feedback during development:

```
watch -n 1 'curl -s -o - "http://127.0.0.1:9901/metrics" | grep -E "^(zfs|zpool)"'
```

Format code:

```sh
git add . && nix fmt
```

Run VM tests:

```sh
git add . && nix flake check -L
```
