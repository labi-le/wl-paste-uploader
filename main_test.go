package main

import (
	"bytes"
	"os"
	"os/exec"
	"slices"
	"testing"
)

func TestOCRArgs(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want []string
	}{
		{name: "default reads stdin writes stdout", lang: "", want: []string{"stdin", "stdout"}},
		{name: "language appends -l flag", lang: "eng", want: []string{"stdin", "stdout", "-l", "eng"}},
		{name: "multi language passes value verbatim", lang: "eng+rus", want: []string{"stdin", "stdout", "-l", "eng+rus"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ocrArgs(tt.lang); !slices.Equal(got, tt.want) {
				t.Fatalf("ocrArgs(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

func TestRecognize(t *testing.T) {
	requireTesseract(t)

	data, err := os.ReadFile("testdata/ocr_sample.png")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := Recognize(bytes.NewBuffer(data))
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}

	const want = "Hello OCR World"
	if got != want {
		t.Fatalf("Recognize = %q, want %q", got, want)
	}
}

func TestRecognizeRejectsNonImage(t *testing.T) {
	requireTesseract(t)

	if _, err := Recognize(bytes.NewBufferString("not an image")); err == nil {
		t.Fatal("Recognize accepted non-image input, want error")
	}
}

func requireTesseract(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
}
