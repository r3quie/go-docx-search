package main

import (
	"archive/zip"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

type IcRc struct {
	search bool
	rc     bool
}

type Found struct {
	subdir     string
	filename   string
	modtime    time.Time
	truthvalue int
}

type FoundSlice []Found

// Returns a string representation of a Found struct (struct{subdir, filename, modtime})
func (f Found) String() string {
	if len(f.subdir+f.filename) > 63 {
		return fmt.Sprintf("%-63s %s", f.subdir+f.filename[:58]+"...", f.modtime.Format("02.01.2006"))
	}
	return fmt.Sprintf("%-63s %s", f.subdir+f.filename, f.modtime.Format("02.01.2006"))
}

// Sorts the slice of Found structs ([]struct{subdir, filename, modtime}) by modtime
func (f FoundSlice) Sort() {
	sort.Slice(f, func(i, j int) bool {
		return f[i].modtime.After(f[j].modtime)
	})
}

// Returns a string representation of FoundSlice ([]struct{subdir, filename, modtime})
func (f FoundSlice) WidgetText() string {
	var text strings.Builder
	for _, x := range f {
		text.WriteString(x.String() + "\n")
	}
	return text.String()
}

func (f FoundSlice) Options() []string {
	s := make([]string, len(f))
	for i, x := range f {
		s[i] = x.subdir + x.filename
	}
	return s
}

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

// Searches for a term in a docx file, checks for RČ/IČ and returns true if criteria is met
func search(term string, text string, rcOrIc IcRc) bool {
	if text == "" {
		return false
	}
	if rcOrIc.search {
		// returns bool: A && (!B!C || BC); may also be written as A!B!C || ABC
		return strings.Contains(text, term) && (!rcOrIc.rc && strings.Count(text, "IČ") > strings.Count(text, "RČ") || rcOrIc.rc && strings.Count(text, "RČ") > strings.Count(text, "IČ"))
	}
	return strings.Contains(text, term)
}

func truthCount(truth []bool) int {
	count := 0
	for _, x := range truth {
		if x {
			count++
		}
	}
	return count
}
