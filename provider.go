package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
)

// userAgent identifies this uploader to file hosts. Several hosts (e.g. 0x0.st)
// ask third-party tools to send a descriptive User-Agent with a contact URL.
const userAgent = "wl-uploader/1.0 (https://github.com/labi-le/wl-paste-uploader)"

// Provider names usable with --provider or UPLOADER_PROVIDER.
const (
	providerZeroX0 = "0x0"
	providerX0     = "x0"
	providerEnvs   = "envs"
	providerCatbox = "catbox"
)

// defaultProvider is used when neither the --provider flag nor the
// UPLOADER_PROVIDER environment variable selects one.
const defaultProvider = providerZeroX0

// Multipart form field names and values used by the providers.
const (
	fieldFile         = "file"
	fieldFileToUpload = "fileToUpload"
	fieldReqType      = "reqtype"
	reqTypeFileUpload = "fileupload"
)

// Provider describes an upload target reachable with a single multipart POST
// whose response body is the resulting URL in plain text.
type Provider struct {
	Name      string            // identifier used by the --provider flag
	Endpoint  string            // upload URL
	FileField string            // multipart form field carrying the file
	Fields    map[string]string // extra form fields sent alongside the file
}

// Upload posts content to the provider and returns the URL from its response.
func (p Provider) Upload(client *http.Client, filename string, content *bytes.Buffer) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range p.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return "", fmt.Errorf("write form field %q: %w", key, err)
		}
	}

	form, err := writer.CreateFormFile(p.FileField, filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err = io.Copy(form, content); err != nil {
		return "", fmt.Errorf("copy file content: %w", err)
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, p.Endpoint, body)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload to %s: %w", p.Name, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return p.parseResponse(resp)
}

// parseResponse turns an upload response into the resulting URL, or an error
// that includes the server's message when the upload was rejected.
func (p Provider) parseResponse(resp *http.Response) (string, error) {
	raw, readErr := io.ReadAll(resp.Body)
	respBody := strings.TrimSpace(string(raw))

	if resp.StatusCode != http.StatusOK {
		if respBody != "" {
			return "", fmt.Errorf("%s response error: %s\n%s", p.Name, resp.Status, respBody)
		}
		return "", fmt.Errorf("%s response error: %s", p.Name, resp.Status)
	}

	if readErr != nil {
		return "", fmt.Errorf("%s: read response: %w", p.Name, readErr)
	}
	if !strings.HasPrefix(respBody, "http") {
		return "", fmt.Errorf("%s: unexpected response: %s", p.Name, respBody)
	}
	return respBody, nil
}

// providers is the registry of supported upload targets.
var providers = map[string]Provider{
	providerZeroX0: {Endpoint: "https://0x0.st", FileField: fieldFile},
	providerX0:     {Endpoint: "https://x0.at", FileField: fieldFile},
	providerEnvs:   {Endpoint: "https://envs.sh", FileField: fieldFile},
	providerCatbox: {
		Endpoint:  "https://catbox.moe/user/api.php",
		FileField: fieldFileToUpload,
		Fields:    map[string]string{fieldReqType: reqTypeFileUpload},
	},
}

// providerNames returns the registered provider names in sorted order.
func providerNames() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveProvider selects a provider by explicit flag value, falling back to the
// UPLOADER_PROVIDER environment variable and finally to defaultProvider.
func resolveProvider(flagValue string) (Provider, error) {
	name := flagValue
	if name == "" {
		name = env("UPLOADER_PROVIDER")
	}
	if name == "" {
		name = defaultProvider
	}

	p, ok := providers[name]
	if !ok {
		return Provider{}, fmt.Errorf("unknown provider %q, available: %s", name, strings.Join(providerNames(), ", "))
	}
	p.Name = name
	return p, nil
}

// uploadClipboard sends the clipboard buffer to the given provider and returns the URL.
func uploadClipboard(p Provider, content *bytes.Buffer) (string, error) {
	client, err := createProxyClient()
	if err != nil {
		return "", fmt.Errorf("http client creation error: %w", err)
	}

	return p.Upload(client, clipboardFilename(content.Bytes()), content)
}

// clipboardFilename derives an upload filename from the content type so image
// hosts serve the file with an extension browsers can render inline.
func clipboardFilename(content []byte) string {
	return "clipboard" + extForContentType(http.DetectContentType(content))
}

// extForContentType maps a detected MIME type to a file extension, returning an
// empty string for types without a well-known clipboard extension.
func extForContentType(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/png"):
		return ".png"
	case strings.HasPrefix(contentType, "image/jpeg"):
		return ".jpg"
	case strings.HasPrefix(contentType, "image/gif"):
		return ".gif"
	case strings.HasPrefix(contentType, "image/webp"):
		return ".webp"
	case strings.HasPrefix(contentType, "image/bmp"):
		return ".bmp"
	case strings.HasPrefix(contentType, "application/pdf"):
		return ".pdf"
	case strings.HasPrefix(contentType, "text/plain"):
		return ".txt"
	default:
		return ""
	}
}
