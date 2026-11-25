package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"

	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/theme"
)

// Style — всё оформление картинки. Плоская структура без указателей,
// чтобы легко сериализоваться в JSON для пресетов.
type Style struct {
	ThemeID string `json:"theme_id"`

	FontSize   float64 `json:"font_size"`
	LineHeight float64 `json:"line_height"`

	Padding      int `json:"padding"`
	CornerRadius int `json:"corner_radius"`
	MaxWidth     int `json:"max_width"` // 0 = auto по содержимому

	Background   color.RGBA `json:"background"`
	Gradient     bool       `json:"gradient"`
	GradientFrom color.RGBA `json:"gradient_from"`
	GradientTo   color.RGBA `json:"gradient_to"`

	ShowLineNumbers bool `json:"show_line_numbers"`
	ShowWindowDots  bool `json:"show_window_dots"`
	Shadow          bool `json:"shadow"`

	Scale float64 `json:"scale"` // 1.0 или 2.0 (HiDPI)
}

// DefaultStyle — разумные значения по умолчанию.
func DefaultStyle() Style {
	return Style{
		ThemeID:         "dracula",
		FontSize:        14,
		LineHeight:      1.5,
		Padding:         48,
		CornerRadius:    12,
		Background:      color.RGBA{R: 30, G: 30, B: 46, A: 255},
		GradientFrom:    color.RGBA{R: 118, G: 75, B: 162, A: 255},
		GradientTo:      color.RGBA{R: 255, G: 128, B: 181, A: 255},
		ShowWindowDots:  true,
		Shadow:          true,
		Scale:           2.0,
	}
}

// Renderer превращает токены в изображение.
type Renderer interface {
	Render(tokens []highlight.Token, th theme.Theme, s Style) (image.Image, error)
}

func New() Renderer { return renderer{} }

type renderer struct{}

// Шрифт парсим один раз при первом рендере.
// gomono встроен в golang.org/x/image — не нужны внешние TTF-файлы.
var monoFont *opentype.Font

func loadFont() (*opentype.Font, error) {
	if monoFont != nil {
		return monoFont, nil
	}
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("render: parse font: %w", err)
	}
	monoFont = f
	return monoFont, nil
}

