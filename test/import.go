package test

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

// LoadTestFile loads a test file from the testdata directory
//
// File contents are returned as a buffer and a map for the HTTP request headers
func LoadTestFile(t *testing.T, filePath string) (*bytes.Buffer, map[string]string) {
	path := path.Join("../../../testdata", filePath)
	body := new(bytes.Buffer)

	mw := multipart.NewWriter(body)

	file, err := os.Open(path)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	w, err := mw.CreateFormFile("file", filePath)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	if _, err := io.Copy(w, file); err != nil {
		assert.Fail(t, err.Error())
	}

	mw.Close()

	return body, map[string]string{"Content-Type": mw.FormDataContentType()}
}
