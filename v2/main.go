package v2

import (
	"os"
	"os/exec"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func main() {
	a := app.New()
	w := a.NewWindow("Vyhledávač rozhodnutí")
	w.Resize(fyne.NewSize(1000, 800))
	w.CenterOnScreen()

	title := widget.NewLabel("Vyhledávač rozhodnutí")
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

	var rc DuoBool
	rcOrIc := widget.NewSelect([]string{"RČ", "IČ", "Obě"}, func(s string) {
		switch s {
		case "RČ":
			rc.search = true
			rc.term = true
		case "IČ":
			rc.search = true
			rc.term = false
		case "Obě":
			rc.search = false
			rc.term = false
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
		/*
			y, err := os.ReadFile("env/env")
			if err != nil {
				panic(err)
			}*/
		db, err := os.ReadFile("db/db.json")
		if err != nil {
			panic(err)
		}
		res, err := findInJson(db, strings.Split(input.Text, "\n"), zvirepath, rc, DuoBool{}, DuoBool{})
		if err != nil {
			panic(err)
		}
		vysledekstr.Set(res.WidgetText())
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
