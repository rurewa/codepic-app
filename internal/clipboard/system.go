package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"

	xclip "golang.design/x/clipboard"
)

func New() (Writer, error) {
	if err := xclip.Init(); err != nil {
		return nil, fmt.Errorf("clipboard: init: %w", err)
	}
	return systemWriter{}, nil
}

type systemWriter struct{}

func (systemWriter) WriteImage(img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("clipboard: encode png: %w", err)
	}

	// Новый API: context первым, error последним.
	// Возвращаемый канал сигнализирует, что запись устарела (её перезаписали);
	// для разового копирования его можно игнорировать.
	_, err := xclip.Write(context.Background(), xclip.FmtImage, buf.Bytes())
	if err != nil {
		return fmt.Errorf("clipboard: write image: %w", err)
	}
	return nil
}

func (systemWriter) WriteText(s string) error {
	_, err := xclip.Write(context.Background(), xclip.FmtText, []byte(s))
	if err != nil {
		return fmt.Errorf("clipboard: write text: %w", err)
	}
	return nil
}