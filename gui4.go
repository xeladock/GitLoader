package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"configtool.local/conf"
	"configtool.local/progdl"
	"fyne.io/fyne/v2/theme"
	//"configtool.local/start_stop"
	// "configtool.local/window_action" // removed, using systray instead
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/getlantern/systray"
	//"github.com/tadvi/systray"
)

//go:embed icon/icon.png

var trayIcon []byte
var isAutoRun = false // ← флаг: запущено ли по расписанию
//var skipManualDialog = false
//
//func clearLogKeepHeader(textBinding binding.String, entry *widget.Entry) {
//	current, _ := textBinding.Get()
//	lines := strings.Split(current, "\n")
//	for len(lines) > 0 && lines[len(lines)-1] == "" {
//		lines = lines[:len(lines)-1]
//	}
//
//	// Оставляем ТОЛЬКО последние 2 строки (заголовок)
//	if len(lines) > 2 {
//		header := strings.Join(lines[len(lines)-2:], "\n") + "\n\n"
//		textBinding.Set(header)
//	} else if len(lines) > 0 {
//		// Если меньше 2 строк — оставляем как есть, но с переносами
//		textBinding.Set(strings.Join(lines, "\n") + "\n\n")
//	}
//
//	// ← ПРОСТО ВЫЗЫВАЕМ Refresh() — без type assertion!
//	entry.Refresh()
//}

// Вызов:

func isValidTime(s string) bool {
	if len(s) != 5 {
		return false
	}
	if s[2] != ':' {
		return false
	}
	hour := s[:2]
	minute := s[3:]
	h, errH := strconv.Atoi(hour)
	m, errM := strconv.Atoi(minute)
	return errH == nil && errM == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
}

//	func copyDir(src, dst string) error {
//		return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
//			if err != nil {
//				return err
//			}
//			rel, _ := filepath.Rel(src, path)
//			target := filepath.Join(dst, rel)
//
//			if info.IsDir() {
//				return os.MkdirAll(target, info.Mode())
//			}
//			data, err := os.ReadFile(path)
//			if err != nil {
//				return err
//			}
//			return os.WriteFile(target, data, info.Mode())
//		})
//	}

func checkModeSelected(asIsCheck, platformCheck, regionCheck *widget.Check, w fyne.Window) bool {
	if !asIsCheck.Checked && !platformCheck.Checked && !regionCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите режим сортировки.", w)
		return false
	}
	return true
}

func passModeSelected(updateCheck, progressCheck *widget.Check, w fyne.Window) bool {
	if !updateCheck.Checked && !progressCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите режим сохранения.", w)
		//fyne.CurrentApp().Driver().RunOnMain()
		return false
	}
	return true
}

func targetModeSelected(dcCheck, lanCheck *widget.Check, w fyne.Window) bool {
	if !dcCheck.Checked && !lanCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите цель загрузки.", w)
		return false
	}
	return true
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // ошибка или файла нет
	}
	return !info.IsDir() // существует и это файл
}

func appendOutput(bindStr binding.String, msg string) error {
	current, err := bindStr.Get()
	if err != nil {
		return fmt.Errorf("ошибка чтения binding.String: %w", err)
	}

	if err := bindStr.Set(current + msg + "\n"); err != nil {
		return fmt.Errorf("ошибка записи binding.String: %w", err)
	}

	return nil
}

func RemoveGitFolder(dir string, output binding.String) error {
	gitPath := filepath.Join(dir, ".git")

	// Проверяем, существует ли .git
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		//appendOutput(output, "Папка .git не найдена (уже удалена или clone прошёл без неё).\n")
		return nil
	}

	// Удаляем полностью
	if err := os.RemoveAll(gitPath); err != nil {
		appendOutput(output, fmt.Sprintf("Ошибка удаления .git: %v\n", err))
		return err
	}
	//time.Sleep(1)
	//appendOutput(output, "Папка .git удалена.\n")
	return nil
}

