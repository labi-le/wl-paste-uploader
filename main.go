package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"golang.org/x/net/proxy"
)

const (
	ProxyDefaultTimeout = time.Second * 5
)

func main() {
	ocr := flag.Bool("ocr", false, "recognize text from the clipboard image via OCR and copy it to the clipboard instead of uploading")
	provider := flag.String("provider", "", "upload provider: "+strings.Join(providerNames(), ", ")+" (env UPLOADER_PROVIDER, default "+defaultProvider+")")
	flag.Parse()

	cmd := exec.CommandContext(context.Background(), "wl-paste")
	out, _ := cmd.Output()

	clipboard := bytes.NewBuffer(out)

	if *ocr {
		text, err := Recognize(clipboard)
		if err != nil {
			notify(err.Error())
		}

		if err = clipboardCopy(text); err != nil {
			notify(err.Error())
		}

		notify("Text recognized\n" + text)
		return
	}

	target, err := resolveProvider(*provider)
	if err != nil {
		notify(err.Error())
	}

	file, err := uploadClipboard(target, clipboard)
	if err != nil {
		notify(err.Error())
	}

	if err = clipboardCopy(file); err != nil {
		notify(err.Error())
	}

	notify("Clipboard uploaded\n" + file)
}

// Recognize runs OCR on the given image via the tesseract CLI and returns the
// recognized text. The OCR language can be overridden with the OCR_LANG env var.
func Recognize(clip *bytes.Buffer) (string, error) {
	cmd := exec.CommandContext(context.Background(), "tesseract", ocrArgs(env("OCR_LANG"))...) //nolint:gosec // OCR_LANG comes from the user's own environment, not untrusted input
	cmd.Stdin = ocrInput(clip.Bytes())

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("tesseract not found in PATH, install it to use --ocr")
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("tesseract: %s", msg)
		}
		return "", fmt.Errorf("tesseract: %w", err)
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return "", errors.New("no text recognized")
	}

	return text, nil
}

// ocrMinDimension is the pixel size below which tesseract fails to recognize
// text. Smaller rasters (e.g. small screenshots) are upscaled to around this
// size first, which is what makes OCR work on them.
const ocrMinDimension = 1000

const maxOCRScale = 8

// ocrInput prepares clipboard bytes for tesseract: raster images whose largest
// side is below ocrMinDimension are upscaled, everything else is passed through
// untouched (including non-images, which tesseract may still decode itself).
func ocrInput(raw []byte) io.Reader {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return bytes.NewReader(raw)
	}

	bounds := src.Bounds()
	maxDim := max(bounds.Dx(), bounds.Dy())
	if maxDim == 0 || maxDim >= ocrMinDimension {
		return bytes.NewReader(raw)
	}

	scale := (ocrMinDimension + maxDim - 1) / maxDim
	scale = min(scale, maxOCRScale)

	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx()*scale, bounds.Dy()*scale))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Src, nil)

	var buf bytes.Buffer
	if err = png.Encode(&buf, dst); err != nil {
		return bytes.NewReader(raw)
	}
	return &buf
}

const (
	ocrStreamStdin  = "stdin"
	ocrStreamStdout = "stdout"
)

// ocrArgs builds the tesseract CLI arguments, reading the image from stdin and
// writing the recognized text to stdout, optionally for the given language.
func ocrArgs(lang string) []string {
	args := []string{ocrStreamStdin, ocrStreamStdout}
	if lang != "" {
		args = append(args, "-l", lang)
	}
	return args
}

// clipboardCopy writes content to the Wayland clipboard via wl-copy. Content is
// piped through stdin so text starting with '-' or spanning multiple lines is
// copied verbatim.
func clipboardCopy(content string) error {
	cmd := exec.CommandContext(context.Background(), "wl-copy")
	cmd.Stdin = strings.NewReader(content)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("wl-copy: %w", err)
	}
	return nil
}

func notify(msg string) {
	_ = exec.CommandContext(context.Background(), "notify-send", "wl-uploader", msg).Run()
	os.Exit(0)
}

func env(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
		if val := os.Getenv(strings.ToLower(key)); val != "" {
			return val
		}
	}
	return ""
}

func createProxyClient() (*http.Client, error) {
	proxyRaw := env("HTTPS_PROXY", "HTTP_PROXY", "SOCKS_PROXY", "ALL_PROXY")

	if proxyRaw == "" {
		return http.DefaultClient, nil
	}

	proxyURL, err := url.Parse(proxyRaw)
	if err != nil {
		return nil, fmt.Errorf("proxy url parsing error '%s': %w", proxyRaw, err)
	}

	proxyTimeout, proxyErr := proxyTimeout()
	if proxyErr != nil {
		return nil, proxyErr
	}

	transport := &http.Transport{}

	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)
	case "socks5":
		dialer, dialErr := proxy.FromURL(proxyURL, proxy.Direct)
		if dialErr != nil {
			return nil, fmt.Errorf("failed to create socks5 dialer: %w", dialErr)
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, errors.New("proxy dialer does not support context")
		}
		transport.DialContext = contextDialer.DialContext
	default:
		return nil, fmt.Errorf("unsupported proxy scheme: %s", proxyURL.Scheme)
	}

	return &http.Client{Transport: transport, Timeout: proxyTimeout}, nil
}

func proxyTimeout() (time.Duration, error) {
	raw := env("PROXY_TIMEOUT")
	if raw == "" {
		return ProxyDefaultTimeout, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse proxy timeout %q: %w", raw, err)
	}
	return duration, nil
}