func (renderer) Render(tokens []highlight.Token, th theme.Theme, s Style) (image.Image, error) {
	s = s.withDefaults()

	fnt, err := loadFont()
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    s.FontSize * s.Scale,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("render: new face: %w", err)
	}
	defer face.Close()

	metrics := face.Metrics()
	fontHeight := float64(metrics.Height.Ceil())
	ascent := float64(metrics.Ascent.Ceil())
	lineHeight := fontHeight * s.LineHeight

	lines := splitLines(tokens)
	if len(lines) == 0 {
		lines = [][]highlight.Token{{}}
	}

	// Ширина самой длинной строки
	var maxLineWidth float64
	for _, line := range lines {
		w := measureLine(face, line)
		if w > maxLineWidth {
			maxLineWidth = w
		}
	}

	// Ширина колонки с номерами строк (по самой длинной цифре)
	var lineNumWidth float64
	if s.ShowLineNumbers {
		sample := fmt.Sprintf("%d", len(lines))
		lineNumWidth = float64(font.MeasureString(face, sample).Ceil()) + 24*s.Scale
	}

	padX := float64(s.Padding) * s.Scale
	padY := float64(s.Padding) * s.Scale
	cardPadX := padX * 0.75
	cardPadY := padY * 0.75

	var headerHeight float64
	if s.ShowWindowDots {
		headerHeight = 36 * s.Scale
	}

	cardWidth := maxLineWidth + lineNumWidth + cardPadX*2
	cardHeight := lineHeight*float64(len(lines)) + cardPadY*2 + headerHeight

	width := int(cardWidth + padX*2)
	height := int(cardHeight + padY*2)

	if s.MaxWidth > 0 && width > s.MaxWidth*int(s.Scale) {
		width = s.MaxWidth * int(s.Scale)
	}
	if width < 100 {
		width = 100
	}
	if height < 100 {
		height = 100
	}

	dc := gg.NewContext(width, height)

	// 1. Внешний фон или градиент
	if s.Gradient {
		g := gg.NewLinearGradient(0, 0, float64(width), float64(height))
		g.AddColorStop(0, s.GradientFrom)
		g.AddColorStop(1, s.GradientTo)
		dc.SetFillStyle(g)
	} else {
		dc.SetColor(s.Background)
	}
	dc.DrawRectangle(0, 0, float64(width), float64(height))
	dc.Fill()

	// 2. Карточка
	cardX := padX
	cardY := padY
	radius := float64(s.CornerRadius) * s.Scale

	if s.Shadow {
		dc.SetColor(color.RGBA{0, 0, 0, 80})
		dc.DrawRoundedRectangle(cardX+4*s.Scale, cardY+6*s.Scale, cardWidth, cardHeight, radius)
		dc.Fill()
	}

	dc.SetColor(th.Background)
	dc.DrawRoundedRectangle(cardX, cardY, cardWidth, cardHeight, radius)
	dc.Fill()

	// 3. Три точки в шапке карточки
	if s.ShowWindowDots {
		dotRadius := 6 * s.Scale
		dotY := cardY + headerHeight/2
		dotX := cardX + 20*s.Scale + dotRadius
		dots := []color.RGBA{
			{R: 255, G: 95, B: 86, A: 255},
			{R: 255, G: 189, B: 46, A: 255},
			{R: 39, G: 201, B: 63, A: 255},
		}
		for i, c := range dots {
			dc.SetColor(c)
			dc.DrawCircle(dotX+float64(i)*20*s.Scale, dotY, dotRadius)
			dc.Fill()
		}
	}

	// 4. Код
	dc.SetFontFace(face)
	codeX := cardX + cardPadX + lineNumWidth
	codeTop := cardY + headerHeight + cardPadY
	baseline := codeTop + ascent

	for i, line := range lines {
		y := baseline + float64(i)*lineHeight

		if s.ShowLineNumbers {
			dc.SetColor(color.RGBA{R: 128, G: 128, B: 128, A: 180})
			dc.DrawString(fmt.Sprintf("%d", i+1), cardX+cardPadX*0.5, y)
		}

		cursor := codeX
		for _, tok := range line {
			c := tok.Style.Color
			if c == nil {
				c = th.Foreground
			}
			dc.SetColor(c)
			dc.DrawString(tok.Text, cursor, y)
			cursor += measureString(face, tok.Text)
		}
	}

	return dc.Image(), nil
}

func (s Style) withDefaults() Style {
	d := DefaultStyle()
	if s.FontSize <= 0 {
		s.FontSize = d.FontSize
	}
	if s.LineHeight <= 0 {
		s.LineHeight = d.LineHeight
	}
	if s.Padding <= 0 {
		s.Padding = d.Padding
	}
	if s.CornerRadius <= 0 {
		s.CornerRadius = d.CornerRadius
	}
	if s.Scale <= 0 {
		s.Scale = d.Scale
	}
	if s.Background == (color.RGBA{}) {
		s.Background = d.Background
	}
	if s.GradientFrom == (color.RGBA{}) {
		s.GradientFrom = d.GradientFrom
	}
	if s.GradientTo == (color.RGBA{}) {
		s.GradientTo = d.GradientTo
	}
	return s
}

// splitLines разбивает токены по '\n', сохраняя стили.
// Токены с многострочным текстом (комментарии) разрезаются на части.
func splitLines(tokens []highlight.Token) [][]highlight.Token {
	var lines [][]highlight.Token
	var current []highlight.Token

	for _, tok := range tokens {
		parts := strings.Split(tok.Text, "\n")
		for i, part := range parts {
			if part != "" {
				current = append(current, highlight.Token{
					Text:  part,
					Style: tok.Style,
				})
			}
			if i < len(parts)-1 {
				lines = append(lines, current)
				current = nil
			}
		}
	}
	if len(current) > 0 || len(lines) == 0 {
		lines = append(lines, current)
	}
	return lines
}

func measureLine(face font.Face, line []highlight.Token) float64 {
	var total float64
	for _, tok := range line {
		total += measureString(face, tok.Text)
	}
	return total
}

func measureString(face font.Face, s string) float64 {
	return float64(font.MeasureString(face, s).Ceil())
}