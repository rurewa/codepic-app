package clipboard

import "image"

// NewStub возвращает Writer, который ничего не делает.
// Используется в тестах и в CLI-режиме без CGO.
func NewStub() Writer { return stubWriter{} }

type stubWriter struct{}

func (stubWriter) WriteImage(image.Image) error { return nil }
func (stubWriter) WriteText(string) error       { return nil }