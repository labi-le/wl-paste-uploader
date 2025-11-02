# wl-paste-uploader

wl-paste share to 0x0.st and copy result url to clipboard

usage:

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