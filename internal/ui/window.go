package ui

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/rurewa/codepic-app/internal/app"
	"github.com/rurewa/codepic-app/internal/render"
)

const debounceDelay = 400 * time.Millisecond

type View struct {
	svc    *app.Service
	window fyne.Window

	editor  *widget.Entry
	preview *canvas.Image
	status  *widget.Label

	style       render.Style
	currentLang string
	debounce    *time.Timer

	langSelect  *widget.Select
	themeSelect *widget.Select

	paddingSlider *widget.Slider
	fontSlider    *widget.Slider
	radiusSlider  *widget.Slider

	lineNumsCheck    *widget.Check
	windowDotsCheck  *widget.Check
	shadowCheck      *widget.Check
	gradientCheck    *widget.Check
	transparentCheck *widget.Check

	presetSelect *widget.Select

	themeIDByName map[string]string
	langIDByName  map[string]string
	presetNames   []string
}

func NewView(svc *app.Service, w fyne.Window) *View {
	v := &View{
		svc:           svc,
		window:        w,
		currentLang:   "auto",
		themeIDByName: make(map[string]string),
		langIDByName:  make(map[string]string),
	}

	if cfg, err := svc.Config(); err == nil {
		v.style = cfg.LastStyle
	} else {
		v.style = render.DefaultStyle()
	}

	v.editor = v.buildEditor()
	v.preview = v.buildPreview()
	v.status = widget.NewLabel("Готов")

	sidebar := v.buildSidebar()
	toolbar := v.buildToolbar()

	split := container.NewHSplit(v.editor, v.preview)
	split.SetOffset(0.4)

	body := container.NewBorder(toolbar, v.status, nil, sidebar, split)
	w.SetContent(body)
	w.Resize(fyne.NewSize(1400, 900))

	v.refreshPreview()
	return v
}

func (v *View) buildEditor() *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.SetPlaceHolder("Вставьте код сюда…")
	e.Wrapping = fyne.TextWrapOff
	e.SetMinRowsVisible(20)
	e.OnChanged = func(string) { v.scheduleRender() }
	return e
}

func (v *View) buildPreview() *canvas.Image {
	img := canvas.NewImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(600, 400))
	return img
}

func (v *View) buildToolbar() fyne.CanvasObject {
	langNames := []string{"Авто"}
	v.langIDByName["Авто"] = "auto"
	for _, l := range v.svc.ListLanguages() {
		if l.ID == "plaintext" {
			continue
		}
		langNames = append(langNames, l.Name)
		v.langIDByName[l.Name] = l.ID
	}
	v.langSelect = widget.NewSelect(langNames, func(name string) {
		v.currentLang = v.langIDByName[name]
		v.scheduleRender()
	})
	v.langSelect.SetSelected("Авто")

	var themeNames []string
	var selectedThemeName string
	for _, th := range v.svc.ListThemes() {
		themeNames = append(themeNames, th.Name)
		v.themeIDByName[th.Name] = th.ID
		if th.ID == v.style.ThemeID {
			selectedThemeName = th.Name
		}
	}
	v.themeSelect = widget.NewSelect(themeNames, func(name string) {
		v.style.ThemeID = v.themeIDByName[name]
		v.scheduleRender()
	})
	if selectedThemeName != "" {
		v.themeSelect.SetSelected(selectedThemeName)
	}

	copyBtn := widget.NewButton("Копировать", v.onCopy)
	copyBtn.Importance = widget.HighImportance
	if !v.svc.HasClipboard() {
		copyBtn.Disable()
		copyBtn.SetText("Копировать (нет буфера)")
	}
	saveBtn := widget.NewButton("Сохранить…", v.onSave)

	return container.NewHBox(
		widget.NewLabel("Язык:"), v.langSelect,
		widget.NewLabel("Тема:"), v.themeSelect,
		widget.NewSeparator(),
		copyBtn, saveBtn,
	)
}

