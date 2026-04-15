// gloss/glossary.go
package gloss

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ShowGlossary открывает окно с глоссарием по нажатию F1
func ShowGlossary(w fyne.Window) {
	glossaryWindow := fyne.CurrentApp().NewWindow("Глоссарий — GitTornado")

	glossaryWindow.SetFixedSize(false)
	glossaryWindow.Resize(fyne.NewSize(780, 580))

	content := widget.NewRichText(
		&widget.TextSegment{
			Text: `Глоссарий программы GitTornado

Горячие клавиши:
• Ctrl + Shift + F — Открыть папку пути загрузки.
• Ctrl + Shift + Escape — Свернуть окно в трей.
• Ctrl + Shift + F — Открыть глоссарий.

Поля ввода:
• Логин — Поле ввода логина configs.net.rt.ru.
• Пароль — Поле ввода пароль configs.net.rt.ru.
• NetBox API — Поле ввода API-ключа netbox.rt.ru. (https://netbox.rt.ru/user/api-tokens/)
• Путь загрузки - Поле ввода пути загрузки файлов конфигураций из configs.net.rt.ru.
		По-умолчанию текущая папка/папка расположения GitTornado.
		Кнопка "Обзор" позволяет выбрать папку через браузер файловой структуры ОС.
• Интервал дней - Поле ввода интервала дней загрузки файлов из configs.net.rt.ru. 
		Минимум - 1 день (каждый день). Максимум - 31 день (интервал/каждые 31 день/месяц)
		По-умолчанию 1 день
• Время загрузки - Поле ввода времени загрузки файлов из configs.net.rt.ru в формате ЧЧ:ММ.
		По-умолчанию 00:00 (Pагрузка в 0 часов 0 минут).

Чекбоксы:
• Сортировка: "Как есть" - файлы будут загружены из configs.net.rt.ru в оригинальном виде.
• Сортировка: "Платформа" - файлы будут загружены и отсортированы согласно их платформе/вендору:
		Cisco IOS/CIsco ASA/Huwei VRP/...
• Сортировка: Регион" - файлы будут загружены из configs.net.rt.ru и отсортированы согласно МРФ:
		Юг/Волга/КЦ/...
• Сортировка: "ACL" - файлы будут загружены и отсортированы согласно наличию в них ACL-листов.
		Файлы вендоров не имеющих функцию ACL будут проигнорированы. (Балансировщики/Консольные коммутаторы/...)
		+Parser: Активируется только при выборе "ACL" . Загружает программу ACL Parser.
• Режим работы: "Обновление" - файлы будут загружены из configs.net.rt.ru и перезаписаны по расписанию.
		Папка загрузки "сonfig_files_clear".
• Режим работы: "Прогресс" - файлы будут загружены из configs.net.rt.ru в собственную папку на момент загрузки.
		Создаётся структура папок по дате загрузки: 01-01-26/02-01-26/03-01-26/...
		Папка загрузки "сonfig_files_clear" будет размещена внутри структуры.
• Цель: "ЦОД" - будут загружены файлы конфигураций УЭСЦОД.
• Цель: "ЛВС" - будут загружены файлы конфигураций УЭСЛВС.


Кнопки:
• "Скачать" - запускает процесс загрузки. Можно использовать без сохранения настроек или принудительно.
• "Сбросить" - сбрасывает настройки программы.
• "Старт\Пауза" - запускает\ставит на паузу планировщик.

Общие сведения:
После сохранения настроек все поля ввода данных блокируются.
	Настройки сохраняются в файл на ПК под управлением АльтЛинукс. 
	Все чувствительные данные зашифрованы.
	После сброса настроек данные уничтожаются.
Программа сворачивается в трей (оперативную память) и работает оттуда по расписанию.
	Для выхода из программы необходимо кликнуть ПКМ по пиктограмме программы (Торнадо) и
    	нажать "Выход".

`,
			//Style: widget.RichTextStyleNormal,   // обычный текст
		},
	)

	//content = widget.NewRichText(
	//	&widget.TextSegment{
	//		Text:  "Глоссарий программы GitTornado\n\n",
	//		Style: widget.RichTextStyleHeading,
	//	},
	//	&widget.TextSegment{
	//		Text:  "Горячие клавиши:\n",
	//		Style: widget.RichTextStyleSubHeading,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Ctrl + Shift + F (или Ctrl + Shift + А) — Открыть папку заданную в пути\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Ctrl + Shift + Escape — Свернуть окно в трей\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Ctrl + Shift + F1 — Открыть глоссарий\n\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//
	//	&widget.TextSegment{
	//		Text:  "Основные функции:\n",
	//		Style: widget.RichTextStyleSubHeading,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Кнопка Скачать — Принудительная загрузка конфигураций с GitLab\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Сохранить — Сохранить текущие настройки в config.json\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Сбросить — Полный сброс настроек и удаление папки ~/.config/gittornado\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Старт / Пауза — Управление планировщиком автоматической загрузки\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//
	//	&widget.TextSegment{
	//		Text:  "\nРежимы работы:\n",
	//		Style: widget.RichTextStyleSubHeading,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Как есть — Копирует файлы без сортировки\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• По платформам — Сортирует по типу оборудования (через NetBox)\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• По регионам — Сортирует по географическим регионам\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//	&widget.TextSegment{
	//		Text: "• Access-lists — Сортирует только указанные платформы ACL\n",
	//		//Style: widget.RichTextStyleNormal,
	//	},
	//)

	glossaryWindow.SetContent(container.NewVScroll(content))
	glossaryWindow.CenterOnScreen()
	glossaryWindow.Show()
}
