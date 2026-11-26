// cmd/codepic-cli/main.go
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/rurewa/codepic-app/internal/clipboard"
	"github.com/rurewa/codepic-app/internal/config"
	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/render"
	"github.com/rurewa/codepic-app/internal/theme"
)

func main() {
	out := flag.String("out", "out.png", "output PNG path")
	lang := flag.String("lang", "auto", "language (auto | go | python | ...)")
	themeID := flag.String("theme", "dracula", "theme ID")
	preset := flag.String("preset", "", "load style from named preset")
	savePreset := flag.String("save-preset", "", "save current style as named preset")
	listPresets := flag.Bool("list-presets", false, "list saved presets and exit")
	copyFlag := flag.Bool("copy", false, "copy image to clipboard")
	flag.Parse()

	store, err := config.NewFileStore()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		log.Printf("warning: %v (using defaults)", err)
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

	// Стиль: preset > флаги > last saved > default
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

	// Код: файл или встроенный пример
	code := sampleCode
	if path := flag.Arg(0); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		code = string(data)
	}

	h := highlight.NewChroma()
	if *lang != "auto" {
		cfg.LastLanguage = *lang
	}
	tokens, err := h.Highlight(code, *lang)
	if err != nil {
		log.Fatal(err)
	}

	th, err := theme.Builtin().Get(style.ThemeID)
	if err != nil {
		log.Fatal(err)
	}

	img, err := render.New().Render(tokens, th, style)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		log.Fatal(err)
	}
	f.Close()
	log.Printf("wrote %s (lang=%s, theme=%s)", *out, h.DetectLanguage(code), th.Name)

	if *copyFlag {
		cb, err := clipboard.New()
		if err != nil {
			log.Printf("warning: clipboard unavailable (%v), PNG saved only", err)
		} else if err := cb.WriteImage(img); err != nil {
			log.Printf("warning: clipboard write failed (%v), PNG saved only", err)
		} else {
			log.Println("copied to clipboard")
		}
	}

	if *savePreset != "" {
		cfg, err := cfg.AddPreset(*savePreset, style)
		if err != nil {
			log.Fatal(err)
		}
		if err := store.Save(cfg); err != nil {
			log.Fatal(err)
		}
		log.Printf("preset %q saved (%s)", *savePreset, store.Path())
	}
}

const sampleCode = `package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
`