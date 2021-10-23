package multipartestutils_test

import (
	"io/ioutil"
	"testing"

	"github.com/goplaid/multipartestutils"
)

func TestCreateMultipartFileHeader(t *testing.T) {
	f := multipartestutils.CreateMultipartFileHeader("test.txt", []byte("hello"))
	if f.Filename != "test.txt" {
		t.Error(f.Filename)
	}
	file, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "hello" {
		t.Error(string(content))
	}
}
