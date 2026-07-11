{
  description = "wl-paste share to 0x0.st and copy result url to clipboard";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      version = "1.3.0";
      pname = "wl-uploader";
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      systemConfigs = {
        x86_64-linux = {
          arch = "linux_amd64";
          hash = "sha256-vm2P1aCQgdbZV9eXCuFCnKO8sOGlQs8PyGSb8b1PRJs="; # x86_64-linux
        };
        aarch64-linux = {
          arch = "linux_arm64";
          hash = "sha256-j3Pfvm6ru6lDUq4T1yTanu8Mye05zKjFyLwcIMuz1po="; # aarch64-linux
        };
      };
    in
    flake-utils.lib.eachSystem supportedSystems (
      system:
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
            mainProgram = pname;
            platforms = supportedSystems;
          };
        };
      }
    )
    // {
      nixosModules.default =
        {
          config,
          lib,
          pkgs,
          ...
        }:
        let
          cfg = config.programs.wl-paste-uploader;
          defaultPackage = self.packages.${pkgs.stdenv.hostPlatform.system}.default;
        in
        {
          options.programs.wl-paste-uploader = with lib; {
            enable = mkEnableOption "wl-paste-uploader (wl-paste share to 0x0.st)";

            package = mkOption {
              type = types.package;
              default = defaultPackage;
              description = "The wl-paste-uploader package to use";
            };

            ocr = mkOption {
              type = types.bool;
              default = false;
              description = "Install tesseract to enable OCR (the --ocr flag)";
            };
          };

          # wl-uploader is a one-shot command bound to a keybind, not a daemon,
          # so the module only wires up the binary and its runtime dependencies.
          config = lib.mkIf cfg.enable {
            environment.systemPackages = [
              cfg.package
              pkgs.wl-clipboard # wl-paste / wl-copy
              pkgs.libnotify # notify-send
            ]
            ++ lib.optional cfg.ocr pkgs.tesseract;
          };
        };
    };
}
