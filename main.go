package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const ApiEndpoint = "https://0x0.st"

func main() {
	rFile := "wl-paste" + randomString(15)
	f, _ := os.CreateTemp(os.TempDir(), rFile)

	defer f.Close()
	defer os.Remove(f.Name())

	cmd := exec.Command("wl-paste")
	out, _ := cmd.Output()
	_, _ = f.Write(out)

	f, _ = os.Open(f.Name())
	file, err := UploadFile(f)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		return
	}

	if err = exec.Command("wl-copy", file).Run(); err != nil {
		os.Stderr.WriteString(err.Error())
		return
	}

	os.Stdout.WriteString(file)
}

// UploadFile takes a file and uploads that file to a file host.
// It returns the url to the uploaded file as a string and any error encountered.
func UploadFile(file *os.File) (string, error) {
	var err error
	var result string

	result, err = UploadToHost(ApiEndpoint, file)
	if err != nil {
		return "", err
	}
	return result, nil
}

//goland:noinspection ALL
func UploadToHost(url string, file *os.File) (string, error) {
	var err error

	values := map[string]io.Reader{
		"file": file,
	}

	var client http.Client
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	for key, r := range values {
		var fw io.Writer
		if x, ok := r.(io.Closer); ok {
			defer x.Close()
		}
		// Add an image file
		if x, ok := r.(*os.File); ok {
			if fw, err = writer.CreateFormFile(key, x.Name()); err != nil {
				return "", err
			}
		}
		if _, err = io.Copy(fw, r); err != nil {
			return "", err
		}

	}
	writer.Close()

	req, err := http.NewRequest("POST", url, &b)
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
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return strings.Replace(bodyString, "\n", "", -1), nil
	}
	return "", fmt.Errorf("bad status: %s", resp.Status)
}

func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:length]
}