func saveConfig(cfg *config.AppConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
func loadConfig(path string) (*config.AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg := &config.AppConfig{}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// для остановки

//type AppConfig struct {
//	GitLabLogin    string `json:"gitlab_login"`
//	GitLabPass     string `json:"gitlab_pass_hash"`
//	NetboxToken    string `json:"netbox_token_hash"`
//	Mode           string `json:"mode"` // as-is / platform / region
//	SaveMode       string `json:"save_mode"`
//	ScheduleDays   int    `json:"schedule_days"`
//	ScheduleTime   string `json:"schedule_time"` // "HH:MM"
//	LastRun        string `json:"last_run,omitempty"`
//	SchedulerState string `json:"scheduler_state"`
//}

type ReadOnlyEntry struct {
	widget.Entry
}

func NewReadOnlyEntry() *ReadOnlyEntry {
	e := &ReadOnlyEntry{}
	e.ExtendBaseWidget(e)
	return e
}

//func (e *ReadOnlyEntry) Refresh() {
//	e.Entry.Refresh()
//}

// ❗ Полностью блокируем ввод с клавиатуры
func (e *ReadOnlyEntry) TypedRune(r rune)           {}
func (e *ReadOnlyEntry) TypedKey(ev *fyne.KeyEvent) {}

//type CleanLightTheme struct{}

//	func (CleanLightTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
//		if n == theme.ColorNameInputBackground || n == theme.ColorNameDisabled {
//			return color.White // белый фон у полей и отключённых полей
//		}
//		if n == theme.ColorNameDisabled {
//			return color.NRGBA{80, 80, 80, 255} // тёмно-серый текст (читаемо!)
//		}
//		return theme.LightTheme{}.Color(n, v)
//	}
type hiddenTheme struct{ fyne.Theme }

func (hiddenTheme) ScrollBarSize() int { return 0 }
func main() {

	const configPath = "config.json"

	var cfg *config.AppConfig
	cfg, err := loadConfig(configPath)
	if err != nil {
		// файла нет — просим пользователя ввести данные
		cfg = &config.AppConfig{}
	}

	targetDir := "./configs" // куда клонируем репо
	sortedDst := "./config_files_clear"

	a := app.NewWithID("rt_gitloader")
	a.Settings().SetTheme(theme.LightTheme())
	//a.Settings().SetTheme(hiddenTheme{})
	// Set Fyne app icon as well (optional)
	a.SetIcon(fyne.NewStaticResource("icon.png", trayIcon))

	w := a.NewWindow("GitLab Downloader")
	w.Resize(fyne.NewSize(600, 250))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	// Сворачивание в трей по крестику
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// === SYSTRAY (getlantern) ===
	go func() {
		systray.Run(func() {
			if len(trayIcon) > 0 {
				systray.SetIcon(trayIcon)
				//systray.SetTooltip("GitLab Downloader v0.71")
			}
			//systray.SetTooltip("GitLab Downloader")
			systray.SetTitle("GitLab Downloader")
			systray.SetTooltip("GitLab Downloader v1.0")

			open := systray.AddMenuItem("Открыть", "Показать программу")
			quit := systray.AddMenuItem("Выход", "Закрыть программу")

			go func() {
				for range open.ClickedCh {
					fyne.Do(func() {
						w.Show()
						w.RequestFocus()
					})
				}
			}()

			go func() {
				for range quit.ClickedCh {
					// ← Всё, что касается Fyne — в UI-потоке
					fyne.DoAndWait(func() {

						w.Close() // ← закрываем окно
						if err := os.RemoveAll(targetDir); err != nil {
						}
						//appendOutput(output, fmt.Sprintf("Ошибка удаления .git: %v\n", err))

						a.Quit() // ← завершаем приложение (БЕЗ ОШИБКИ!)
					})

					systray.Quit() // ← это можно вне UI-потока
					// os.Exit(0) — НЕ НУЖНО! a.Quit() уже завершает приложение
				}
			}()
		}, func() {})
	}()

	// === END SYSTRAY ===

	// Поля ввода

	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("Введите логин GitLab")
	loginEntry.TextStyle = fyne.TextStyle{}
	passEntry := widget.NewEntry()
	passEntry.Password = true
	passEntry.SetPlaceHolder("Введите пароль GitLab")
	passEntry.TextStyle = fyne.TextStyle{}

	netboxEntry := widget.NewEntry()
	netboxEntry.SetPlaceHolder("API NetBox Token")
	netboxEntry.Password = true

	scheduleEntry := widget.NewEntry()
	scheduleEntry.SetPlaceHolder("Интервал (дней, 1–31). По-умолчанию - 1")
	scheduleEntry.Resize(fyne.NewSize(140, 40))
	// Поле выбора времени
	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Время обновления (HH:MM). По-умолчанию - 00:00")
	//aaaa
	outputText := binding.NewString()
	//output := widget.NewMultiLineEntry()

	output := NewReadOnlyEntry()
	output.MultiLine = true
	//output.Disable()

	output.Bind(outputText)
	//output.Wrapping = fyne.TextWrapWord
	//output.SetMinRowsVisible(15)
	//scroll := container.NewVScroll(output)
	scroll := container.NewVScroll(output)
	//scroll.SetMinSize(fyne.NewSize(600, 300))
	//scroll.SetMinSize(fyne.NewSize(600, 300))  // твой размер
	//scroll.Resize(fyne.NewSize(600, 300))
	//scroll.SetOverlayScrollbars(false)

	scroll.SetMinSize(fyne.NewSize(760, 250))
	//scroll.Resize(fyne.NewSize(460, 250))
	output.SetMinRowsVisible(15)
	//scroll.Offset = fyne.NewPos(0, 0)
	output.Scroll = container.ScrollNone
	output.Validator = nil
	//scroll.ShowScrollbarsOnlyWhenNeeded = true
	//scroll.SetShowScrollbarsWhenNeeded(true)
	//scroll.SetOverlayScrollbars(false)
	// Убираем отступы окна, чтобы выглядело как настоящее приложение
	w.SetPadded(false)
	//scroll.Offset = fyne.NewPos(0, scroll.Content.Size().Height)
	//scroll.SetMinSize(fyne.NewSize(600, 300))
	//scroll.SetMaxSize(fyne.NewSize(600, 300))
	scroll.ScrollToBottom()
	scroll.Refresh()

	w.CenterOnScreen()
	asIsCheck := widget.NewCheck("Как есть", nil)
	platformCheck := widget.NewCheck("Платформа", nil)
	regionCheck := widget.NewCheck("Регион          ", nil)

	updateCheck := widget.NewCheck("Обновление", nil)
	progressCheck := widget.NewCheck("Прогресс", nil)

	dcCheck := widget.NewCheck("ЦОД", nil)
	lanCheck := widget.NewCheck("ЛВС", nil)

	updateHint := func() {
		var lines []string

		// --- Сортировка ---
		if asIsCheck.Checked {
			lines = append(lines, "🔎Сортировка: как есть (без изменений).")
		} else if platformCheck.Checked {
			lines = append(lines, "🔎Сортировка: по платформам.")
		} else if regionCheck.Checked {
			lines = append(lines, "🔎Сортировка: по регионам.")
		}

		// --- Режим сохранения ---
		if updateCheck.Checked {
			lines = append(lines, "📂Режим: Обновление текущих файлов (перезапись)")
		} else if progressCheck.Checked {
			lines = append(lines, "📂Режим: Сохранение истории (архив по дням)")
		}

		if dcCheck.Checked && lanCheck.Checked {
			lines = append(lines, "ЦОД и ЛВС\n")
		} else if dcCheck.Checked {
			lines = append(lines, "ЦОД\n")
		} else if lanCheck.Checked {
			lines = append(lines, "ЛВС\n")
		}

		// Выводим
		if len(lines) == 0 {
			_ = outputText.Set("")
		} else {
			_ = outputText.Set(strings.Join(lines, "\n"))
		}
	}

	// === ОБРАБОТЧИКИ СОРТИРОВКИ ===
	asIsCheck.OnChanged = func(checked bool) {
		if checked {
			platformCheck.SetChecked(false)
			regionCheck.SetChecked(false)
		}
		updateHint()
	}

	platformCheck.OnChanged = func(checked bool) {
		if checked {
			asIsCheck.SetChecked(false) // ← было platformCheck!
			regionCheck.SetChecked(false)
		}
		updateHint()
	}

	regionCheck.OnChanged = func(checked bool) {
		if checked {
			asIsCheck.SetChecked(false)
			platformCheck.SetChecked(false) // ← было platformCheck!
		}
		updateHint()
	}

	// === ОБРАБОТЧИКИ РЕЖИМА СОХРАНЕНИЯ ===
	updateCheck.OnChanged = func(checked bool) {
		if checked {
			progressCheck.SetChecked(false)
		}
		updateHint()
	}

	progressCheck.OnChanged = func(checked bool) {
		if checked {
			updateCheck.SetChecked(false)
		}
		updateHint()
	}

	dcCheck.OnChanged = func(checked bool) {
		updateHint()
	}

	lanCheck.OnChanged = func(checked bool) {
		updateHint()
	}

	// === ИНИЦИАЛИЗАЦИЯ ПОДСКАЗКИ ПРИ ЗАПУСКЕ ===
	updateHint() // ← теперь текст появляется сразу!
	//updateCheck.OnChanged = func(checked bool) {
	//	if checked {
	//		passChecks("Обновление")
	//	} else {
	//		_ = outputText.Set("") // все выключены
	//	}
	//}
	//
	//progressCheck.OnChanged = func(checked bool) {
	//	if checked {
	//		passChecks("История")
	//	} else {
	//		_ = outputText.Set("") // все выключены
	//	}
	//}
	// Делаем два чекбоксы взаимоисключающими (как радиокнопки)

	// По умолчанию — режим обновления
	updateCheck.SetChecked(false)
	progressCheck.SetChecked(false)
	dcCheck.SetChecked(false)
	lanCheck.SetChecked(false)

	//blockInputs := func() {
	//	loginEntry.Disable()
	//	passEntry.Disable()
	//	netboxEntry.Disable()
	//	scheduleEntry.Disable()
	//	timeEntry.Disable()
	//
	//}
	//
	//unblockInputs := func() {
	//	loginEntry.Enable()
	//	passEntry.Enable()
	//	netboxEntry.Enable()
	//	scheduleEntry.Enable()
	//	timeEntry.Enable()
	//}

	// Восстанавливаем сохранённый режим
	if cfg.SaveMode == "progress" {
		updateCheck.SetChecked(false)
		progressCheck.SetChecked(true)
	} else if cfg.SaveMode == "update" {
		updateCheck.SetChecked(true)
		progressCheck.SetChecked(false)
	}

	if cfg.TargetMode == "both" {
		dcCheck.SetChecked(true)
		lanCheck.SetChecked(true)
	} else if cfg.TargetMode == "LAN" {
		lanCheck.SetChecked(true)
		dcCheck.SetChecked(false)
	} else if cfg.TargetMode == "DC" {
		dcCheck.SetChecked(true)
		lanCheck.SetChecked(false)
	}

	if cfg.GitLabLogin != "" {
		loginEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
	}
	if cfg.GitLabPass != "" {
		passEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
	}
	if cfg.NetboxToken != "" {
		netboxEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
	}

	if cfg.Mode == "as-is" {
		asIsCheck.SetChecked(true)
	} else if cfg.Mode == "platform" {
		platformCheck.SetChecked(true)
	} else if cfg.Mode == "region" {
		regionCheck.SetChecked(true)
	}

	if cfg.ScheduleDays > 0 {
		scheduleEntry.SetPlaceHolder(fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)")
	}
	if cfg.ScheduleTime != "" {
		timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
	}

	saveBtn := widget.NewButton("Сохранить", nil)

	scheduleEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			scheduleEntry.SetText("")
			scheduleEntry.SetPlaceHolder((fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)"))
		}
	}
	///чанги
	timeEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			timeEntry.SetText("")
			timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
		}
	}

	startPauseBtn := widget.NewButton("Старт", nil)
	startPauseBtn.Importance = widget.MediumImportance
	//PauseUpdateButtonState(startPauseBtn, cfg, configPath)

	//записываем настройки
	setConfig := func() {
		cfg.GitLabLogin = loginEntry.Text
		cfg.GitLabPass = passEntry.Text
		cfg.NetboxToken = netboxEntry.Text
		//
		//if startPauseBtn.Text == "Старт" {
		//	cfg.SchedulerState = "paused"
		//} else {
		//	cfg.SchedulerState = "running"
		//}

		switch {
		case asIsCheck.Checked:
			cfg.Mode = "as-is"
		case platformCheck.Checked:
			cfg.Mode = "platform"
		case regionCheck.Checked:
			cfg.Mode = "region"
		}

		switch {
		case updateCheck.Checked:
			cfg.SaveMode = "update"
		case progressCheck.Checked:
			cfg.SaveMode = "progress"
		}

		switch {
		case lanCheck.Checked && dcCheck.Checked:
			cfg.TargetMode = "both"
		case lanCheck.Checked && !dcCheck.Checked:
			cfg.TargetMode = "LAN"
		case !lanCheck.Checked && dcCheck.Checked:
			cfg.TargetMode = "DC"
		}

		//cfg.SaveMode = modeRadio.Selected

		//day, err := strconv.Atoi(scheduleEntry.Text)
		//if days < 1 {
		//	days = 1
		//} else if
		//{else if days > 31 {
		//	days = 31
		//}

		//days := 1 // значение по умолчанию

		daysRaw := strings.TrimSpace(scheduleEntry.Text)
		days := 1 // значение по умолчанию

		if daysRaw != "" {
			d, err := strconv.Atoi(daysRaw)
			if err != nil || d < 1 || d > 31 {

				dialog.ShowInformation(
					"Ой!",
					"Введите число от 1 до 31.\n",
					w,
				)

				return // ← прерываем сохранение, если ошибка

			}
			days = d
		}
		// Если поле пустое — оставляем 1 (или можно cfg.ScheduleDays, если уже есть)
		cfg.ScheduleDays = days

		timeValue := strings.TrimSpace(timeEntry.Text)
		if timeValue == "" {
			timeValue = "00:00"
			cfg.ScheduleTime = timeValue
		} else if !isValidTime(timeValue) {
			dialog.ShowInformation(
				"Ой!",
				"Введите время в формате ЧЧ:ММ.\n",
				w,
			)

			return

		} else {
			cfg.ScheduleTime = strings.TrimSpace(timeEntry.Text)
		}

		cfg.ScheduleDays = days
		cfg.LastRun = time.Now().Format("2006-01-02T15:04:05Z07:00")
		if err := saveConfig(cfg, configPath); err != nil {
			appendOutput(outputText, fmt.Sprintf("❌ Ошибка сохранения: %v", err))
		} else {
			loginEntry.SetText("")
			loginEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			//loginEntry.TextStyle = fyne.TextStyle{Bold: false, Italic: false, Monospace: false}
			passEntry.SetText("")
			passEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			//passEntry.TextStyle = fyne.TextStyle{Bold: false, Italic: false, Monospace: false}
			netboxEntry.SetText("")
			netboxEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			//netboxEntry.TextStyle = fyne.TextStyle{Bold: false, Italic: false, Monospace: false}
			scheduleEntry.SetText("")
			scheduleEntry.SetPlaceHolder((fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)"))
			timeEntry.SetText("")
			timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
			PauseUpdateButtonState(startPauseBtn, cfg, configPath)
			_ = saveConfig(cfg, configPath)
			appendOutput(outputText, "✅ Настройки сохранены.")
			saveBtn.SetText("Сбросить")
			//blockInputs()

		}
	}

	//если конфига есть, то меняет название кнопки
	updateButtonState := func() {
		if _, err := os.Stat(configPath); err == nil {
			saveBtn.SetText("Сбросить")
			//blockInputs()
		} else {
			saveBtn.SetText("Сохранить")
			//unblockInputs()
		}
	}

	updateButtonState()
	outputView := widget.NewLabelWithData(outputText)
	//сбросить конфигу
	resetConfig := func() {
		saveBtn.SetText("Сохранить")
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			outputView.SetText(outputView.Text + "\n❌ Ошибка удаления config.json: " + err.Error())
			return
		}

		// Сбрасываем GUI
		loginEntry.SetText("")
		passEntry.SetText("")
		netboxEntry.SetText("")
		loginEntry.SetPlaceHolder("Вставьте логин GitLab")
		passEntry.SetPlaceHolder("Вставьте пароль GitLab")
		netboxEntry.SetPlaceHolder("Вставьте токен NetBox")

		asIsCheck.SetChecked(false)
		platformCheck.SetChecked(false)
		regionCheck.SetChecked(false)

		updateCheck.SetChecked(false)
		progressCheck.SetChecked(false)

		dcCheck.SetChecked(false)
		lanCheck.SetChecked(false)

		scheduleEntry.SetText("")
		scheduleEntry.SetPlaceHolder("Интервал (дней, 1–31)")
		timeEntry.SetText("")
		timeEntry.SetPlaceHolder("Время обновления (HH:MM)")
		appendOutput(outputText, "⚙️ Настройки сброшены.\n")
		PauseUpdateButtonState(startPauseBtn, cfg, configPath)
		updateButtonState()
		//unblockInputs()
	}
	//действие кнопки "скачать/сохранить"
	saveBtn.OnTapped = func() {
		if saveBtn.Text == "Сохранить" {

			//ошибка логина
			if loginEntry.Text == "" && !fileExists(configPath) {
				dialog.ShowInformation("Ой!", "⚠️ Нет логина.", w)
				return
			}

			if passEntry.Text == "" && !fileExists(configPath) {
				dialog.ShowInformation("Ой!", "⚠️ Нет пароля.", w)
				return
			}
			if netboxEntry.Text == "" && !fileExists(configPath) {
				dialog.ShowInformation("Ой!", "⚠️ Нет токена.", w)
				return
			}

			if !checkModeSelected(asIsCheck, platformCheck, regionCheck, w) {
				return
			}
			if !passModeSelected(updateCheck, progressCheck, w) {
				return
			}
			if !targetModeSelected(updateCheck, progressCheck, w) {
				return
			}

		}

		//
		if saveBtn.Text == "Сбросить" {
			dialog.ShowConfirm(
				"Подтверждение",
				"Вы уверены, что хотите удалить все сохранённые данные?",
				func(confirmed bool) {
					if confirmed {
						resetConfig() //функция сброса
						//unblockInputs()
						saveBtn.SetText("Сохранить")

					}
				},
				w, // ← окно, к которому относится диалог
			)
			return
		} else {
			setConfig() //функция установки
			//blockInputs()

		}
	}

	// 1. Создаём кнопку для загрузки
	cloneBtn := widget.NewButton("Скачать", nil)
	cloneBtn.Importance = widget.HighImportance

	loginEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			loginEntry.SetText("")
			loginEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
		}
	}
	passEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			passEntry.SetText("")
			passEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
		}
	}

	netboxEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			netboxEntry.SetText("")
			netboxEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
		}
	}

	allBlock := func() {
		cloneBtn.Disable()
		saveBtn.Disable()
		asIsCheck.Disable()
		platformCheck.Disable()
		regionCheck.Disable()
		progressCheck.Disable()
		updateCheck.Disable()
		startPauseBtn.Disable()
		dcCheck.Disable()
		lanCheck.Disable()
		//startPauseBtn.Enable()
		//btn.Disable()
	}

	allUnblock := func() {
		cloneBtn.Enable()
		saveBtn.Enable()
		asIsCheck.Enable()
		platformCheck.Enable()
		regionCheck.Enable()
		progressCheck.Enable()
		updateCheck.Enable()
		dcCheck.Enable()
		lanCheck.Enable()
		if fileExists(configPath) {
			startPauseBtn.Enable()
		}
	}

	//passChecks.Horizontal = true
	//modeRadio.SetSelected("Обновление")

	// Горизонтальная строка: слева — подпись, справа — радиокнопки

	//modeRow := container.NewHBox(
	//	label,
	//	layout.NewSpacer(),
	//	modeRadio, // справа — радиокнопки
	//)
	//modeCard2 := widget.NewCard("", "", modeRow)
	//modeRadioContainer := container.NewCenter(modeCard2)
	startDownload := func() {

		if loginEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет логина.", w)
			return
		}

		if passEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет пароля.", w)
			return
		}
		if netboxEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет токена.", w)
			return
		}

		if !checkModeSelected(asIsCheck, platformCheck, regionCheck, w) {
			return
		}
		if !passModeSelected(updateCheck, progressCheck, w) {
			return
		}
		if !targetModeSelected(dcCheck, lanCheck, w) {
			return
		}

		outputText.Set("")
		updateHint()
		//clearLogKeepHeader(outputText, &output.Entry)

		// ... проверки логина/пароля ...
		fyne.Do(allBlock)
		//allBlock()
		_ = appendOutput(outputText, "Начинаю загрузку из GitLab...\n")

		login := strings.TrimSpace(loginEntry.Text)
		pass := strings.TrimSpace(passEntry.Text)
		token := strings.TrimSpace(netboxEntry.Text)

		if login == "" && cfg.GitLabLogin != "" {
			login = cfg.GitLabLogin
		}
		if pass == "" && cfg.GitLabPass != "" {
			pass = cfg.GitLabPass
		}
		if token == "" && cfg.NetboxToken != "" {
			token = cfg.NetboxToken
		}
		//!загрузка ЦОД
		if dcCheck.Checked {

			repoURL := "https://configs.net.rt.ru/dc/configs.git"
			authURL := repoURL
			dstDir := filepath.Join(targetDir, "ЦОД")

			if login != "" && pass != "" {
				//passBytes, _ := base64.StdEncoding.DecodeString(pass)
				//passString := string(passBytes)
				encodedPass := url.QueryEscape(pass)
				//doubleEncoded := url.QueryEscape(encodedPass)
				authURL = fmt.Sprintf("https://%s:%s@%s", login, encodedPass, strings.TrimPrefix(repoURL, "https://"))
				//println(authURL)
			}

			cmd := exec.Command("git", "clone", "--depth", "1", authURL, dstDir)

			outputBytes, err := cmd.CombinedOutput()

			_ = string(outputBytes)

			if err != nil {

				switch {
				case strings.Contains(string(outputBytes), "Authentication failed"):
					_ = appendOutput(outputText, "Ошибка: неверный логин или пароль GitLab\n")
				case strings.Contains(string(outputBytes), "not found"):
					_ = appendOutput(outputText, "Ошибка: git не найден в PATH\n")
				case strings.Contains(string(outputBytes), "Could not resolve host"):
					_ = appendOutput(outputText, "Ошибка: нет интернета или сервер недоступен\n")
				case strings.Contains(string(outputBytes), "Repository not found"):
					_ = appendOutput(outputText, "Ошибка: репозиторий не найден или нет доступа\n")
				default:
					_ = appendOutput(outputText, "Ошибка git clone: "+err.Error()+"\n")
				}
				fyne.Do(allUnblock)
				return
			}
			if err := RemoveGitFolder(dstDir, outputText); err != nil {
				return
			}

			// ВСЁ НОРМА — запускаем нужный режим
			isProgressMode := progressCheck.Checked

			if isProgressMode {
				//_ = appendOutput(outputText, "Режим: Сохранение истории (архив по дням)\n")
				if err := progdl.RunProgressMode(targetDir, sortedDst, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked,
					token, outputText, scroll); err != nil {
					_ = appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {

					_ = appendOutput(outputText, "Все операции для ЦОД выполнены!\n")

				}
			} else {
				//_ = appendOutput(outputText, "Режим: Обновление текущих файлов\n")
				if err := progdl.RunUpdateMode(targetDir, sortedDst,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked,
					token, outputText, scroll); err != nil {
					_ = appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						return
					} else {
						_ = appendOutput(outputText, "Все операции для ЦОД выполнены!\n")
					}

				}
			}

			cfg.LastRun = time.Now().Format(time.RFC3339)

			fyne.Do(func() {
				scroll.ScrollToBottom()
				scroll.Refresh()
				allUnblock()
			})

		}

		if lanCheck.Checked {

			repoURL := "https://configs.net.rt.ru/lan/configs.git"
			authURL := repoURL
			dstDir := filepath.Join(targetDir, "ЛВС")
			if login != "" && pass != "" {
				//passBytes, _ := base64.StdEncoding.DecodeString(pass)
				//passString := string(passBytes)
				encodedPass := url.QueryEscape(pass)
				//doubleEncoded := url.QueryEscape(encodedPass)
				authURL = fmt.Sprintf("https://%s:%s@%s", login, encodedPass, strings.TrimPrefix(repoURL, "https://"))
				//println(authURL)
			}

			cmd := exec.Command("git", "clone", "--depth", "1", authURL, dstDir)
			//RemoveGitFolder(dstDir, outputText)
			outputBytes, err := cmd.CombinedOutput()

			_ = string(outputBytes)

			if err != nil {

				switch {
				case strings.Contains(string(outputBytes), "Authentication failed"):
					_ = appendOutput(outputText, "Ошибка: неверный логин или пароль GitLab\n")
				case strings.Contains(string(outputBytes), "not found"):
					_ = appendOutput(outputText, "Ошибка: git не найден в PATH\n")
				case strings.Contains(string(outputBytes), "Could not resolve host"):
					_ = appendOutput(outputText, "Ошибка: нет связи или сервер недоступен\n")
				case strings.Contains(string(outputBytes), "Repository not found"):
					_ = appendOutput(outputText, "Ошибка: репозиторий не найден или нет доступа\n")
				default:
					_ = appendOutput(outputText, "Ошибка git clone: "+err.Error()+"\n")
				}
				//_ = RemoveGitFolder(dstDir, outputText)
				fyne.Do(allUnblock)
				return
			}

			if err := RemoveGitFolder(dstDir, outputText); err != nil {
				return
			}

			// ВСЁ НОРМА — запускаем нужный режим
			isProgressMode := progressCheck.Checked

			if isProgressMode {
				//_ = appendOutput(outputText, "Режим: Сохранение истории (архив по дням)\n")
				if err := progdl.RunProgressMode(targetDir, sortedDst, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked,
					token, outputText, scroll); err != nil {
					_ = appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {
					_ = appendOutput(outputText, "Все операции для ЛВС выполнены!\n")

				}
			} else {
				//_ = appendOutput(outputText, "Режим: Обновление текущих файлов\n")
				if err := progdl.RunUpdateMode(targetDir, sortedDst,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked,
					token, outputText, scroll); err != nil {
					_ = appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {
					_ = appendOutput(outputText, "Все операции для ЛВС выполнены!\n")
				}
			}

			cfg.LastRun = time.Now().Format(time.RFC3339)

			fyne.Do(func() {
				scroll.ScrollToBottom()
				scroll.Refresh()
				allUnblock()
			})

		}

	}

	cloneBtn.OnTapped = func() {

		if isAutoRun {
			isAutoRun = false // сбрасываем
			startDownload()
			return
		}

		// Если config.json существует И расписание настроено → спрашиваем
		if fileExists(configPath) && cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
			dialog.ShowConfirm(
				"Ой!",
				"Всё равно скачать сейчас?\n",
				func(confirmed bool) {
					if confirmed {
						//clearLogKeepHeader(outputText, &output.Entry)
						_ = appendOutput(outputText, "Запуск по запросу пользователя.\n")
						go func() {
							startDownload()
						}()
						// ← запускаем скачивание
					}
				},
				w,
			)
		} else {
			// Если расписания нет — скачиваем сразу
			go startDownload()
		}
	}

	cloneBtn.Resize(fyne.NewSize(140, 40))
	cloneButtonContainer := container.NewHBox(layout.NewSpacer(), cloneBtn, layout.NewSpacer())

	saveBtn.Resize(fyne.NewSize(140, 40))
	startPauseBtn = CreateStartPauseButton(cfg, configPath, func() { cloneBtn.OnTapped() }, outputText, scroll, w)
	PauseUpdateButtonState(startPauseBtn, cfg, configPath)
	//startPauseBtn = CreateStartPauseButton(cfg, configPath, func() { cloneBtn.OnTapped() }, outputText, w)

	//// Добавляешь в интерфейс
	//saveButtonContainer := container.NewHBox(
	//	layout.NewSpacer(),
	//	saveBtn,
	//	startPauseBtn,
	//	layout.NewSpacer(),
	//)
	//saveButtonContainer := container.NewHBox(layout.NewSpacer(), saveBtn, layout.NewSpacer())
	saveButtonContainer := container.NewHBox(
		layout.NewSpacer(),
		saveBtn,
		startPauseBtn, // ← вот она!
		layout.NewSpacer(),
	)
	sortLabel := widget.NewLabel("Сортировка:      ")
	sortLabel.TextStyle = fyne.TextStyle{Bold: true}

	sortBox := container.NewGridWithColumns(3,
		container.NewCenter(asIsCheck),
		container.NewCenter(platformCheck),
		container.NewCenter(regionCheck),
	)
	sortCard := widget.NewCard("", "", sortBox)
	sortBlock := container.NewHBox(
		container.NewCenter(sortLabel), // центрируем по вертикали
		sortCard,
	)

	passLabel := widget.NewLabel("Режим работы:")
	passLabel.TextStyle = fyne.TextStyle{Bold: true}

	passBox := container.NewGridWithColumns(2,
		container.NewCenter(updateCheck),
		container.NewCenter(progressCheck),
	)
	passCard := widget.NewCard("", "", passBox)
	passBlock := container.NewHBox(
		container.NewCenter(passLabel), // центрируем по вертикали
		passCard,
	)

	targetLabel := widget.NewLabel("Цель:")
	targetLabel.TextStyle = fyne.TextStyle{Bold: true}

	targetBox := container.NewGridWithColumns(2,
		container.NewCenter(dcCheck),
		container.NewCenter(lanCheck),
	)

	targetCard := widget.NewCard("", "", targetBox)
	targetBlock := container.NewHBox(
		container.NewCenter(targetLabel), // центрируем по вертикали
		targetCard,
	)

	passtargetButtonContainer := container.NewHBox(
		//layout.NewSpacer(),
		passBlock,
		targetBlock,
		//layout.NewSpacer(),
	)

	//modeBox := container.NewGridWithColumns(3,
	//	container.NewCenter(asIsCheck),
	//	container.NewCenter(platformCheck),
	//	container.NewCenter(regionCheck),
	//)
	//modeCard := widget.NewCard("", "", modeBox)
	//centeredModeBox := container.NewCenter(modeCard)

	//запуск расписания
	//if cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
	//	go scheduler.Start(
	//		configPath,       // ← путь к файлу
	//		cfg.ScheduleDays, // ← дни
	//		cfg.ScheduleTime, // ← время
	//		cfg.LastRun,      // ← текущий LastRun
	//		func(newLastRun string) { // ← callback: обновляем cfg и сохраняем
	//			cfg.LastRun = newLastRun
	//			_ = saveConfig(*cfg, configPath)
	//		},
	//		func() { cloneBtn.OnTapped() }, // ← действие
	//		outputText,
	//		cancel,
	//	)
	//
	//}

	// Сборка формы
	form := container.NewVBox(
		loginEntry,
		passEntry,
		netboxEntry,
		sortBlock,
		passtargetButtonContainer,
		//passBlock,
		//centeredModeBox,
		//modeRadioContainer,

		cloneButtonContainer,
		scroll,
		//saveButtonContainer := container.NewHBox(layout.NewSpacer(),  saveBtn,  startPauseBtn  )
		saveButtonContainer,
		scheduleEntry,
		timeEntry,
	)

	w.SetContent(form)
	w.ShowAndRun()
}
