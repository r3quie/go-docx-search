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
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
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

// Recursive function to walk through directories and files
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

func truthCount(truth []bool) int {
	count := 0
	for _, x := range truth {
		if x {
			count++
		}
	}
	return count
}

// Searches for terms in docx files in a directory.
// Sets the results to a widget, vars target and optiontargets may be omitted if not needed, returns will be needed if done so, see comments in the function
func docxSearch(terms string, path string, target binding.String, optiontarget *widget.Select, rcOrIc IcRc) /*FoundSlice*/ {
	optiontarget.Options = []string{}
	if terms == "" {
		target.Set("Zadejte hledaný výraz")
		return
	}
	if len(terms) < 3 {
		target.Set("Hledaný výraz \"" + terms + "\" je příliš krátký")
		return
	}
	if strings.Contains(terms, "\r") {
		terms = strings.ReplaceAll(terms, "\r", "")
	}
	if strings.Contains(terms, "odst ") {
		terms = strings.ReplaceAll(terms, "odst ", "odst. ")
	}
	if strings.Contains(terms, "písm ") {
		terms = strings.ReplaceAll(terms, "písm ", "písm. ")
	}

	//fmt.Printf("terms are: %v\n", terms)

	var t []string
	var results FoundSlice // FoundSlice is a slice of Found structs, Found is a struct with path, subdir, filename and modtime

	files, _ := os.ReadDir(path)
	// check if there are multiple terms, split them into a slice; if not, put the term into a single-element slice
	if strings.Contains(terms, "\n") {
		t = strings.Split(terms, "\n")
	} else {
		t = []string{terms}
	}

	/*
		for _, x := range t {
			fmt.Printf("element in list t is: \"%v\"\n", x)
		}
	*/

	// Generative function, should be used inside a loop.
	// Should return Found(struct{subdir, filename, modtime}), right now directly modifies the target widget(s)
	walk := func(doc fs.DirEntry, subdr string) {

		// open the docx file
		text, erread := readDocx(path + subdr + doc.Name())
		if erread != nil {
			return
		}
		// search for each term in the document
		var truth []bool
		for _, term := range t {
			truth = append(truth, search(term, text, rcOrIc))
		}

		// if all terms are found in the document, add it to the results
		if !slices.Contains(truth, false) {
			// if modime found, add it to the results
			if nfo, err := doc.Info(); err == nil {
				results = append(results, Found{subdir: subdr, filename: doc.Name(), modtime: nfo.ModTime()})

				// Sort by modtime
				results.Sort()

				// Add results to the target widget and options to the open widget
				target.Set(results.WidgetText())
				optiontarget.Options = results.Options()
				return
			}
			// same thing if modtime not found
			results = append(results, Found{subdir: subdr, filename: doc.Name()})
			target.Set(results.WidgetText())
			optiontarget.Options = append(optiontarget.Options, subdr+doc.Name())
		}
	}

	// walk through the files and first level subdirectories
	walker(files, walk, path, "")

	// if no results are found, return "Not found"
	if len(results) == 0 {
		//return "Not found"
		target.Set("Nenalezeno, vyhledávám nejblžší výsledky")
		results = FoundSlice{Found{truthvalue: 1}}
		// if no results are found, search for the closest results
		walkapprox := func(doc fs.DirEntry, subdr string) {

			// open the docx file
			text, erread := readDocx(path + subdr + doc.Name())
			if erread != nil {
				return
			}
			// search for each term in the document
			var truth []bool
			for _, term := range t {
				truth = append(truth, search(term, text, rcOrIc))
			}

			// if same number of terms were found in the document, add it to the results
			if results[0].truthvalue == truthCount(truth) {
				// if modime found, add it to the results
				if nfo, err := doc.Info(); err == nil {
					results = append(results, Found{subdr, doc.Name(), nfo.ModTime(), truthCount(truth)})

					// Sort by modtime
					results.Sort()

					// Add results to the target widget and options to the open widget
					// unsure whether prepaths is needed, will rewrite in the future
					target.Set(results.WidgetText())
					optiontarget.Options = results.Options()
					return
				}
				// same thing if modtime not found
				results = append(results, Found{subdir: subdr, filename: doc.Name(), truthvalue: truthCount(truth)})
				target.Set(results.WidgetText())
				optiontarget.Options = append(optiontarget.Options, subdr+doc.Name())
			} else if results[0].truthvalue < truthCount(truth) {
				if nfo, err := doc.Info(); err == nil {
					results = FoundSlice{Found{subdir: subdr, filename: doc.Name(), modtime: nfo.ModTime(), truthvalue: truthCount(truth)}}
					target.Set(results.WidgetText())
					optiontarget.Options = results.Options()
					return
				}
				results = FoundSlice{Found{subdir: subdr, filename: doc.Name(), truthvalue: truthCount(truth)}}
				target.Set(results.WidgetText())
				optiontarget.Options = append(optiontarget.Options, subdr+doc.Name())
			}
		}
		walker(files, walkapprox, path, "")
		y, _ := target.Get()
		target.Set(y + fmt.Sprintf("Nejbližší shody v %d ", results[0].truthvalue))
	}

	// if all terms are found in all documents, return "Done" (add to end of widget)
	y, _ := target.Get()
	target.Set(y + "Dokončeno")
	//return results
}

func main() {
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
		case "Obě":
			rc.search = false
			rc.rc = false
		}
	})

	rcOrIc.PlaceHolder = "RČ/IČ"

	zvirata.PlaceHolder = "Vyberte druh zvířete"

	vysledekstr := binding.NewString()
	vysledekstr.Set("")

	vysledek := widget.NewLabelWithData(vysledekstr)
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
		go docxSearch(input.Text, string(y)+zvirepath, vysledekstr, open, rc)
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

	w.SetContent(container.NewHBox(
		layout.NewSpacer(),
		main_container,
		scroll,
		layout.NewSpacer(),
	))

	w.ShowAndRun()
}
