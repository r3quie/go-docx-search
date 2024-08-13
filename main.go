package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type IcRc struct {
	search bool
	rc     bool
}

type Found struct {
	subdir   string
	filename string
	modtime  time.Time
}

type FoundSlice []Found

// Returns a string representation of a Found struct (struct{subdir, filename, modtime})
func (f Found) String() string {
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
	var text string
	for _, x := range f {
		text += x.String() + "\n"
	}
	return text
}

func (f FoundSlice) Options() []string {
	s := []string{}
	for _, x := range f {
		s = append(s, x.subdir+x.filename)
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

func search(term string, path string, rcOrIc IcRc) bool {
	text, err := readDocx(path)
	if err != nil {
		return false
	}
	if rcOrIc.search {
		return strings.Contains(text, term) && (!rcOrIc.rc && strings.Count(text, "IČ") > strings.Count(text, "RČ") || rcOrIc.rc && strings.Count(text, "RČ") > strings.Count(text, "IČ"))
	}
	return strings.Contains(text, term)
}
func walker(files []fs.DirEntry, walk func(fs.DirEntry, string), path string, subdr string) {
	for _, file := range files {
		if file.IsDir() {
			subdir, _ := os.ReadDir(path + subdr + file.Name())
			walker(subdir, walk, path, subdr+file.Name()+"\\")
			continue
		}
		walk(file, subdr)
	}
}

func docxSearch(terms string, path string, target *widget.Label, optiontarget *widget.Select, rcOrIc IcRc) {
	optiontarget.Options = []string{}
	if terms == "" {
		target.SetText("Zadejte hledaný výraz")
		return
	}
	if len(terms) < 3 {
		target.SetText("Hledaný výraz \"" + terms + "\" je příliš krátký")
		return
	}

	if strings.Contains(terms, "odst ") {
		terms = strings.ReplaceAll(terms, "odst ", "odst. ")
	}
	if strings.Contains(terms, "písm ") {
		terms = strings.ReplaceAll(terms, "písm ", "písm. ")
	}

	var t []string
	var results FoundSlice // FoundSlice is a slice of Found structs, Found is a struct with path, subdir, filename and modtime

	files, _ := os.ReadDir(path)

	// check if there are multiple terms, split them into a slice; if not, put the term into a single-element slice
	if strings.Contains(terms, "\n") {
		t = strings.Split(terms, "\n")
	} else {
		t = []string{terms}
	}

	// generative function, should be used inside a loop
	// should return FoundSlice([]struct{subdir, filename, modtime}), right now directly modifies the target widget(s)
	walk := func(doc fs.DirEntry, subdr string) {

		// search for each term in the document
		var truth []bool
		for _, term := range t {
			truth = append(truth, search(term, path+subdr+doc.Name(), rcOrIc))
		}

		// if all terms are found in the document, add it to the results
		if !slices.Contains(truth, false) {
			// if modime found, add it to the results
			if nfo, err := doc.Info(); err == nil {
				results = append(results, Found{subdr, doc.Name(), nfo.ModTime()})

				// Sort by modtime
				results.Sort()

				// Add results to the target widget and options to the open widget
				// unsure whether prepaths is needed, will rewrite in the future
				target.SetText(results.WidgetText())
				optiontarget.Options = results.Options()
				return
			}
			// same thing if modtime not found
			results = append(results, Found{subdir: subdr, filename: doc.Name()})
			target.SetText(results.WidgetText())
			optiontarget.Options = append(optiontarget.Options, subdr+doc.Name())
		}
	}

	// walk through the files and first level subdirectories
	walker(files, walk, path, "")

	// if no results are found, return "Not found"
	if len(results) == 0 {
		//return "Not found"
		target.SetText("Nenalezeno")
		return
	}

	// if all terms are found in all documents, return "Done" (add to end of widget)
	//return paths
	target.SetText(target.Text + "Dokončeno")
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
	w := a.NewWindow("Vyhledávač rozhodnuí")
	w.Resize(fyne.NewSize(1000, 800))
	w.CenterOnScreen()

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
			if zvirepath == "" {
				return
			}
			zvirepath = ""
		}
	})

	var rc IcRc
	rcOrIc := widget.NewSelect([]string{"RČ", "IČ", "Obě"}, func(s string) {
		switch s {
		case "RČ":
			rc.search = true
			rc.rc = true
		case "IČ":
			rc.search = true
			rc.rc = false
		default:
			rc.search = false
		}
	})

	rcOrIc.PlaceHolder = "RČ/IČ"

	zvirata.PlaceHolder = "Vyberte druh zvířete"

	vysledek := widget.NewLabel("")
	vysledek.TextStyle = fyne.TextStyle{Monospace: true}

	open := widget.NewSelect([]string{}, func(s string) {
		// open env to get path
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}

		// open file in explorer
		exec.Command(`explorer`, `/select,`, string(y)+zvirepath+s).Run()
	})
	open.PlaceHolder = "Vyberte příkaz k otevření"

	search := widget.NewButton("Hledat", func() {
		vysledek.SetText("Hledám...")
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}
		go docxSearch(input.Text, string(y)+zvirepath, vysledek, open, rc)
	})

	choices := container.New(layout.NewHBoxLayout(), zvirata, rcOrIc)

	main_container := container.New(layout.NewVBoxLayout(),
		title,
		labeldat,
		input,
		choices,
		search,
		open,
	)

	scroll := container.NewScroll(vysledek)
	scroll.SetMinSize(fyne.Size{Width: 700, Height: 200})

	//vysledek_container := container.New(layout.NewHBoxLayout(), scroll)

	w.SetContent(container.NewHBox(
		layout.NewSpacer(),
		main_container,
		scroll,
		layout.NewSpacer(),
	))

	w.ShowAndRun()
}
