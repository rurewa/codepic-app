package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"

	xclip "golang.design/x/clipboard"
)

// New инициализирует системный буфер обмена.
//
// Ошибка инициализации библиотеки не фатальна: на Wayland основной путь —
// wl-copy, а xclip.Init() нужен только как fallback для X11 или data-control.
func New() (Writer, error) {
	if err := xclip.Init(); err != nil {
		// Не возвращаем ошибку: возможно, wl-copy справится.
		fmt.Fprintf(os.Stderr, "clipboard: init warning: %v\n", err)
	}
	return systemWriter{}, nil
}

type systemWriter struct{}

// WriteImage кодирует изображение в PNG и помещает в системный буфер.
//
// Порядок:
//  1. Wayland + wl-copy → основной путь (надёжен в KDE/GNOME/Sway).
//  2. golang.design/x/clipboard → fallback для X11 и Wayland с data-control.
func (systemWriter) WriteImage(img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("clipboard: encode png: %w", err)
	}

	// 1. Wayland: wl-copy.
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if path, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command(path, "--type", "image/png")
			cmd.Stdin = bytes.NewReader(buf.Bytes())
			// ВАЖНО: не использовать CombinedOutput/Output — они создают pipe,
			// который наследует форкнутый wl-copy и блокирует Wait() навсегда.
			// nil → /dev/null, os.Stderr → прямой fd без pipe.
			cmd.Stdout = nil
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "clipboard: wl-copy failed: %v\n", err)
				// Падать не будем — попробуем библиотеку как fallback.
			} else {
				return nil
			}
		}
	}

	// 2. Fallback: golang.design/x/clipboard.
	_, err := xclip.Write(context.Background(), xclip.FmtImage, buf.Bytes())
	if err != nil {
		return fmt.Errorf("clipboard: write image: %w", err)
	}
	return nil
}

// WriteText помещает текст в буфер обмена.
func (systemWriter) WriteText(s string) error {
	_, err := xclip.Write(context.Background(), xclip.FmtText, []byte(s))
	if err != nil {
		return fmt.Errorf("clipboard: write text: %w", err)
	}
	return nil
}