package main

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"slices"
	"testing"

	"golang.org/x/image/draw"
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

func TestOCRInputUpscalesSmallImage(t *testing.T) {
	small := scalePNG(t, "testdata/ocr_sample.png", 96, 24)

	out, err := io.ReadAll(ocrInput(small))
	if err != nil {
		t.Fatalf("read ocrInput: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("ocrInput output is not an image: %v", err)
	}
	if got := img.Bounds().Dx(); got <= 96 {
		t.Fatalf("small image not upscaled: width = %d, want > 96", got)
	}
}

func TestOCRInputPassesThroughNonImage(t *testing.T) {
	raw := []byte("https://youtu.be/not-an-image")

	out, err := io.ReadAll(ocrInput(raw))
	if err != nil {
		t.Fatalf("read ocrInput: %v", err)
	}
	if !bytes.Equal(out, raw) {
		t.Fatalf("ocrInput mutated non-image input: got %q, want %q", out, raw)
	}
}

func TestRecognizeSmallScreenshot(t *testing.T) {
	requireTesseract(t)

	small := scalePNG(t, "testdata/ocr_sample.png", 96, 24)

	got, err := Recognize(bytes.NewBuffer(small))
	if err != nil {
		t.Fatalf("Recognize on small image: %v", err)
	}

	const want = "Hello OCR World"
	if got != want {
		t.Fatalf("Recognize(small) = %q, want %q", got, want)
	}
}

// scalePNG resizes the fixture to w x h and returns it as PNG bytes.
func scalePNG(t *testing.T, path string, w, h int) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		t.Fatalf("encode small png: %v", err)
	}
	return buf.Bytes()
}

func requireTesseract(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
}
