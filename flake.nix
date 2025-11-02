{
  description = "wl-paste share to 0x0.st and copy result url to clipboard";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      version = "1.1.0";
      pname = "wl-uploader";
      supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

      systemConfigs = {
        x86_64-linux = {
          arch = "linux_amd64";
          hash = "sha256-vswK3SfO/msXEykM3uZWAGX3UtSTlS6ldfoX6sDBNqs="; # x86_64-linux
        };
        aarch64-linux = {
          arch = "linux_armv6";
          hash = "sha256-dKZWP5fBsiKgjQCxSA/IQ8ZpRMtdC/h1soIVFXDifCM="; # aarch64-linux
        };
      };
    in
    flake-utils.lib.eachSystem supportedSystems (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        config = systemConfigs.${system};
      in
      {
        packages.default = pkgs.stdenv.mkDerivation {
          inherit pname version;

          src = pkgs.fetchurl {
            url = "https://github.com/labi-le/wl-paste-uploader/releases/download/v${version}/${pname}_${version}_${config.arch}";
            hash = config.hash;
          };

          dontUnpack = true;

          installPhase = ''
            mkdir -p $out/bin
            cp $src $out/bin/${pname}
            chmod +x $out/bin/${pname}
          '';

          meta = with pkgs.lib; {
            description = "wl-paste share to 0x0.st and copy result url to clipboard";
            homepage = "https://github.com/labi-le/wl-paste-uploader";
            license = licenses.mit;
            platforms = supportedSystems;
          };
        };
      }
    );
}