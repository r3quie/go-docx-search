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

type Found struct {
	path     string
	subdir   string
	filename string
	modtime  time.Time
}

func (f Found) String() string {
	return fmt.Sprintf("%-63s %s", f.subdir+f.filename, f.modtime.Format("02.01.2006"))
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

func search(term string, path string) bool {
	text, err := readDocx(path)
	if err != nil {
		return false
	}
	return strings.Contains(text, term)
}

func docxSearch(terms string, path string, target *widget.Label, optiontarget *widget.Select) {
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
	var results []Found
	var paths string

	files, _ := os.ReadDir(path)
	if strings.Contains(terms, "\n") {
		t = strings.Split(terms, "\n")
	} else {
		t = []string{terms}
	}

	// generative function, should be used inside a loop, will change to return a [][string, time.Time] in the future
	walk := func(doc fs.DirEntry, subdr string) {
		var truth []bool
		for _, term := range t {
			truth = append(truth, search(term, path+subdr+doc.Name()))
		}

		if !slices.Contains(truth, false) {
			if nfo, err := doc.Info(); err == nil {
				found := Found{path, subdr, doc.Name(), nfo.ModTime()}
				results = append(results, found)
				sort.Slice(results, func(i, j int) bool {
					return results[i].modtime.Before(results[j].modtime)
				})
				var prepaths string
				for _, x := range results {
					prepaths += x.String() + "\n"
				}
				paths = prepaths
				target.SetText(paths)
				optiontarget.Options = append(optiontarget.Options, found.subdir+found.filename)
				return
			}
			paths += (subdr + doc.Name() + "\n")
			target.SetText(paths)
			optiontarget.Options = append(optiontarget.Options, subdr+doc.Name())
		}
	}

	for _, file := range files {
		if file.IsDir() {
			subdir, _ := os.ReadDir(path + file.Name())
			for _, subfile := range subdir {
				walk(subfile, file.Name()+"\\")
			}
			continue
		}
		walk(file, "")
	}
	if paths == "" {
		//return "Not found"
		target.SetText("Nenalezeno")
		return
	}
	paths += "Dokončeno"
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
	zvirata.PlaceHolder = "Vyberte druh zvířete"

	vysledek := widget.NewLabel("")
	vysledek.TextStyle = fyne.TextStyle{Monospace: true}

	open := widget.NewSelect([]string{}, func(s string) {
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}
		if zvirepath == "" {
			zvirepath = "\\"
		}
		if zvirepath == "" {
			exec.Command(`explorer`, `/select,`, string(y)+s).Run()
			return
		}
		exec.Command(`explorer`, `/select,`, string(y)+zvirepath+s).Run()
	})
	open.PlaceHolder = "Vyberte příkaz k otevření"

	search := widget.NewButton("Hledat", func() {
		vysledek.SetText("Hledám...")
		y, err := os.ReadFile("env/env")
		if err != nil {
			panic(err)
		}
		go docxSearch(input.Text, string(y)+zvirepath, vysledek, open)
	})

	main_container := container.New(layout.NewVBoxLayout(),
		title,
		labeldat,
		input,
		zvirata,
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
