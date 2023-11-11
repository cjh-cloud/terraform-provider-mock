package main

import (
	"regexp"
	"testing"
)

func TestCreateDir(t *testing.T) {
	dirName := "test_dir"

	err := createDir(dirName)
	if err != nil {
    t.Fatalf(`createDir("%s"), %v`, dirName, err)
  }
}

func TestWriteFile(t *testing.T) {
	filePath := "./test_dir/test.txt"
	fileContents := []byte("Test file contents.")

	err := writeFile(filePath, fileContents)
	if err != nil {
    t.Fatalf(`writeFile("%s","%b"), %v`, filePath, fileContents, err)
  }
}

func TestReadFile(t *testing.T) {
	filePath := "./test_dir/test.txt"
	fileContents := []byte("Test file contents.")

	want := regexp.MustCompile(string(fileContents))
	readContents, err := readFile(filePath)
	if !want.MatchString(readContents[0]) || err != nil {
    t.Fatalf(`readFile("%s","%s"), returned "%s", %v`, filePath, string(fileContents), readContents[0], err)
  }
}

func TestCopyFile(t *testing.T) {
	src := "./test_dir/test.txt"
	dst := "./test_dir/copy.txt"

	nBytes, err := copyFile(src, dst)
	if nBytes < 1 || err != nil {
		t.Fatalf(`copyFile("%s","%s"), %v`, src, dst, err)
	}
}
