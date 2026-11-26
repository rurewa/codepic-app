package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/rurewa/codepic-app/internal/app"
)

func main() {
	out := flag.String("out", "out.png", "output PNG path")
	lang := flag.String("lang", "auto", "language (auto | go | python | ...)")
	themeID := flag.String("theme", "", "theme ID (default: from last style)")
	preset := flag.String("preset", "", "load style from named preset")
	savePreset := flag.String("save-preset", "", "save current style as named preset")
	listPresets := flag.Bool("list-presets", false, "list saved presets and exit")
	listThemes := flag.Bool("list-themes", false, "list available themes and exit")
	copyFlag := flag.Bool("copy", false, "copy image to clipboard")
	flag.Parse()

	svc, err := app.New()
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := svc.Config()
	if err != nil {
		log.Printf("warning: %v (using defaults)", err)
	}

	if *listThemes {
		for _, th := range svc.ListThemes() {
			fmt.Printf("%-16s %s\n", th.ID, th.Name)
		}
		return
	}
	if *listPresets {
		if len(cfg.Presets) == 0 {
			fmt.Println("no presets")
			return
		}
		for _, name := range cfg.Names() {
			fmt.Println(name)
		}
		return
	}

	// Стиль: preset > --theme > last saved.
	style := cfg.LastStyle
	if *preset != "" {
		s, err := cfg.GetPreset(*preset)
		if err != nil {
			log.Fatal(err)
		}
		style = s
	}
	if *themeID != "" {
		style.ThemeID = *themeID
	}

	// Код: файл или встроенный пример.
	code := sampleCode
	if path := flag.Arg(0); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		code = string(data)
	}

	// Сохраняем PNG.
	if err := svc.SavePNG(*out, code, *lang, style); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (lang=%s, theme=%s)", *out, svc.DetectLanguage(code), style.ThemeID)

	// Опционально — в буфер.
	if *copyFlag {
		if err := svc.CopyToClipboard(code, *lang, style); err != nil {
			if err == app.ErrNoClipboard {
				log.Println("warning: clipboard unavailable in this environment, PNG saved only")
			} else {
				log.Printf("warning: %v, PNG saved only", err)
			}
		} else {
			log.Println("copied to clipboard")
		}
	}

	// Опционально — сохранить пресет.
	if *savePreset != "" {
		if _, err := svc.AddPreset(*savePreset, style); err != nil {
			log.Fatal(err)
		}
		log.Printf("preset %q saved", *savePreset)
	}
}

const sampleCode = `package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
`