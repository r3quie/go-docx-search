package main

import (
	"archive/zip"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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

func docxSearch(terms string, path string, target *widget.Label) {
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
			truth = append(truth, search(term, path+file.Name()))
		}
		if !slices.Contains(truth, false) {
			paths += (file.Name() + "\n")
			target.SetText(paths)
		}
	}
	if paths == "" {
		//return "Not found"
		target.SetText("Nenalezeno")
	}
	paths = paths[:len(paths)-1]
	//return paths
	target.SetText(paths)
}

func main() {
	/*
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}
		fmt.Println(input("§ 15 odst. 1\n§ 19 odst. 1", string(y)))
	*/
	a := app.New()
	w := a.NewWindow("Kalkulačka lhůt")

	title := widget.NewLabel("Vyhledávač rozhodnuí")
	labeldat := widget.NewLabel("Zadejte hledaná ustanovení")

	input := widget.NewMultiLineEntry()
	input.PlaceHolder = "§ 15 odst. 1\n§ 19 odst. 1\n§ 23 odst. 1 písm. c)"

	var zvirepath string

	zvirata := widget.NewSelect([]string{"Koně", "Ovce/kozy", "Prasata", "Tuři", "Všechna"}, func(s string) {
		switch s {
		case "Koně":
			zvirepath = "K\\"
		case "Ovce/kozy":
			zvirepath = "O\\"
		case "Prasata":
			zvirepath = "P\\"
		case "Tuři":
			zvirepath = "T\\"
		case "Všechna":
			zvirepath = ""
		}
	})
	zvirata.PlaceHolder = "Vyberte druh zvířete"

	vysledek := widget.NewLabel("")

	search := widget.NewButton("Hledat", func() {
		vysledek.SetText("Hledám...")
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}
		go docxSearch(input.Text, string(y)+zvirepath, vysledek)
	})

	w.SetContent(container.NewVBox(
		title,
		labeldat,
		input,
		zvirata,
		search,
		vysledek,
	))

	w.ShowAndRun()
}
