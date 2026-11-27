package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rurewa/codepic-app/internal/render"
)

// Version — версия схемы конфига. При breaking changes увеличиваем
// и добавляем миграцию в Load.
const Version = 1

// Preset — именованный набор настроек оформления.
type Preset struct {
	Name  string       `json:"name"`
	Style render.Style `json:"style"`
}

// Config — всё, что приложение сохраняет между запусками.
type Config struct {
	Version      int          `json:"version"`
	LastLanguage string       `json:"last_language"`
	LastPreset   string       `json:"last_preset"`
	LastStyle    render.Style `json:"last_style"`
	Presets      []Preset     `json:"presets"`
	WindowWidth  int          `json:"window_width"`
	WindowHeight int          `json:"window_height"`
}

// DefaultConfig — значения при первом запуске.
func DefaultConfig() Config {
	return Config{
		Version:      Version,
		LastLanguage: "auto",
		LastPreset:   "",
		LastStyle:    render.DefaultStyle(),
		Presets:      nil,
		WindowWidth:  1200,
		WindowHeight: 800,
	}
}

// Store — абстракция над хранилищем конфига.
// Реализация — FileStore; для тестов можно подставить in-memory.
type Store interface {
	Load() (Config, error)
	Save(Config) error
	Path() string
}

// FileStore хранит конфиг в JSON-файле
// (~/.config/codepic/config.json на Linux, %AppData%\codepic\config.json на Windows).
type FileStore struct {
	path string
}

// NewFileStore создаёт хранилище в стандартной директории конфигов.
func NewFileStore() (*FileStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("config: user config dir: %w", err)
	}
	return &FileStore{path: filepath.Join(dir, "codepic", "config.json")}, nil
}

// NewFileStoreAt — для тестов: явный путь.
func NewFileStoreAt(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Path() string { return s.path }

// Load читает конфиг.
//   - файла нет        → (DefaultConfig(), nil)
//   - файл битый       → (DefaultConfig(), error)
//   - версия новее     → (DefaultConfig(), error)
func (s *FileStore) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return DefaultConfig(), fmt.Errorf("config: read: %w", err)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return DefaultConfig(), fmt.Errorf("config: parse %s: %w", s.path, err)
	}

	if c.Version > Version {
		return DefaultConfig(), fmt.Errorf(
			"config: %s has version %d, this build supports %d — upgrade CodePic",
			s.path, c.Version, Version,
		)
	}

	if c.LastLanguage == "" {
		c.LastLanguage = "auto"
	}
	if c.LastStyle == (render.Style{}) {
		c.LastStyle = render.DefaultStyle()
	}
	return c, nil
}

// Save пишет конфиг атомарно: tmp-файл рядом + rename.
func (s *FileStore) Save(c Config) error {
	c.Version = Version

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "config-*.json.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(c); err != nil {
		tmp.Close()
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		return fmt.Errorf("config: rename: %w", err)
	}
	return nil
}