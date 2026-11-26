package config

import (
	"fmt"

	"github.com/rurewa/codepic-app/internal/render"
)

// Names возвращает имена пресетов в порядке добавления.
func (c Config) Names() []string {
	out := make([]string, len(c.Presets))
	for i, p := range c.Presets {
		out[i] = p.Name
	}
	return out
}

// GetPreset ищет пресет по имени.
func (c Config) GetPreset(name string) (render.Style, error) {
	for _, p := range c.Presets {
		if p.Name == name {
			return p.Style, nil
		}
	}
	return render.Style{}, fmt.Errorf("config: preset %q not found", name)
}

// AddPreset добавляет новый пресет или заменяет существующий с тем же именем.
// Возвращает обновлённый Config (value-семантика — удобно для тестов).
func (c Config) AddPreset(name string, style render.Style) (Config, error) {
	if name == "" {
		return c, fmt.Errorf("config: preset name must not be empty")
	}
	for i, p := range c.Presets {
		if p.Name == name {
			c.Presets[i].Style = style
			return c, nil
		}
	}
	c.Presets = append(c.Presets, Preset{Name: name, Style: style})
	return c, nil
}

// RemovePreset удаляет пресет по имени.
func (c Config) RemovePreset(name string) (Config, error) {
	for i, p := range c.Presets {
		if p.Name == name {
			c.Presets = append(c.Presets[:i], c.Presets[i+1:]...)
			return c, nil
		}
	}
	return c, fmt.Errorf("config: preset %q not found", name)
}