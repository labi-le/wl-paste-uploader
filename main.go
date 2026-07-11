package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const (
	ApiEndpoint         = "https://0x0.st"
	ProxyDefaultTimeout = time.Second * 5
)

func main() {
	ocr := flag.Bool("ocr", false, "recognize text from the clipboard image via OCR and copy it to the clipboard instead of uploading")
	flag.Parse()

	cmd := exec.Command("wl-paste")
	out, _ := cmd.Output()

	clipboard := bytes.NewBuffer(out)

	if *ocr {
		text, err := Recognize(clipboard)
		if err != nil {
			notify(err.Error(), true)
		}

		if err := clipboardCopy(text); err != nil {
			notify(err.Error(), true)
		}

		notify("Text recognized\n"+text, true)
		return
	}

	file, err := Upload(clipboard)
	if err != nil {
		notify(err.Error(), true)
	}

	if err := clipboardCopy(file); err != nil {
		notify(err.Error(), true)
	}

	notify("Clipboard uploaded\n"+file, true)
}

// Upload takes a file and uploads that file to a file host.
// It returns the url to the uploaded file as a string and any error encountered.
func Upload(file *bytes.Buffer) (string, error) {
	var err error
	var result string

	result, err = UploadToHost(ApiEndpoint, file)
	if err != nil {
		return "", err
	}
	return result, nil
}

//goland:noinspection ALL
func UploadToHost(endpointURL string, fileContent *bytes.Buffer) (string, error) {
	client, err := createProxyClient()
	if err != nil {
		return "", fmt.Errorf("http client creation error: %w", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	form, err := writer.CreateFormFile("file", "clipboard-data")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(form, fileContent); err != nil {
		return "", err
	}

	writer.Close()

	req, err := http.NewRequest("POST", endpointURL, body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "wl-uploader/1.0 (https://github.com/labi-le/wl-paste-uploader)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return strings.TrimSpace(string(bodyBytes)), nil
	}

	return "", fmt.Errorf("server response error:%s", resp.Status)
}

// Recognize runs OCR on the given image via the tesseract CLI and returns the
// recognized text. The OCR language can be overridden with the OCR_LANG env var.
func Recognize(image *bytes.Buffer) (string, error) {
	cmd := exec.Command("tesseract", ocrArgs(env("OCR_LANG"))...)
	cmd.Stdin = image

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("tesseract not found in PATH, install it to use --ocr")
		}
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("tesseract: %s", msg)
		}
		return "", fmt.Errorf("tesseract: %w", err)
	}

	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return "", fmt.Errorf("no text recognized")
	}

	return text, nil
}

// ocrArgs builds the tesseract CLI arguments, reading the image from stdin and
// writing the recognized text to stdout, optionally for the given language.
func ocrArgs(lang string) []string {
	args := []string{"stdin", "stdout"}
	if lang != "" {
		args = append(args, "-l", lang)
	}
	return args
}

// clipboardCopy writes content to the Wayland clipboard via wl-copy. Content is
// piped through stdin so text starting with '-' or spanning multiple lines is
// copied verbatim.
func clipboardCopy(content string) error {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func notify(msg string, exit bool) {
	_ = exec.Command("notify-send", "wl-uploader", msg).Run()

	if exit {
		os.Exit(0)
	}
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
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create socks5 dialer: %w", err)
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("proxy dialer does not support context")
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

	return time.ParseDuration(raw)
}
