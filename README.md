# wl-paste-uploader

wl-paste share to 0x0.st and copy result url to clipboard

### Installation

- [Prebuilt binaries](https://github.com/labi-le/wl-paste-uploader/releases)
- Nix flake
  <details> <summary>as profile</summary>

  ```sh
  nix profile install github:labi-le/wl-paste-uploader
  ```
  </details>
  <details>
  <summary>import the module</summary>

  ```nix
  {
    # inputs
    wl-uploader.url = "github:labi-le/wl-paste-uploader";
    # outputs
    overlay-wl-uploader = final: prev: {
      wl-uploader = wl-uploader.packages.${system}.default;
    };
  
    modules = [
      ({ config, pkgs, ... }: { nixpkgs.overlays = [ overlay-wl-uploader ]; })
    ];
  
    # add package
    environment.systemPackages = with pkgs; [
      wl-uploader
    ];
  }
  ```
  </details>

### usage:

```sh
wl-uploader
```

example for sway

```conf
bindsym --to-code Mod4+p exec wl-uploader
```

with proxy

```sh
socks_proxy=socks5://127.0.0.1:1088 wl-uploader
```

you can also use env `HTTPS_PROXY`, `HTTP_PROXY`, `SOCKS_PROXY`, `ALL_PROXY`