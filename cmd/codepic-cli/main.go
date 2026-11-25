// cmd/codepic-cli/main.go
package main

import (
	"fmt"
	"os"

	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/theme"
)

func main() {
	code := sampleCode
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		code = string(data)
	}

	h := highlight.NewChroma()
	tokens, err := h.Highlight(code, "auto")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	th, err := theme.Builtin().Get("dracula")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Detected language: %s\n", h.DetectLanguage(code))
	fmt.Printf("Theme: %s (bg=%v, fg=%v)\n\n", th.Name, th.Background, th.Foreground)

	for _, tok := range tokens {
		fmt.Print(theme.ANSIColor(tok.Style, th), tok.Text, "\x1b[0m")
	}
}

const sampleCode = `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`