package app

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/rurewa/codepic-app/internal/clipboard"
	"github.com/rurewa/codepic-app/internal/config"
	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/render"
	"github.com/rurewa/codepic-app/internal/theme"
)

// ErrNoClipboard — буфер обмена недоступен (TTY, нет X11/Wayland,
// отсутствует wl-copy). GUI должен дизейблить кнопку «Копировать»,
// CLI — печатать warning и сохранять PNG.
var ErrNoClipboard = errors.New("app: clipboard unavailable in this environment")

// Service — единая точка входа в бизнес-логику приложения.
// UI и CLI зависят ТОЛЬКО от этого пакета, а не от highlight/render/... напрямую.
type Service struct {
	Highlighter highlight.Highlighter // для DetectLanguage и Languages (без стиля)
	Themes      theme.Provider
	Renderer    render.Renderer
	Clipboard   clipboard.Writer // может быть nil, если недоступен
	Store       config.Store
}

// New собирает сервис с дефолтными реализациями.
// Если буфер обмена недоступен — это НЕ ошибка: сервис создаётся,
// но методы CopyToClipboard вернут ErrNoClipboard.
func New() (*Service, error) {
	store, err := config.NewFileStore()
	if err != nil {
		return nil, fmt.Errorf("app: config store: %w", err)
	}

	svc := &Service{
		Highlighter: highlight.NewChroma(),
		Themes:      theme.Builtin(),
		Renderer:    render.New(),
		Store:       store,
	}

	// Инициализация буфера обмена не критична.
	if cb, err := clipboard.New(); err == nil {
		svc.Clipboard = cb
	}

	return svc, nil
}

// NewWithDeps — для тестов: подставить всё явно.
func NewWithDeps(
	h highlight.Highlighter,
	th theme.Provider,
	r render.Renderer,
	cb clipboard.Writer,
	store config.Store,
) *Service {
	return &Service{
		Highlighter: h,
		Themes:      th,
		Renderer:    r,
		Clipboard:   cb,
		Store:       store,
	}
}

// HasClipboard сообщает, доступен ли буфер обмена.
// GUI использует для дизейбла кнопки.
func (s *Service) HasClipboard() bool { return s.Clipboard != nil }

// RenderPreview — главный метод: код + язык + стиль → картинка.
// Создаёт highlighter со стилем выбранной темы, чтобы цвета совпадали.
func (s *Service) RenderPreview(code, lang string, style render.Style) (image.Image, error) {
	th, err := s.Themes.Get(style.ThemeID)
	if err != nil {
		return nil, err
	}
	h := highlight.NewChromaWithStyle(th.ChromaStyle)
	tokens, err := h.Highlight(code, lang)
	if err != nil {
		return nil, fmt.Errorf("app: highlight: %w", err)
	}
	img, err := s.Renderer.Render(tokens, th, style)
	if err != nil {
		return nil, fmt.Errorf("app: render: %w", err)
	}
	return img, nil
}

// CopyToClipboard рендерит и копирует PNG в буфер.
func (s *Service) CopyToClipboard(code, lang string, style render.Style) error {
	if s.Clipboard == nil {
		return ErrNoClipboard
	}
	img, err := s.RenderPreview(code, lang, style)
	if err != nil {
		return err
	}
	if err := s.Clipboard.WriteImage(img); err != nil {
		return fmt.Errorf("app: clipboard write: %w", err)
	}
	return nil
}

// SavePNG рендерит и сохраняет PNG по указанному пути.
func (s *Service) SavePNG(path, code, lang string, style render.Style) error {
	img, err := s.RenderPreview(code, lang, style)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("app: create %s: %w", path, err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("app: encode png: %w", err)
	}
	return nil
}

// DetectLanguage возвращает ID языка (для подсветки текущего кода в UI).
func (s *Service) DetectLanguage(code string) string {
	return s.Highlighter.DetectLanguage(code)
}

// ListThemes — для UI-селектора.
func (s *Service) ListThemes() []theme.Theme { return s.Themes.List() }

// ListLanguages — для UI-селектора.
func (s *Service) ListLanguages() []highlight.Language {
	return s.Highlighter.Languages()
}

// GetTheme — обёртка для удобства UI (получить по ID с ошибкой).
func (s *Service) GetTheme(id string) (theme.Theme, error) { return s.Themes.Get(id) }

// --- Конфиг и пресеты ---

// Config загружает текущий конфиг.
func (s *Service) Config() (config.Config, error) { return s.Store.Load() }

// SaveConfig сохраняет конфиг.
func (s *Service) SaveConfig(cfg config.Config) error { return s.Store.Save(cfg) }

// AddPreset — load-modify-save: добавляет пресет и сохраняет конфиг.
func (s *Service) AddPreset(name string, style render.Style) (config.Config, error) {
	cfg, err := s.Store.Load()
	if err != nil {
		return cfg, err
	}
	cfg, err = cfg.AddPreset(name, style)
	if err != nil {
		return cfg, err
	}
	if err := s.Store.Save(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// RemovePreset — load-modify-save.
func (s *Service) RemovePreset(name string) (config.Config, error) {
	cfg, err := s.Store.Load()
	if err != nil {
		return cfg, err
	}
	cfg, err = cfg.RemovePreset(name)
	if err != nil {
		return cfg, err
	}
	if err := s.Store.Save(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}