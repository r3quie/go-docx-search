package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
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

func search(term string, path string) bool {
	text, err := readDocx(path)
	if err != nil {
		return false
	}
	return strings.Contains(text, term)
}

func input(terms string, path string) string {
	var t []string
	var paths string
	files, _ := os.ReadDir(path)
	if strings.Contains(terms, "\n") {
		t = strings.Split(terms, "\n")
	} else {
		t = []string{terms}
	}
	for _, file := range files {
		var truth []bool
		for _, term := range t {
			if search(term, (path + file.Name())) {
				truth = append(truth, true)
				continue
			}
			truth = append(truth, false)
		}
		if !slices.Contains(truth, false) {
			paths += (file.Name() + "\n")
		}
	}
	if paths == "" {
		return "Not found"
	}
	return paths
}

func main() {
	y, err := os.ReadFile("env/env")
	if err != nil {
		panic(err)
	}
	fmt.Println(input("§ 15 odst. 1\n§ 19 odst. 1", string(y)))
}