func (v *View) buildSidebar() fyne.CanvasObject {
	// --- Пресеты ---
	v.presetSelect = widget.NewSelect([]string{}, func(string) {})
	v.reloadPresets()

	loadPresetBtn := widget.NewButton("Загрузить пресет", v.onLoadPreset)
	savePresetBtn := widget.NewButton("Сохранить как пресет…", v.onSavePreset)
	deletePresetBtn := widget.NewButton("Удалить пресет", v.onDeletePreset)

	presetSection := container.NewVBox(
		widget.NewLabelWithStyle("Пресеты", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		v.presetSelect,
		container.NewGridWithColumns(2, loadPresetBtn, deletePresetBtn),
		savePresetBtn,
		widget.NewSeparator(),
	)

	// --- Слайдеры ---
	v.paddingSlider = widget.NewSlider(16, 128)
	v.paddingSlider.Step = 4
	v.paddingSlider.SetValue(float64(v.style.Padding))
	v.paddingSlider.OnChanged = func(val float64) {
		v.style.Padding = int(val)
		v.scheduleRender()
	}

	v.fontSlider = widget.NewSlider(10, 24)
	v.fontSlider.Step = 1
	v.fontSlider.SetValue(v.style.FontSize)
	v.fontSlider.OnChanged = func(val float64) {
		v.style.FontSize = val
		v.scheduleRender()
	}

	v.radiusSlider = widget.NewSlider(0, 24)
	v.radiusSlider.Step = 1
	v.radiusSlider.SetValue(float64(v.style.CornerRadius))
	v.radiusSlider.OnChanged = func(val float64) {
		v.style.CornerRadius = int(val)
		v.scheduleRender()
	}

	// --- Чекбоксы ---
	v.lineNumsCheck = widget.NewCheck("Номера строк", func(b bool) {
		v.style.ShowLineNumbers = b
		v.scheduleRender()
	})
	v.lineNumsCheck.SetChecked(v.style.ShowLineNumbers)

	v.windowDotsCheck = widget.NewCheck("Три точки", func(b bool) {
		v.style.ShowWindowDots = b
		v.scheduleRender()
	})
	v.windowDotsCheck.SetChecked(v.style.ShowWindowDots)

	v.shadowCheck = widget.NewCheck("Тень", func(b bool) {
		v.style.Shadow = b
		v.scheduleRender()
	})
	v.shadowCheck.SetChecked(v.style.Shadow)

	v.gradientCheck = widget.NewCheck("Градиентный фон", func(b bool) {
		v.style.Gradient = b
		v.scheduleRender()
	})
	v.gradientCheck.SetChecked(v.style.Gradient)

	v.transparentCheck = widget.NewCheck("Прозрачный фон", func(b bool) {
		v.style.Transparent = b
		if b {
			v.gradientCheck.Disable()
		} else {
			v.gradientCheck.Enable()
		}
		v.scheduleRender()
	})
	v.transparentCheck.SetChecked(v.style.Transparent)
	if v.style.Transparent {
		v.gradientCheck.Disable()
	}

	styleSection := container.NewVBox(
		widget.NewLabelWithStyle("Оформление", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Отступ"),
		v.paddingSlider,
		widget.NewLabel("Размер шрифта"),
		v.fontSlider,
		widget.NewLabel("Скругление"),
		v.radiusSlider,
		widget.NewSeparator(),
		v.lineNumsCheck,
		v.windowDotsCheck,
		v.shadowCheck,
		v.gradientCheck,
		v.transparentCheck,
	)

	return container.NewVScroll(container.NewVBox(presetSection, styleSection))
}

// --- Пресеты ---

func (v *View) reloadPresets() {
	cfg, err := v.svc.Config()
	if err != nil {
		return
	}
	v.presetNames = cfg.Names()
	v.presetSelect.Options = v.presetNames
	v.presetSelect.Refresh()
	if len(v.presetNames) == 0 {
		v.presetSelect.ClearSelected()
	}
}

func (v *View) onLoadPreset() {
	name := v.presetSelect.Selected
	if name == "" {
		dialog.ShowInformation("CodePic", "Выберите пресет в списке", v.window)
		return
	}
	cfg, err := v.svc.Config()
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	style, err := cfg.GetPreset(name)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.style = style
	v.applyStyleToUI()
	v.scheduleRender()
	v.status.SetText(fmt.Sprintf("Загружен пресет: %s", name))
}

func (v *View) onSavePreset() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("например, my-dark")

	items := []*widget.FormItem{
		widget.NewFormItem("Имя пресета", nameEntry),
	}

	d := dialog.NewForm("Сохранить пресет", "Сохранить", "Отмена", items, func(ok bool) {
		if !ok {
			return
		}
		name := nameEntry.Text
		if name == "" {
			dialog.ShowInformation("CodePic", "Имя не может быть пустым", v.window)
			return
		}
		if _, err := v.svc.AddPreset(name, v.style); err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.reloadPresets()
		v.presetSelect.SetSelected(name)
		v.status.SetText(fmt.Sprintf("Пресет %q сохранён", name))
	}, v.window)
	d.Resize(fyne.NewSize(500, 200))
	d.Show()
}

func (v *View) onDeletePreset() {
	name := v.presetSelect.Selected
	if name == "" {
		dialog.ShowInformation("CodePic", "Выберите пресет в списке", v.window)
		return
	}
	dialog.ShowConfirm(
		"Удалить пресет",
		fmt.Sprintf("Удалить пресет %q?", name),
		func(ok bool) {
			if !ok {
				return
			}
			if _, err := v.svc.RemovePreset(name); err != nil {
				dialog.ShowError(err, v.window)
				return
			}
			v.reloadPresets()
			v.status.SetText(fmt.Sprintf("Пресет %q удалён", name))
		},
		v.window,
	)
}

func (v *View) applyStyleToUI() {
	for name, id := range v.themeIDByName {
		if id == v.style.ThemeID {
			v.themeSelect.SetSelected(name)
			break
		}
	}
	v.paddingSlider.SetValue(float64(v.style.Padding))
	v.fontSlider.SetValue(v.style.FontSize)
	v.radiusSlider.SetValue(float64(v.style.CornerRadius))
	v.lineNumsCheck.SetChecked(v.style.ShowLineNumbers)
	v.windowDotsCheck.SetChecked(v.style.ShowWindowDots)
	v.shadowCheck.SetChecked(v.style.Shadow)
	v.gradientCheck.SetChecked(v.style.Gradient)
	v.transparentCheck.SetChecked(v.style.Transparent)
	if v.style.Transparent {
		v.gradientCheck.Disable()
	} else {
		v.gradientCheck.Enable()
	}
}

// --- Рендер ---

func (v *View) scheduleRender() {
	if v.debounce != nil {
		v.debounce.Stop()
	}
	v.debounce = time.AfterFunc(debounceDelay, func() {
		fyne.Do(func() { v.refreshPreview() })
	})
}

func (v *View) refreshPreview() {
	code := v.editor.Text
	if code == "" {
		v.status.SetText("Введите код для превью")
		return
	}
	img, err := v.svc.RenderPreview(code, v.currentLang, v.style)
	if err != nil {
		v.status.SetText(fmt.Sprintf("Ошибка рендера: %v", err))
		return
	}
	v.preview.Image = img
	v.preview.Refresh()
	v.status.SetText(fmt.Sprintf("Lang: %s  •  Theme: %s", v.svc.DetectLanguage(code), v.style.ThemeID))
}

// --- Действия ---

func (v *View) onCopy() {
	if !v.svc.HasClipboard() {
		dialog.ShowInformation("CodePic", "Буфер обмена недоступен в этом окружении", v.window)
		return
	}
	if err := v.svc.CopyToClipboard(v.editor.Text, v.currentLang, v.style); err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	dialog.ShowInformation("CodePic", "Изображение скопировано в буфер обмена", v.window)
}

func (v *View) onSave() {
	home, _ := os.UserHomeDir()
	startURI := storage.NewFileURI(home)
	startLister, _ := storage.ListerForURI(startURI)

	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer w.Close()
		img, rerr := v.svc.RenderPreview(v.editor.Text, v.currentLang, v.style)
		if rerr != nil {
			dialog.ShowError(rerr, v.window)
			return
		}
		if encErr := png.Encode(w, img); encErr != nil {
			dialog.ShowError(encErr, v.window)
			return
		}
		v.status.SetText("Сохранено: " + w.URI().Path())
	}, v.window)

	d.SetFileName("codepic.png")
	if startLister != nil {
		d.SetLocation(startLister)
	}
	d.Resize(fyne.NewSize(900, 600))
	d.Show()
}