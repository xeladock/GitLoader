// gui/tray.go — РАБОТАЕТ НА ЛЮБОЙ ВЕРСИИ FYNE
package window_action

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func SetupTray(app fyne.App, window fyne.Window) {
	if desk, ok := app.(desktop.App); ok {
		// ← Создаём меню с одним пунктом "Выход" — он заменит стандартный
		menu := fyne.NewMenu("ConfigTool",
			fyne.NewMenuItem("Показать", func() {
				window.Show()
			}),
			fyne.NewMenuItemSeparator(),
		)

		// ← Устанавливаем наше меню
		desk.SetSystemTrayMenu(menu)
	}

	// Сворачивание в трей по крестику
	window.SetCloseIntercept(func() {
		window.Hide()
	})
}
