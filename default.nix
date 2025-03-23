{ lib, buildGoModule }:
buildGoModule {
  pname = "prometheus-zfs-exporter";
  version = "1.0.0";

  src = lib.cleanSource ./.;

  vendorHash = "sha256-V2mTkkIRSePf/THSFK0xH0ibxfbB30XvxN5Ypy7OlZw=";
  subPackages = [ "." ];

  meta = with lib; {
    supportedPlatforms = platforms.linux;
  };
}
