package main

import (
	"image/png"
	"log"
	"os"

	"github.com/rurewa/codepic-app/internal/clipboard" // ← новое
	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/render"
	"github.com/rurewa/codepic-app/internal/theme"
)

func main() {
	// 1. Флаг --copy — в самом начале, до чтения файла
	copyToClipboard := len(os.Args) > 2 && os.Args[2] == "--copy"

	// 2. Чтение кода (без изменений)
	code := sampleCode
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		code = string(data)
	}

	h := highlight.NewChroma()
	tokens, err := h.Highlight(code, "auto")
	if err != nil {
		log.Fatal(err)
	}

	th, err := theme.Builtin().Get("dracula")
	if err != nil {
		log.Fatal(err)
	}

	r := render.New()
	img, err := r.Render(tokens, th, render.DefaultStyle())
	if err != nil {
		log.Fatal(err)
	}

	// 3. Сохранение PNG (без изменений)
	f, err := os.Create("out.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}

	// 4. Копирование в буфер — после png.Encode
	if copyToClipboard {
		cb, err := clipboard.New()
		if err != nil {
			log.Fatalf("clipboard: %v", err)
		}
		if err := cb.WriteImage(img); err != nil {
			log.Fatalf("clipboard: %v", err)
		}
		log.Println("copied to clipboard")
	}

	log.Printf("wrote out.png (lang=%s, theme=%s)", h.DetectLanguage(code), th.Name)
}

const sampleCode = `package main

import "fmt"

func main() {
	fmt.Println("hello, world")
}
`