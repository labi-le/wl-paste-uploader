package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderUploadSuccessSendsMultipart(t *testing.T) {
	var gotField, gotFilename, gotReqtype, gotMethod, gotUA, gotContent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotUA = r.Header.Get("User-Agent")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		gotReqtype = r.FormValue("reqtype")
		for name, files := range r.MultipartForm.File {
			gotField = name
			gotFilename = files[0].Filename
			f, openErr := files[0].Open()
			if openErr != nil {
				t.Errorf("open uploaded file: %v", openErr)
				continue
			}
			b, _ := io.ReadAll(f)
			_ = f.Close()
			gotContent = string(b)
		}
		io.WriteString(w, "https://files.example.com/abc.png\n")
	}))
	defer srv.Close()

	p := Provider{
		Name:      "test",
		Endpoint:  srv.URL,
		FileField: "fileToUpload",
		Fields:    map[string]string{"reqtype": "fileupload"},
	}
	url, err := p.Upload(srv.Client(), "clipboard.png", bytes.NewBufferString("PNGDATA"))
	if err != nil {
		t.Fatalf("Upload error: %v", err)
	}
	if url != "https://files.example.com/abc.png" {
		t.Fatalf("url = %q, want trimmed plain url", url)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotField != "fileToUpload" {
		t.Errorf("file field = %q, want fileToUpload", gotField)
	}
	if gotFilename != "clipboard.png" {
		t.Errorf("filename = %q, want clipboard.png", gotFilename)
	}
	if gotReqtype != "fileupload" {
		t.Errorf("reqtype = %q, want fileupload", gotReqtype)
	}
	if gotUA == "" {
		t.Errorf("User-Agent header not set")
	}
	if gotContent != "PNGDATA" {
		t.Errorf("uploaded content = %q, want PNGDATA", gotContent)
	}
}

func TestProviderUploadErrorIncludesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, "uploads disabled")
	}))
	defer srv.Close()

	p := Provider{Name: "0x0", Endpoint: srv.URL, FileField: "file"}
	_, err := p.Upload(srv.Client(), "clipboard.png", bytes.NewBufferString("x"))
	if err == nil {
		t.Fatal("expected error on 503")
	}
	if !strings.Contains(err.Error(), "uploads disabled") || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error must include status and body, got: %v", err)
	}
}

func TestProviderUploadRejectsNonURLResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "something went wrong")
	}))
	defer srv.Close()

	p := Provider{Name: "catbox", Endpoint: srv.URL, FileField: "fileToUpload"}
	_, err := p.Upload(srv.Client(), "clipboard.png", bytes.NewBufferString("x"))
	if err == nil {
		t.Fatal("expected error when 200 body is not a url")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Fatalf("error should surface unexpected response, got: %v", err)
	}
}

func TestResolveProviderPrecedence(t *testing.T) {
	t.Setenv("UPLOADER_PROVIDER", "x0")

	p, err := resolveProvider("catbox")
	if err != nil || p.Name != "catbox" {
		t.Fatalf("flag must win over env: name=%q err=%v", p.Name, err)
	}

	p, err = resolveProvider("")
	if err != nil || p.Name != "x0" {
		t.Fatalf("env must apply when flag empty: name=%q err=%v", p.Name, err)
	}
}

func TestResolveProviderDefault(t *testing.T) {
	t.Setenv("UPLOADER_PROVIDER", "")

	p, err := resolveProvider("")
	if err != nil {
		t.Fatalf("default resolve: %v", err)
	}
	if p.Name != defaultProvider {
		t.Fatalf("default = %q, want %q", p.Name, defaultProvider)
	}
}

func TestResolveProviderUnknown(t *testing.T) {
	t.Setenv("UPLOADER_PROVIDER", "")

	_, err := resolveProvider("bogus")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("error should name the bad provider, got: %v", err)
	}
}

func TestExtForContentType(t *testing.T) {
	cases := map[string]string{
		"image/png":                 ".png",
		"image/jpeg":                ".jpg",
		"image/gif":                 ".gif",
		"image/webp":                ".webp",
		"text/plain; charset=utf-8": ".txt",
		"application/octet-stream":  "",
	}
	for ct, want := range cases {
		if got := extForContentType(ct); got != want {
			t.Errorf("extForContentType(%q) = %q, want %q", ct, got, want)
		}
	}
}

func TestClipboardFilenameDetectsPNG(t *testing.T) {
	png := append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 32)...)
	if got := clipboardFilename(png); got != "clipboard.png" {
		t.Fatalf("clipboardFilename(png) = %q, want clipboard.png", got)
	}
}

func TestClipboardFilenameNoExtForBinary(t *testing.T) {
	if got := clipboardFilename([]byte{0x00, 0x01, 0x02, 0x03}); got != "clipboard" {
		t.Fatalf("clipboardFilename(binary) = %q, want clipboard", got)
	}
}
