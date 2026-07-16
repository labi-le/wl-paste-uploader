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
      version = "1.3.2";
      pname = "wl-uploader";
      supportedSystems = [
        "x86_64-linux"
        "aarch64-linux"
      ];

      systemConfigs = {
        x86_64-linux = {
          arch = "linux_amd64";
          hash = "sha256-OS+BPZNSsydfIBszHoUF/pXDylk86CGNVh03oxTpp8c="; # x86_64-linux
        };
        aarch64-linux = {
          arch = "linux_arm64";
          hash = "sha256-l/PmyD9/GhnjDD0HbGs5Dlix5iQ4lP8W/aZRic3eWqY="; # aarch64-linux
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

            provider = mkOption {
              type = types.nullOr (types.enum [ "0x0" "x0" "envs" "catbox" ]);
              default = null;
              example = "catbox";
              description = ''
                Default upload provider. When set, the binary is wrapped to export
                UPLOADER_PROVIDER so `wl-uploader` uses it; still overridable
                per-invocation with --provider. Null keeps the built-in default (0x0).
              '';
            };
          };

          # wl-uploader is a one-shot command bound to a keybind, not a daemon,
          # so the module only wires up the binary and its runtime dependencies.
          config = lib.mkIf cfg.enable {
            environment.systemPackages = [
              (
                if cfg.provider == null then
                  cfg.package
                else
                  pkgs.symlinkJoin {
                    name = "${pname}-${version}-${cfg.provider}";
                    paths = [ cfg.package ];
                    nativeBuildInputs = [ pkgs.makeWrapper ];
                    postBuild = ''
                      wrapProgram $out/bin/${pname} \
                        --set-default UPLOADER_PROVIDER ${cfg.provider}
                    '';
                  }
              )
              pkgs.wl-clipboard # wl-paste / wl-copy
              pkgs.libnotify # notify-send
            ]
            ++ lib.optional cfg.ocr pkgs.tesseract;
          };
        };
    };
}
