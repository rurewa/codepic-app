package theme

import (
	"fmt"
	"image/color"
	"sort"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/rurewa/codepic-app/internal/highlight"
)

// Theme — оформление кода: цвета «карточки», базовый цвет текста
// и стиль Chroma для раскраски токенов.
type Theme struct {
	ID          string
	Name        string
	Background  color.RGBA
	Foreground  color.RGBA
	ChromaStyle *chroma.Style
}

// Provider — источник тем для UI и рендерера.
type Provider interface {
	List() []Theme
	Get(id string) (Theme, error)
}

// curated — курируемый список тем, которые видит пользователь.
// Порядок не важен: List() отсортирует по Name.
var curated = []struct {
	id   string
	name string
}{
	{"dracula", "Dracula"},
	{"monokai", "Monokai"},
	{"nord", "Nord"},
	{"solarized-dark", "Solarized Dark"},
	{"solarized-light", "Solarized Light"},
	{"github", "GitHub Light"},
	{"gruvbox", "Gruvbox Dark"},
	{"gruvbox-light", "Gruvbox Light"},
	{"native", "Native"},
	{"vim", "Vim"},
	{"emacs", "Emacs"},
	{"tango", "Tango"},
	{"vs", "Visual Studio"},
	{"xcode", "Xcode"},
	{"friendly", "Friendly"},
}

// Builtin — провайдер встроенных тем на основе Chroma.
func Builtin() Provider { return builtinProvider{} }

type builtinProvider struct{}

func (builtinProvider) List() []Theme {
	existing := make(map[string]struct{}, len(styles.Names()))
	for _, n := range styles.Names() {
		existing[n] = struct{}{}
	}

	result := make([]Theme, 0, len(curated))
	for _, c := range curated {
		if _, ok := existing[c.id]; !ok {
			continue // тема есть в curated, но отсутствует в этой версии Chroma
		}
		result = append(result, makeTheme(c.id, c.name))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (builtinProvider) Get(id string) (Theme, error) {
	for _, c := range curated {
		if c.id == id {
			return makeTheme(c.id, c.name), nil
		}
	}
	return Theme{}, fmt.Errorf("theme: unknown theme %q", id)
}

// makeTheme собирает Theme из зарегистрированного Chroma-стиля.
// styles.Get возвращает Fallback, если стиль не найден, — это допустимая
// деградация: цвета будут нейтральные, но приложение не упадёт.
func makeTheme(id, name string) Theme {
	s := styles.Get(id)
	bgEntry := s.Get(chroma.Background)
	return Theme{
		ID:          id,
		Name:        name,
		Background:  chromaToRGBA(bgEntry.Background),
		Foreground:  chromaToRGBA(bgEntry.Colour),
		ChromaStyle: s,
	}
}

// chromaToRGBA конвертирует chroma.Colour в color.RGBA.
// Незаданный цвет (IsSet() == false) даёт прозрачный чёрный —
// рендерер сам решит, что подставить.
func chromaToRGBA(c chroma.Colour) color.RGBA {
	if !c.IsSet() {
		return color.RGBA{}
	}
	return color.RGBA{R: c.Red(), G: c.Green(), B: c.Blue(), A: 0xFF}
}

// ANSIColor возвращает ANSI-префикс для отрисовки токена в терминале.
// Если у токена нет своего цвета — используется Foreground темы.
// Завершающий сброс "\x1b[0m" добавляет вызывающий.
func ANSIColor(s highlight.Style, t Theme) string {
	var r, g, b uint8
	if s.Color != nil {
		rr, gg, bb, _ := s.Color.RGBA()
		r, g, b = uint8(rr>>8), uint8(gg>>8), uint8(bb>>8)
	} else {
		r, g, b = t.Foreground.R, t.Foreground.G, t.Foreground.B
	}

	prefix := ""
	if s.Bold {
		prefix += "\x1b[1m"
	}
	if s.Italic {
		prefix += "\x1b[3m"
	}
	if s.Underline {
		prefix += "\x1b[4m"
	}
	prefix += fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
	return prefix
}