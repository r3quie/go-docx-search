package main

import (
	"archive/zip"
	"io"
	"regexp"
	"strings"
)

// Returns text from a doc or docx file as string
func readDocx(src string) (string, error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			// fmt.Println("found")
			doc, errdoc := f.Open()
			if errdoc != nil {
				return "", errdoc
			}
			buf := new(strings.Builder)
			_, errd := io.Copy(buf, doc)
			if errd != nil {
				return "", errd
			}
			reg, _ := regexp.Compile(`\<.*?\>`)
			return reg.ReplaceAllString(buf.String(), ""), nil

		}
	}
	return "", nil
}

func search(term string, path string) (bool, error) {
	text, err := readDocx(path)
	if err != nil {
		return false, err
	}
	return strings.Contains(text, term), nil
}

func main() {

}
