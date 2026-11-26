package app

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/rurewa/codepic-app/internal/clipboard"
	"github.com/rurewa/codepic-app/internal/config"
	"github.com/rurewa/codepic-app/internal/highlight"
	"github.com/rurewa/codepic-app/internal/render"
	"github.com/rurewa/codepic-app/internal/theme"
)

// spyClipboard записывает факт вызова WriteImage.
type spyClipboard struct {
	imageWritten bool
	lastImage    image.Image
}

func (s *spyClipboard) WriteImage(img image.Image) error {
	s.imageWritten = true
	s.lastImage = img
	return nil
}
func (s *spyClipboard) WriteText(string) error { return nil }

// newTestService собирает сервис с настоящими highlight/theme/render
// и подставным clipboard + временным store.
func newTestService(t *testing.T) (*Service, *spyClipboard, config.Store) {
	t.Helper()
	store := config.NewFileStoreAt(filepath.Join(t.TempDir(), "config.json"))
	spy := &spyClipboard{}
	return NewWithDeps(
		highlight.NewChroma(),
		theme.Builtin(),
		render.New(),
		spy,
		store,
	), spy, store
}

const sample = `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`

func TestRenderPreview(t *testing.T) {
	svc, _, _ := newTestService(t)
	style := render.DefaultStyle()
	style.ThemeID = "dracula"

	img, err := svc.RenderPreview(sample, "go", style)
	if err != nil {
		t.Fatalf("RenderPreview: %v", err)
	}
	if img == nil {
		t.Fatal("nil image")
	}
	b := img.Bounds()
	if b.Dx() < 100 || b.Dy() < 100 {
		t.Errorf("image too small: %dx%d", b.Dx(), b.Dy())
	}
}

func TestRenderPreview_UnknownTheme(t *testing.T) {
	svc, _, _ := newTestService(t)
	style := render.DefaultStyle()
	style.ThemeID = "definitely-not-a-theme"

	if _, err := svc.RenderPreview(sample, "go", style); err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestCopyToClipboard(t *testing.T) {
	svc, spy, _ := newTestService(t)
	if err := svc.CopyToClipboard(sample, "go", render.DefaultStyle()); err != nil {
		t.Fatalf("CopyToClipboard: %v", err)
	}
	if !spy.imageWritten {
		t.Error("WriteImage was not called")
	}
}

func TestCopyToClipboard_NoClipboard(t *testing.T) {
	svc, _, _ := newTestService(t)
	svc.Clipboard = nil

	err := svc.CopyToClipboard(sample, "go", render.DefaultStyle())
	if !errors.Is(err, ErrNoClipboard) {
		t.Fatalf("err = %v, want ErrNoClipboard", err)
	}
}

func TestSavePNG(t *testing.T) {
	svc, _, _ := newTestService(t)
	out := filepath.Join(t.TempDir(), "out.png")

	if err := svc.SavePNG(out, sample, "go", render.DefaultStyle()); err != nil {
		t.Fatalf("SavePNG: %v", err)
	}
	st, err := os.Stat(out)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Size() == 0 {
		t.Error("empty png")
	}
}

func TestDetectLanguage(t *testing.T) {
	svc, _, _ := newTestService(t)
	got := svc.DetectLanguage(sample)
	if got != "go" {
		t.Errorf("DetectLanguage = %q, want go", got)
	}
}

func TestListThemesAndLanguages(t *testing.T) {
	svc, _, _ := newTestService(t)
	if len(svc.ListThemes()) == 0 {
		t.Error("no themes")
	}
	if len(svc.ListLanguages()) == 0 {
		t.Error("no languages")
	}
}

func TestAddPreset_Persists(t *testing.T) {
	svc, _, store := newTestService(t)

	style := render.DefaultStyle()
	style.ThemeID = "nord"

	if _, err := svc.AddPreset("my-nord", style); err != nil {
		t.Fatalf("AddPreset: %v", err)
	}

	// Перечитываем конфиг из хранилища — пресет должен сохраниться.
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := cfg.GetPreset("my-nord")
	if err != nil {
		t.Fatalf("GetPreset: %v", err)
	}
	if got.ThemeID != "nord" {
		t.Errorf("theme = %q, want nord", got.ThemeID)
	}
}

func TestRemovePreset(t *testing.T) {
	svc, _, _ := newTestService(t)
	style := render.DefaultStyle()
	_, _ = svc.AddPreset("x", style)

	if _, err := svc.RemovePreset("x"); err != nil {
		t.Fatalf("RemovePreset: %v", err)
	}
	cfg, _ := svc.Config()
	if len(cfg.Presets) != 0 {
		t.Errorf("presets = %d, want 0", len(cfg.Presets))
	}
}

// Гарантируем, что интерфейс clipboard.Writer реализуется spy.
var _ clipboard.Writer = (*spyClipboard)(nil)