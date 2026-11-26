package ui

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/rurewa/codepic-app/internal/app"
	"github.com/rurewa/codepic-app/internal/render"
)

// View — минимальный GUI: редактор, кнопка «Обновить», превью.
type View struct {
	svc    *app.Service
	window fyne.Window

	editor  *widget.Entry
	preview *canvas.Image
	style   render.Style // текущий стиль (пока просто из конфига)
}

// NewView собирает окно и возвращает его.
func NewView(svc *app.Service, w fyne.Window) *View {
	v := &View{
		svc:    svc,
		window: w,
	}

	// 1. Поле ввода — многострочное, с прокруткой.
	v.editor = widget.NewMultiLineEntry()
	v.editor.SetPlaceHolder("Вставьте код сюда…")
	v.editor.Wrapping = fyne.TextWrapOff
	v.editor.SetMinRowsVisible(20) // не сжимается меньше 20 строк

	// 2. Превью — пустая картинка-заглушка.
	v.preview = canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	v.preview.FillMode = canvas.ImageFillContain // сохранять пропорции
	v.preview.SetMinSize(fyne.NewSize(400, 300))

	// 3. Кнопка «Обновить превью».
	renderBtn := widget.NewButton("Обновить превью", func() {
		v.refreshPreview()
	})
	renderBtn.Importance = widget.HighImportance

	// 4. Загружаем последний стиль из конфига.
	cfg, err := svc.Config()
	if err == nil {
		v.style = cfg.LastStyle
	} else {
		v.style = render.DefaultStyle()
	}

	// 5. Компоновка: слева редактор, справа превью, сверху — кнопка.
	split := container.NewHSplit(v.editor, v.preview)
	split.SetOffset(0.5) // 50/50

	content := container.NewBorder(
		container.NewVBox(renderBtn), // top
		nil,                          // bottom
		nil,                          // left
		nil,                          // right
		split,                        // center
	)
	w.SetContent(content)
	w.Resize(fyne.NewSize(1200, 800))

	return v
}

// refreshPreview — синхронный вызов рендера (дебаунс добавим позже).
func (v *View) refreshPreview() {
	code := v.editor.Text
	if code == "" {
		return
	}

	img, err := v.svc.RenderPreview(code, "auto", v.style)
	if err != nil {
		// Пока просто логируем; позже покажем диалог.
		fyne.LogError("render preview", err)
		return
	}

	v.preview.Image = img
	v.preview.Refresh() // без этого Fyne не перерисует картинку
}