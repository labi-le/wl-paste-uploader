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

### OCR

With `--ocr` the text is recognized from the clipboard image and copied to the
clipboard instead of being uploaded:

```sh
wl-uploader --ocr
```

Requires [`tesseract`](https://github.com/tesseract-ocr/tesseract) in `PATH`.
Pick the recognition language(s) with `OCR_LANG` (defaults to tesseract's own
default, usually `eng`):

```sh
OCR_LANG=eng+rus wl-uploader --ocr
```

example for sway

```conf
bindsym --to-code Mod4+o exec wl-uploader --ocr
```