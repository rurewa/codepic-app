package main

import (
	"log"

	fyneapp "fyne.io/fyne/v2/app"

	"fyne.io/fyne/v2"

	"github.com/rurewa/codepic-app/internal/app"
	"github.com/rurewa/codepic-app/internal/ui"
)

func main() {
	// 1. Наш сервис: highlight + render + config + clipboard.
	svc, err := app.New()
	if err != nil {
		log.Fatalf("codepic: init service: %v", err)
	}

	// 2. Fyne-приложение. App ID = обратный домен GitHub Pages:
	//    используется для хранения настроек окна, позиции и т.п.
	a := fyneapp.NewWithID("io.github.rurewa.codepic")

	// 3. Главное окно.
	w := a.NewWindow("CodePic")
	w.SetMaster() // закрытие этого окна = выход из приложения

	// 4. Собираем UI.
	ui.NewView(svc, w)

	// 5. Показываем окно и запускаем event loop (блокируется до закрытия).
	w.ShowAndRun()

	// Сюда попадём только после закрытия окна.
	// Сюда можно добавить финализацию (сохранение конфига и т.п.), если понадобится.
	_ = fyne.CurrentApp()
}