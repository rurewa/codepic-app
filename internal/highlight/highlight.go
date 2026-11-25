package highlight

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Style — упрощённое представление визуального стиля токена.
// Не зависит от Chroma, чтобы render и UI не тащили его API.
type Style struct {
	Color     color.Color
	Bold      bool
	Italic    bool
	Underline bool
}

// Token — атомарный фрагмент кода с единым стилем.
// Newline == true означает, что токен заканчивается символом '\n'
// (нужно рендереру для построчной разбивки).
type Token struct {
	Text    string
	Style   Style
	Newline bool
}

// Language — метаданные для UI-списка.
type Language struct {
	ID   string // chroma syntax ID: "go", "python", "typescript"
	Name string // человекочитаемое: "Go", "Python", "TypeScript"
}

// Highlighter — единственный публичный интерфейс пакета.
type Highlighter interface {
	Highlight(code, lang string) ([]Token, error)
	DetectLanguage(code string) string
	Languages() []Language
}

// chromaHighlighter — реализация на Chroma v2.
type chromaHighlighter struct {
	// style — стиль Chroma, из которого берутся цвета токенов.
	// Может быть переопределён через NewChromaWithStyle.
	style *chroma.Style
}

// NewChroma создаёт Highlighter с дефолтным стилем Chroma (Fallback).
// Для выбора конкретной темы используйте NewChromaWithStyle.
func NewChroma() Highlighter {
	return &chromaHighlighter{style: styles.Fallback}
}

// NewChromaWithStyle позволяет передать тему Chroma (например, из пакета theme).
// Если style == nil, используется styles.Fallback.
func NewChromaWithStyle(style *chroma.Style) Highlighter {
	if style == nil {
		style = styles.Fallback
	}
	return &chromaHighlighter{style: style}
}

// Highlight разбирает code и возвращает плоский список токенов.
// lang == "auto" → автоопределение; "" или "plaintext" → без подсветки.
func (h *chromaHighlighter) Highlight(code, lang string) ([]Token, error) {
	lexer := h.chooseLexer(code, lang)
	// Coalesce склеивает соседние токены одного типа — меньше токенов, проще рендер.
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return nil, fmt.Errorf("highlight: tokenise: %w", err)
	}

	var tokens []Token
	for _, tok := range iterator.Tokens() {
		// Пропускаем служебные токены (EOF, Error и т.п. с пустым текстом).
		if tok.Value == "" {
			continue
		}

		entry := h.style.Get(tok.Type)
		tokens = append(tokens, Token{
			Text: tok.Value,
			Style: Style{
				Color:     chromaColourToColor(entry.Colour),
				Bold:      entry.Bold == chroma.Yes,
				Italic:    entry.Italic == chroma.Yes,
				Underline: entry.Underline == chroma.Yes,
			},
			Newline: strings.HasSuffix(tok.Value, "\n"),
		})
	}
	return tokens, nil
}

// DetectLanguage возвращает Chroma syntax ID наиболее вероятного языка.
// Если определить не удалось — "plaintext".
func (h *chromaHighlighter) DetectLanguage(code string) string {
	lexer := lexers.Analyse(code)
	if lexer == nil {
		return "plaintext"
	}
	cfg := lexer.Config()
	if len(cfg.Aliases) > 0 {
		return cfg.Aliases[0] // "go", "python", "rust"
	}
	return cfg.Name
}

// Languages возвращает курируемый список языков для UI-селектора.
// Не все 200+ лексеров Chroma — только популярные.
func (h *chromaHighlighter) Languages() []Language {
	return []Language{
		{ID: "go", Name: "Go"},
		{ID: "python", Name: "Python"},
		{ID: "javascript", Name: "JavaScript"},
		{ID: "typescript", Name: "TypeScript"},
		{ID: "rust", Name: "Rust"},
		{ID: "java", Name: "Java"},
		{ID: "c", Name: "C"},
		{ID: "cpp", Name: "C++"},
		{ID: "csharp", Name: "C#"},
		{ID: "sql", Name: "SQL"},
		{ID: "bash", Name: "Bash"},
		{ID: "json", Name: "JSON"},
		{ID: "yaml", Name: "YAML"},
		{ID: "html", Name: "HTML"},
		{ID: "css", Name: "CSS"},
		{ID: "markdown", Name: "Markdown"},
		{ID: "plaintext", Name: "Plain text"},
	}
}

// chooseLexer — внутренняя логика выбора лексера.
func (h *chromaHighlighter) chooseLexer(code, lang string) chroma.Lexer {
	switch lang {
	case "", "auto":
		if lexer := lexers.Analyse(code); lexer != nil {
			return lexer
		}
		return lexers.Fallback
	case "plaintext":
		return lexers.Fallback
	default:
		if lexer := lexers.Get(lang); lexer != nil {
			return lexer
		}
		return lexers.Fallback
	}
}

// chromaColourToColor конвертирует chroma.Colour в color.Color.
// Если цвет не задан, возвращает nil — рендерер подставит цвет темы.
func chromaColourToColor(c chroma.Colour) color.Color {
	if !c.IsSet() {
		return nil
	}
	return color.RGBA{R: c.Red(), G: c.Green(), B: c.Blue(), A: 0xFF}
}