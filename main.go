package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const ApiEndpoint = "https://0x0.st"

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
func UploadToHost(url string, file *bytes.Buffer) (string, error) {
	var (
		err    error
		client http.Client
	)

	writer := multipart.NewWriter(file)

	form, err := writer.CreateFormFile("file", "file")
	if err != nil {
		return "", err
	}

	form.Write(file.Bytes())

	writer.Close()

	req, err := http.NewRequest("POST", url, file)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Submit the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		return strings.Replace(bodyString, "\n", "", -1), nil
	}

	return "", fmt.Errorf("bad status: %s", resp.Status)
}

func notify(msg string, exit bool) {
	_ = exec.Command("notify-send", "wl-uploader", msg).Run()

	if exit {
		os.Exit(1)
	}
}
