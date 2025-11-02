package main

import (
	"bytes"
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
	cmd := exec.Command("wl-paste")
	out, _ := cmd.Output()

	b := bytes.NewBuffer([]byte(""))
	if _, err := b.Write(out); err != nil {
		notify(err.Error(), true)
	}

	file, err := Upload(b)
	if err != nil {
		notify(err.Error(), true)
	}

	if err = exec.Command("wl-copy", file).Run(); err != nil {
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

	// 3. Создаем поле формы для файла.
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
