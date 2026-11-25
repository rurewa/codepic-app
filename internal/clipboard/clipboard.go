package clipboard

import "image"

// Writer — абстракция над системным буфером обмена.
// Реализация — в system.go (CGO), для тестов — stub.go.
type Writer interface {
	WriteImage(img image.Image) error
	WriteText(s string) error
}