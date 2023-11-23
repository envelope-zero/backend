package controllers_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path"
)

func (suite *TestSuiteStandard) loadTestFile(filePath string) (*bytes.Buffer, map[string]string) {
	path := path.Join("../../testdata", filePath)
	body := new(bytes.Buffer)

	mw := multipart.NewWriter(body)

	file, err := os.Open(path)
	if err != nil {
		suite.Assert().Fail(err.Error())
	}

	w, err := mw.CreateFormFile("file", filePath)
	if err != nil {
		suite.Assert().Fail(err.Error())
	}

	if _, err := io.Copy(w, file); err != nil {
		suite.Assert().Fail(err.Error())
	}

	mw.Close()

	return body, map[string]string{"Content-Type": mw.FormDataContentType()}
}
