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
	"syscall"
	"time"

	conf "configtool.local/conf"
	"configtool.local/crypt"
	"configtool.local/progdl"
	"configtool.local/sound"
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
var manualRun = false

// var SecretKey []byte
var (
	asIsCheck     *widget.Check
	platformCheck *widget.Check
	regionCheck   *widget.Check

	progressCheck *widget.Check
	updateCheck   *widget.Check

	dcCheck     *widget.Check
	lanCheck    *widget.Check
	aclCheck    *widget.Check
	parserCheck *widget.Check
	scroll      *container.Scroll
	outputText  binding.String
	nextRunTime time.Time
	targetTime  time.Duration
	interval    time.Duration
	cfg         *conf.AppConfig
	cfg2        *conf.AppConfig
	cloneBtn    *widget.Button
	cancel      chan struct{}
	//schedulerRunning bool
	// ... другие виджеты, если нужноasIsCheck
)

// Вызов:
func SaveDoubleEncryptedConfig(cfg *conf.AppConfig, path string, internalKey, externalKey []byte) error {
	// 1. Шифруем чувствительные поля (внутренний слой)
	encPass, err := crypt.Encrypt(cfg.GitLabPass, internalKey)
	if err != nil {
		return err
	}
	encToken, err := crypt.Encrypt(cfg.NetboxToken, internalKey)
	if err != nil {
		return err
	}

	// Создаём копию с зашифрованными полями
	encCfg := *cfg
	encCfg.GitLabPass = encPass
	encCfg.NetboxToken = encToken

	// 2. Сериализуем в JSON
	jsonData, err := json.MarshalIndent(encCfg, "", "  ")
	if err != nil {
		return err
	}

	// 3. Шифруем весь JSON (внешний слой)
	encryptedAll, err := crypt.Encrypt(string(jsonData), externalKey)
	if err != nil {
		return err
	}

	// 4. Сохраняем
	return os.WriteFile(path, []byte(encryptedAll), 0600)
}

func LoadDoubleEncryptedConfig(path string, internalKey, externalKey []byte) (*conf.AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 1. Расшифровываем внешний слой
	jsonStr, err := crypt.Decrypt(string(data), externalKey)
	if err != nil {
		return nil, err
	}

	// 2. Парсим JSON
	cfg := &conf.AppConfig{}
	if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
		return nil, err
	}

	// 3. Расшифровываем внутренние поля
	decPass, err := crypt.Decrypt(cfg.GitLabPass, internalKey)
	if err != nil {
		return nil, err
	}
	decToken, err := crypt.Decrypt(cfg.NetboxToken, internalKey)
	if err != nil {
		return nil, err
	}

	cfg.GitLabPass = decPass
	cfg.NetboxToken = decToken

	return cfg, nil
}

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

func checkModeSelected(asIsCheck, platformCheck, regionCheck, aclCheck *widget.Check, w fyne.Window) bool {
	if !asIsCheck.Checked && !platformCheck.Checked && !regionCheck.Checked && !aclCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите режим сортировки.", w)
		return false
	}
	return true
}

func passModeSelected(updateCheck, progressCheck *widget.Check, w fyne.Window) bool {
	if !updateCheck.Checked && !progressCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите режим работы.", w)
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

//func appendOutput(bindStr binding.String, msg string) error {
//	current, err := bindStr.Get()
//	if err != nil {
//		return fmt.Errorf("ошибка чтения binding.String: %w", err)
//	}
//
//	if err := bindStr.Set(current + msg + "\n"); err != nil {
//		return fmt.Errorf("ошибка записи binding.String: %w", err)
//	}
//
//	return nil
//}

func appendOutput(output binding.String, msg string) {
	fyne.Do(func() {
		current, _ := output.Get()
		_ = output.Set(current + msg)
	})
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

	return nil
}

func saveConfig(cfg *conf.AppConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
func LoadConfig(path string) (*conf.AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	cfg := &conf.AppConfig{}
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// восстановление боксов после переоткрытия
func loadConfigFromFile() {

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return
	}
	switch cfg.Mode {
	case "as-is":
		asIsCheck.SetChecked(true)
	case "platform":
		platformCheck.SetChecked(true)
	case "region":
		regionCheck.SetChecked(true)
	case "ACL":
		aclCheck.SetChecked(true)
	default:
		asIsCheck.SetChecked(false)
		platformCheck.SetChecked(false)
		regionCheck.SetChecked(false)
		aclCheck.SetChecked(false)

	}

	switch cfg.SaveMode {
	case "progress":
		progressCheck.SetChecked(true)
	case "update":
		updateCheck.SetChecked(true)
	default:
		updateCheck.SetChecked(false)
		progressCheck.SetChecked(false)
	}

	switch cfg.TargetMode {
	case "both":
		dcCheck.SetChecked(true)
		lanCheck.SetChecked(true)
	case "DC":
		dcCheck.SetChecked(true)
		lanCheck.SetChecked(false)
	case "LAN":
		dcCheck.SetChecked(false)
		lanCheck.SetChecked(true)
	default:
		dcCheck.SetChecked(false)
		lanCheck.SetChecked(false)
	}

}

// для остановки

//	type AppConfig struct {
//		GitLabLogin    string `json:"gitlab_login"`
//		GitLabPass     string `json:"gitlab_pass_hash"`
//		NetboxToken    string `json:"netbox_token_hash"`
//		Mode           string `json:"mode"` // as-is / platform / region
//		SaveMode       string `json:"save_mode"`
//		ScheduleDays   int    `json:"schedule_days"`
//		ScheduleTime   string `json:"schedule_time"` // "HH:MM"
//		LastRun        string `json:"last_run,omitempty"`
//		SchedulerState string `json:"scheduler_state"`
//	}
//
// uh
func updateHint() {
	var lines []string
	//outputText := binding.NewString()

	// --- Сортировка ---
	if asIsCheck.Checked {
		lines = append(lines, "🔎Сортировка: Как есть (без изменений).")
		//netboxEntry.Disable()
		//netboxEntry.SetPlaceHolder("Введите API NetBox Token")
	} else if platformCheck.Checked {
		//netboxEntry.Enable()
		//netboxEntry.SetPlaceHolder("Введите API NetBox Token")
		lines = append(lines, "🔎Сортировка: по платформам.")
	} else if regionCheck.Checked {
		lines = append(lines, "🔎Сортировка: по регионам.")
		//} else if aclCheck.Checked {
		//	lines = append(lines, "🔎Сортировка: Access-lists.")
	} else if aclCheck.Checked && parserCheck.Checked {
		lines = append(lines, "🔎Сортировка: Access-lists + Загрузка парсера.")
		// Текст про Парсер появляется ТОЛЬКО если галочка включена
	} else if aclCheck.Checked {
		lines = append(lines, "🔎Сортировка: Access-lists.")
	}

	// --- Режим сохранения ---
	if updateCheck.Checked {
		lines = append(lines, "📂Режим: Обновление текущих файлов (перезапись файлов).")
	} else if progressCheck.Checked {
		lines = append(lines, "📂Режим: Сохранение истории (архив по дням).")
	}

	if dcCheck.Checked && lanCheck.Checked {
		lines = append(lines, "🗃️Цель: ЦОД и ЛВС\n")
	} else if dcCheck.Checked {
		lines = append(lines, "🗃️Цель: ЦОД\n")
	} else if lanCheck.Checked {
		lines = append(lines, "🗃️Цель: ЛВС\n")
	}

	// Выводим
	if len(lines) == 0 {
		_ = outputText.Set("")
		fyne.Do(func() {
			scroll.Refresh()
			//allUnblock()
		})
	} else {
		_ = outputText.Set(strings.Join(lines, "\n"))

		fyne.Do(func() {
			scroll.Refresh()
			//allUnblock()
		})

	}
}

type ReadOnlyEntry struct {
	widget.Entry
}

func NewReadOnlyEntry() *ReadOnlyEntry {
	e := &ReadOnlyEntry{}
	e.ExtendBaseWidget(e)
	return e
}

func (e *ReadOnlyEntry) Refresh() {
	e.Entry.Refresh()
}

//func hashString(s string) string {
//	h := sha256.Sum256([]byte(s))
//	return hex.EncodeToString(h[:])
//}

// 2
// ❗ Полностью блокируем ввод с клавиатуры
func (e *ReadOnlyEntry) TypedRune(r rune)           {}
func (e *ReadOnlyEntry) TypedKey(ev *fyne.KeyEvent) {}

//func (e *ReadOnlyEntry) Focusable() bool {
//	return false // ← главное: поле НЕ фокусируемо
//}
//
//func (e *ReadOnlyEntry) FocusGained() {
//	// Ничего не делаем — курсор не появляется
//}

// 2
//type ReadOnlyEntry2 struct {
//	widget.Entry
//	editable bool
//}
//
//func NewReadOnlyEntry2() *ReadOnlyEntry2 {
//	e := &ReadOnlyEntry2{}
//	e.ExtendBaseWidget(e)
//	e.editable = false // по умолчанию заблокировано
//	return e
//}
//
//func (e *ReadOnlyEntry2) Focusable() bool {
//	return e.editable
//}
//
//func (e *ReadOnlyEntry2) Tapped(*fyne.PointEvent) {
//	if !e.editable {
//		if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
//			c.Focus(nil)
//		}
//		return
//	}
//	e.Entry.Tapped(nil)
//}
//
//func (e *ReadOnlyEntry2) FocusGained() {
//	if !e.editable {
//		// Снимаем фокус мгновенно
//		fyne.CurrentApp().Driver().CanvasForObject(e).Focus(nil)
//		return
//	}
//	e.Entry.FocusGained()
//}
//
//func (e *ReadOnlyEntry2) TypedRune(r rune) {
//	if e.editable {
//		e.Entry.TypedRune(r)
//	}
//}
//
//func (e *ReadOnlyEntry2) TypedKey(ev *fyne.KeyEvent) {
//	if e.editable {
//		e.Entry.TypedKey(ev)
//	}
//}
//
//func (e *ReadOnlyEntry2) SetEditable(editable bool) {
//	e.editable = editable
//	e.Refresh()
//
//	// Если отключили редактирование — сразу снимаем фокус
//	if !editable {
//		if c := fyne.CurrentApp().Driver().CanvasForObject(e); c != nil {
//			c.Focus(nil)
//		}
//	}
//}

// Метод переключения режима
//func (e *ReadOnlyEntry2) SetEditable(editable bool) {
//	e.editable = editable
//	e.Refresh()
//}

//2

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

func NotifySuccess(title, message string) {
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "10000", title, message).Run()
	sound.PlayYes()
	//exec.Command("paplay", "./icon/yes.mp3").Run()
	//cmd.Run() // ошибки молча игнорируем
}

func NotifyError(title, message string) {
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "10000", title, message).Run()
	sound.PlayNo()
	//exec.Command("paplay", "./icon/no.mp3").Run()
	//cmd.Run()
}

const configPath = "config.json"

func NextTime(cfg *conf.AppConfig) string {
	if cfg == nil {
		return "Конфигурация не загружена"
	}
	//println("12345")
	if cfg.ScheduleDays <= 0 || cfg.ScheduleTime == "" {
		return "Планировщик не настроен"
	}

	// Парсим scheduleTime (формат "HH:MM")
	parts := strings.Split(cfg.ScheduleTime, ":")
	if len(parts) != 2 {
		return "Неверный формат времени в настройках"
	}

	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "Некорректное время в настройках"
	}

	now := time.Now()
	loc := now.Location()

	// Рассчитываем время следующего запуска
	var nextRun time.Time

	if cfg.LastRun == "" {
		// Первый запуск — сегодня в указанное время
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if now.After(today) || now.Equal(today) {
			nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
		} else {
			nextRun = today
		}
	} else {
		// Есть LastRun — считаем от него
		last, err := time.Parse(time.RFC3339, cfg.LastRun)
		if err != nil {
			return "Ошибка парсинга LastRun"
		}

		nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)

		// Если nextRun в прошлом — добавляем дни
		for !nextRun.After(now) {
			nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
		}
	}

	delay := time.Until(nextRun)

	hours := int(delay.Hours())
	minutes := int(delay.Minutes()) % 60

	//println(
	//	"Следующий запуск: %s в %s. (через %d ч. %d мин.)",
	//	nextRun.Format("02.01.2006"),
	//	nextRun.Format("15:04"),
	//	hours,
	//	minutes,
	//)
	return fmt.Sprintf("▶ Планировщик запущен.\n"+
		"🔄 Следующий запуск: %s в %s. (через %d ч. %d мин.)",
		nextRun.Format("02.01.2006"),
		nextRun.Format("15:04"),
		hours,
		minutes,
	)
}

//const lockFileName = "gittornado.lock"

//	func isProcessRunning(pid int) bool {
//		p, err := os.FindProcess(pid)
//		if err != nil {
//			return false
//		}
//		// Нулевой сигнал — проверка существования процесса
//		err = p.Signal(syscall.Signal(0))
//		return err == nil
//	}
func isProcessRunning(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}

func killProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(syscall.SIGTERM) // мягкое завершение
}

func main() {
	//const configPath = "config.json"
	//var cfg *conf.AppConfig
	//var err error
	cfg, err := LoadConfig(configPath)
	//cfg2, err := LoadDoubleEncryptedConfig("config.secure", crypt.SecretKey, crypt.SecretKey)
	if err != nil {
		// файла нет — просим пользователя ввести данные
		cfg = &conf.AppConfig{}
	}
	//cancel = make(chan struct{})
	targetDir := "./configs" // куда клонируем репо
	sortedDst := "./config_files_clear"
	lockPath := filepath.Join(os.TempDir(), "gittornado.lock")

	oldContent, _ := os.ReadFile(lockPath)
	lines := strings.Split(strings.TrimSpace(string(oldContent)), "\n")

	var oldPID int
	isBusy := false

	if len(lines) >= 1 {
		if lines[0] == "BUSY" {
			isBusy = true
			if len(lines) >= 2 {
				oldPID, _ = strconv.Atoi(lines[1])
			}
		} else {
			oldPID, _ = strconv.Atoi(lines[0])
		}
	}

	// Если BUSY → блокируем запуск
	if isBusy && oldPID > 0 && isProcessRunning(oldPID) {
		exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "3000", "Oй!", "Программа уже запущена!").Run()
		sound.PlayBan()
		//fmt.Println("Программа сейчас выполняет длительную операцию (загрузка/сортировка)")
		//fmt.Println("Запуск новой копии заблокирован. Подождите завершения.")
		os.Exit(1)
	}

	// Если нет BUSY, но процесс жив → убиваем старый
	if oldPID > 0 && isProcessRunning(oldPID) {
		//fmt.Printf("Завершаем старый экземпляр (PID %d)\n", oldPID)
		exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "2000", "Oй!", "Рестарт процесса!").Run()
		killProcess(oldPID)
		time.Sleep(1500 * time.Millisecond)
		os.Remove(lockPath)
	}

	// Создаём новый lock
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		fmt.Println("Ошибка создания lock:", err)
		os.Exit(1)
	}

	fmt.Fprintf(f, "%d\n", os.Getpid())
	syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)

	defer func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
		os.Remove(lockPath)
	}()
	//lockPath := filepath.Join(os.TempDir(), "gittornado.lock")
	//
	//f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0666)
	//if err != nil {
	//	fmt.Println("Ошибка открытия lock-файла:", err)
	//	os.Exit(1)
	//}
	//
	//// Пытаемся заблокировать
	//if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
	//	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "3000", "Oй!", "Программа уже запущена!").Run()
	//	sound.PlayBan()
	//	//fmt.Println("Программа уже запущена")
	//	f.Close()
	//	oldPIDBytes, _ := os.ReadFile(lockPath)
	//	oldPID, _ := strconv.Atoi(strings.TrimSpace(string(oldPIDBytes)))
	//
	//	// Если PID есть и процесс жив — блокируем запуск
	//	if oldPID > 0 && isProcessRunning(oldPID) {
	//		fmt.Println("Программа уже запущена (PID:", oldPID, ")")
	//		if err := killProcess(oldPID); err != nil {
	//			fmt.Println("Не удалось завершить старый процесс:", err)
	//		}
	//		//os.Exit(1)
	//	}
	//	//os.Exit(1)
	//}
	//
	//// Записываем PID
	//f.Seek(0, 0)
	//f.Truncate(0)
	//fmt.Fprintf(f, "%d\n", os.Getpid())
	//
	//defer func() {
	//	syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	//	f.Close()
	//	os.Remove(lockPath)
	//
	//}()

	//fmt.Println("Программа запущена успешно (PID:", os.Getpid(), ")")
	a := app.NewWithID("gittornado")
	a.Settings().SetTheme(theme.LightTheme())
	//a.Settings().SetTheme(hiddenTheme{})
	// Set Fyne app icon as well (optional)
	a.SetIcon(fyne.NewStaticResource("icon.png", trayIcon))

	w := a.NewWindow("GitTornado")
	w.Resize(fyne.NewSize(600, 250))
	w.SetFixedSize(true)
	w.CenterOnScreen()

	// Сворачивание в трей по крестику
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	// Читаем старый PID (если файл уже был)
	// или w.ShowAndRun()

	// === SYSTRAY (getlantern) ===
	go func() {
		systray.Run(func() {
			if len(trayIcon) > 0 {
				systray.SetIcon(trayIcon)
				//systray.SetTooltip("GitLab Downloader v0.71")
			}
			//systray.SetTooltip("GitLab Downloader")
			systray.SetTitle("GitTornado")
			systray.SetTooltip("GitTornado v1.0")
			//999
			open := systray.AddMenuItem("Открыть", "Показать программу")
			quit := systray.AddMenuItem("Выход", "Закрыть программу")

			go func() {
				for range open.ClickedCh {

					fyne.Do(func() {
						loadConfigFromFile()
						outputText.Set("")
						updateHint()
						if fileExists(configPath) && cfg.SchedulerState == "running" {
							appendOutput(outputText, NextTime(cfg))
						} else if fileExists(configPath) && cfg.SchedulerState == "paused" {
							appendOutput(outputText, "⚠️ ВНИМАНИЕ! Планировщик не включен. Нажмите «Старт» для возобновления.")
						} else if fileExists(configPath) && cfg.SchedulerState == "" {
							appendOutput(outputText, "⚠️ Планировщик не запущен. Нажмите «Старт» для запуска.")
						}

						//refreshSchedulerStatus()
						//appendOutput(outputText, fmt.Sprintf("12222222"))

						w.Show()
						w.RequestFocus()
						w.Canvas().Focus(nil)
						scroll.Refresh()

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
	//output := NewReadOnlyEntry()
	//4444
	loginEntry := widget.NewEntry()
	//loginEntry := NewReadOnlyEntry2()

	loginEntry.SetPlaceHolder("Введите логин GitLab")
	loginEntry.TextStyle = fyne.TextStyle{}
	blocker := widget.NewLabel("") // перехватывает мышь
	blocker.Resize(loginEntry.Size())

	passEntry := widget.NewEntry()
	passEntry.Password = true
	passEntry.SetPlaceHolder("Введите пароль GitLab")
	//passEntry.TextStyle = fyne.TextStyle{}

	netboxEntry := widget.NewEntry()
	netboxEntry.SetPlaceHolder("Введите API NetBox Token")
	netboxEntry.Password = true
	//netboxEntry.Disable()

	//выбор папки
	savePathEntry := widget.NewEntry()
	savePathEntry.SetPlaceHolder("Папка сохранения. По-умолчанию - текущая")

	browseBtn := widget.NewButton("Обзор", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				//dialog.ShowError(err, w)
				return
			}
			if uri == nil {
				return // пользователь отменил
			}

			// Получаем путь к выбранной папке
			chosenPath := uri.Path()
			savePathEntry.SetText(chosenPath)
			fyne.Do(func() {
				//savePathEntry.Focus()
				//savePathEntry.CursorPos = len(chosenPath)
				//savePathEntry.Focus()
				w.Canvas().Focus(savePathEntry)
				savePathEntry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
				savePathEntry.Refresh()
			})

			// ← Здесь сохрани путь в конфиг, если нужно
			// cfg.SavePath = chosenPath
			// _ = saveConfig(cfg, configPath)

			//appendOutput(outputText, fmt.Sprintf("Папка сохранения выбрана: %s\n", chosenPath))
		}, w)
	})
	//browseBtn.Importance = widget.WarningImportance

	scheduleEntry := widget.NewEntry()
	scheduleEntry.SetPlaceHolder("Интервал (дней, 1–31). По-умолчанию - 1")
	scheduleEntry.Resize(fyne.NewSize(140, 40))
	// Поле выбора времени
	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Время обновления (HH:MM). По-умолчанию - 00:00")
	//aaaa
	outputText = binding.NewString()
	//output := widget.NewMultiLineEntry()

	output := NewReadOnlyEntry()
	output.MultiLine = true
	//output.Disable()

	output.Bind(outputText)
	//output.Wrapping = fyne.TextWrapWord
	//output.SetMinRowsVisible(15)
	//scroll := container.NewVScroll(output)
	scroll = container.NewVScroll(output)
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
	//scroll.ScrollToBottom()
	scroll.Refresh()

	w.CenterOnScreen()
	//asIsCheck := widget.NewCheck("Как есть (без изменений)", func(b bool) {
	//	cfg.Mode = "as-is"
	//	_ = saveConfig(cfg, configPath)
	//})
	asIsCheck = widget.NewCheck("Как есть", nil)
	platformCheck = widget.NewCheck("Платформа", nil)
	regionCheck = widget.NewCheck("Регион", nil)
	aclCheck = widget.NewCheck("ACL", nil)
	parserCheck = widget.NewCheck("+Parser", nil)
	updateCheck = widget.NewCheck("Обновление", nil)
	progressCheck = widget.NewCheck("Прогресс", nil)
	parserCheck = widget.NewCheck("+Parser", nil)
	dcCheck = widget.NewCheck("ЦОД", nil)
	lanCheck = widget.NewCheck("ЛВС  ", nil)

	// === ОБРАБОТЧИКИ СОРТИРОВКИ ===
	asIsCheck.OnChanged = func(checked bool) {
		if checked {
			platformCheck.SetChecked(false)
			regionCheck.SetChecked(false)
			aclCheck.SetChecked(false)
			parserCheck.SetChecked(false)
			parserCheck.Disable()
		}
		updateHint()
	}

	platformCheck.OnChanged = func(checked bool) {
		if checked {
			asIsCheck.SetChecked(false) // ← было platformCheck!
			regionCheck.SetChecked(false)
			aclCheck.SetChecked(false)
			parserCheck.SetChecked(false)
			parserCheck.Disable()

		}
		updateHint()
	}

	regionCheck.OnChanged = func(checked bool) {
		if checked {
			asIsCheck.SetChecked(false)
			platformCheck.SetChecked(false)
			aclCheck.SetChecked(false)
			parserCheck.SetChecked(false)
			parserCheck.Disable()
		}
		updateHint()
	}
	//ac
	aclCheck.OnChanged = func(checked bool) {
		if checked {
			asIsCheck.SetChecked(false)
			platformCheck.SetChecked(false)
			regionCheck.SetChecked(false)
			parserCheck.Enable()
			//parserCheck.SetChecked(false)
			//updateHint() // ← было platformCheck!
		} else {
			parserCheck.SetChecked(false)
			parserCheck.Disable()

		}
		updateHint()
	}

	parserCheck.OnChanged = func(checked bool) {
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

	updateHint() // ← теперь текст появляется сразу!

	// По умолчанию — режим обновления
	updateCheck.SetChecked(false)
	progressCheck.SetChecked(false)
	dcCheck.SetChecked(false)
	lanCheck.SetChecked(false)
	parserCheck.SetChecked(false)
	parserCheck.Disable()

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
	} else if cfg.Mode == "acl" {
		aclCheck.SetChecked(true)
	}

	if cfg.Parser == "true" {
		parserCheck.SetChecked(true)
	}

	if cfg.SavedPlace != "" {
		savePathEntry.SetPlaceHolder(cfg.SavedPlace + "    ( ✅ Путь загрузки сохранен в файл настроек.)")
	}

	if cfg.ScheduleDays > 0 {
		scheduleEntry.SetPlaceHolder(fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)")
	}
	if cfg.ScheduleTime != "" {
		timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
	}
	//!условие для папки сохранения

	saveBtn := widget.NewButton("Сохранить", nil)

	savePathEntry.OnChanged = func(s string) {
		if saveBtn.Text == "Сбросить" {
			savePathEntry.SetText("")
			savePathEntry.SetPlaceHolder(cfg.SavedPlace + "    ( ✅ Путь загрузки сохранен в файл настроек.)")

		}
	}

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
	if fileExists(configPath) {
		browseBtn.Disable()
	} else {
		browseBtn.Enable()
	}
	if fileExists(configPath) && cfg.SchedulerState == "" {
		appendOutput(outputText, "⚠️ Планировщик не активен! Нажмите «Старт» для запуска.")
	}

	//сюда блок обзора
	//PauseUpdateButtonState(startPauseBtn, cfg, configPath)
	//9999
	//записываем настройки
	setConfig := func() {
		encryptedPass, err := crypt.Encrypt(passEntry.Text, crypt.SecretKey)
		if err != nil {
		}

		encryptedToken, err := crypt.Encrypt(netboxEntry.Text, crypt.SecretKey)
		if err != nil {
			// обработка ошибки
		}

		cfg.GitLabLogin = loginEntry.Text
		cfg.GitLabPass = encryptedPass
		cfg.NetboxToken = encryptedToken

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
		case aclCheck.Checked:
			cfg.Mode = "acl"
		}

		if parserCheck.Checked {
			cfg.Parser = "true"
		} else {
			cfg.Parser = "false"
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

		chosenPath := savePathEntry.Text
		if chosenPath == "" || chosenPath == "." {
			// Если поле пустое или "." — берём текущую директорию программы
			currentDir, err := os.Getwd()
			if err != nil {
				currentDir = "." // fallback
			}
			chosenPath = currentDir
			savePathEntry.SetPlaceHolder(" ✅ Текущая папка сохранения") // показываем пользователю
		}

		cfg.SavedPlace = chosenPath

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
			savePathEntry.SetText("")
			savePathEntry.SetPlaceHolder(cfg.SavedPlace + "    ( ✅ Путь загрузки сохранен в файл настроек.)")
			scheduleEntry.SetText("")

			scheduleEntry.SetPlaceHolder((fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)"))
			timeEntry.SetText("")
			timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
			_ = saveConfig(cfg, configPath)
			time.Sleep(500 * time.Millisecond)
			PauseUpdateButtonState(startPauseBtn, cfg, configPath)
			//if !platformCheck.Checked {
			//	cfg.NetboxToken = ""
			//}

			//
			//SaveDoubleEncryptedConfig(cfg, "config.secure", crypt.SecretKey, crypt.SecretKey)
			//if fileExists(configPath) {
			browseBtn.Disable()
			//}
			outputText.Set("")
			updateHint()
			appendOutput(outputText, "✅ Настройки сохранены. Нажмите СТАРТ для запуска планировщика.")
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

			//ckInputs()
		}
	}

	updateButtonState()
	outputView := widget.NewLabelWithData(outputText)
	//сбросить конфигу
	resetConfig := func() {

		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			outputView.SetText(outputView.Text + "\n❌ Ошибка удаления config.json: " + err.Error())
			return
		}
		//loginEntry.SetEditable(true)
		println("login editable in reset")
		saveBtn.SetText("Сохранить")

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
		aclCheck.SetChecked(false)
		parserCheck.SetChecked(false)
		parserCheck.Disable()
		updateCheck.SetChecked(false)
		progressCheck.SetChecked(false)

		dcCheck.SetChecked(false)
		lanCheck.SetChecked(false)

		scheduleEntry.SetText("")
		scheduleEntry.SetPlaceHolder("Интервал дней (1–31)")
		timeEntry.SetText("")
		timeEntry.SetPlaceHolder("Время обновления (HH:MM)")
		savePathEntry.SetText("")
		savePathEntry.SetPlaceHolder("Папка загрузки")
		browseBtn.Enable()
		//netboxEntry.Disable()
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

			if platformCheck.Checked {
				if netboxEntry.Text == "" && !fileExists(configPath) {
					dialog.ShowInformation("Ой!", "⚠️ Нет токена.", w)
					return

				}
			}

			if !checkModeSelected(asIsCheck, platformCheck, regionCheck, aclCheck, w) {
				return
			}
			if !passModeSelected(updateCheck, progressCheck, w) {
				return
			}
			if !targetModeSelected(dcCheck, lanCheck, w) {
				return
			}

		}
		//passLabel := widget.NewLabel(passEntry.Text)
		//loginLabel.Wrapping = fyne.TextWrapWord
		//
		if saveBtn.Text == "Сбросить" {
			dialog.ShowConfirm(
				"Ой!",
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
		if len(s) > 64 {
			loginEntry.SetText(s[:128])
		}
		if saveBtn.Text == "Сбросить" {
			loginEntry.SetText("")
			loginEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			//time.Sleep(32 * time.Millisecond) // ≈ 1 кадр при 60 fps
			//loginEntry.SetEditable(false)
			//loginEntry.FocusLost()
			//w.Canvas().Focus(nil)

			//loginEntry.FocusLost()
			//fyne.CurrentApp().Driver().CanvasForObject(loginEntry).Focus(nil)
			//println("login not editable in set")
			//ReadOnlyEntry2(loginEntry)
			//5555
		}
	}
	passEntry.OnChanged = func(s string) {
		if len(s) > 128 {
			passEntry.SetText(s[:128])
		}
		if saveBtn.Text == "Сбросить" {
			passEntry.SetText("")
			passEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
		}
	}

	netboxEntry.OnChanged = func(s string) {
		if len(s) > 256 {
			netboxEntry.SetText(s[:128])
		}
		if saveBtn.Text == "Сбросить" {
			netboxEntry.SetText("")
			netboxEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
		}
	}
	//w.Canvas().Focus(nil)
	allBlock := func() {
		cloneBtn.Disable()
		saveBtn.Disable()
		asIsCheck.Disable()
		platformCheck.Disable()
		regionCheck.Disable()
		aclCheck.Disable()
		parserCheck.Disable()
		progressCheck.Disable()
		updateCheck.Disable()
		startPauseBtn.Disable()
		dcCheck.Disable()
		lanCheck.Disable()
		browseBtn.Disable()
		//startPauseBtn.Enable()
		//btn.Disable()
	}

	allUnblock := func() {
		cloneBtn.Enable()
		saveBtn.Enable()
		asIsCheck.Enable()
		platformCheck.Enable()
		regionCheck.Enable()
		aclCheck.Enable()
		if aclCheck.Checked {
			parserCheck.Enable()
		} else {
			parserCheck.Disable()
		}

		progressCheck.Enable()
		updateCheck.Enable()
		dcCheck.Enable()
		lanCheck.Enable()

		if fileExists(configPath) {
			startPauseBtn.Enable()
		}
		if fileExists(configPath) {
			browseBtn.Disable()
		} else {
			browseBtn.Enable()
		}
		//}
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
		lockPath := filepath.Join(os.TempDir(), "gittornado.lock")

		// 1. Отмечаем, что идёт важная операция
		f, err := os.OpenFile(lockPath, os.O_RDWR, 0666)
		if err == nil {
			f.Seek(0, 0)
			f.Truncate(0)
			fmt.Fprintf(f, "BUSY\n%d\n", os.Getpid())
			f.Sync()
			f.Close()
		}

		// 2. После завершения всей работы — возвращаем нормальный lock
		defer func() {
			f, _ := os.OpenFile(lockPath, os.O_RDWR, 0666)
			if f != nil {
				f.Seek(0, 0)
				f.Truncate(0)
				fmt.Fprintf(f, "%d\n", os.Getpid())
				f.Sync()
				f.Close()
			}
		}()

		if loginEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет логина.", w)
			return
		}

		if passEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет пароля.", w)
			return
		}

		//if platformCheck.Checked {
		if netboxEntry.Text == "" && !fileExists(configPath) {
			dialog.ShowInformation("Ой!", "⚠️ Нет токена.", w)
			return
		}
		//}

		if !checkModeSelected(asIsCheck, platformCheck, regionCheck, aclCheck, w) {
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
		if manualRun {
			time.Sleep(500 * time.Millisecond)
			appendOutput(outputText, "🔥 Начинаю загрузку из GitLab...\n")
		} else {
			appendOutput(outputText,
				fmt.Sprintf("🚨 Выполняю загрузку по расписанию: %s в %s\n",
					time.Now().Format("02.01.2006"),
					time.Now().Format("15:04"),
				),
			)
		}

		login := strings.TrimSpace(loginEntry.Text)
		pass := strings.TrimSpace(passEntry.Text)
		//888
		token := strings.TrimSpace(netboxEntry.Text)
		path := strings.TrimSpace(savePathEntry.Text)
		//println(savePathEntry.Text + "путь")
		//println(path)
		decryptedPass, err := crypt.Decrypt(cfg.GitLabPass, crypt.SecretKey)
		if err != nil {
		}
		decryptedToken, err := crypt.Decrypt(cfg.NetboxToken, crypt.SecretKey)
		if err != nil {
		}

		if login == "" && cfg.GitLabLogin != "" {
			login = cfg.GitLabLogin
		}
		if pass == "" && cfg.GitLabPass != "" {
			pass = decryptedPass
		}
		if token == "" && cfg.NetboxToken != "" {
			token = decryptedToken
		}
		if path == "" && cfg.SavedPlace != "" {
			path = cfg.SavedPlace
		}

		os.RemoveAll(filepath.Join(path, targetDir))

		//!загрузка ЦОД
		if dcCheck.Checked {

			repoURL := "https://configs.net.rt.ru/dc/configs.git"
			authURL := repoURL
			dstDir := filepath.Join(path, targetDir, "ЦОД")

			//println(dstDir, sdDst)

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
					appendOutput(outputText, "Ошибка: неверный логин или пароль GitLab\n")

				case strings.Contains(string(outputBytes), "not found"):
					appendOutput(outputText, "Ошибка: git не найден в PATH\n")

				case strings.Contains(string(outputBytes), "Could not resolve host"):
					appendOutput(outputText, "Ошибка: нет интернета или сервер недоступен\n")

				case strings.Contains(string(outputBytes), "Repository not found"):
					appendOutput(outputText, "Ошибка: репозиторий не найден или нет доступа\n")

				case err != nil && (os.IsPermission(err) ||
					strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
					strings.Contains(strings.ToLower(err.Error()), "access denied") ||
					strings.Contains(strings.ToLower(err.Error()), "read-only file system")):
					appendOutput(outputText, "Ошибка: нет прав на запись в целевую папку\n")

				default:
					appendOutput(outputText, "Ошибка git clone: "+err.Error()+"\n")
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
				//appendOutput(outputText, "Режим: Сохранение истории (архив по дням)\n")
				if err := progdl.RunProgressMode(dstDir, path, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, manualRun); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло нет так!")
					})
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "🟢 Файлы ЦОД обработаны. Ожидаем файлы ЛВС.\n")
						})
						//fyne.Do(func() {
						//	scroll.ScrollToBottom()
						//	scroll.Refresh()
						//})
					} else {
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЦОД выполнены!\n")
							NotifySuccess("Ура!", "Конфиги обновлены и отсортированы!")
						})
						//appendOutput(outputText, "Все операции для ЦОД выполнены!\n")
					}
				}
			} else {
				sdDst := filepath.Join(path, sortedDst, "ЦОД")
				//println(dstDir, "-dstdir в gui4", sdDst, "-sdDst в gui4")
				if err := os.MkdirAll(sdDst, 0755); err != nil {
					//appendOutput(outputText, fmt.Sprintf("Ошибка создания подпапки %s: %v\n", sdDst, err))
					//continue
				}
				//appendOutput(outputText, "Режим: Обновление текущих файлов\n")
				if err := progdl.RunUpdateMode(dstDir, sdDst,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло не так!")
					})
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "🟢 Файлы ЦОД обработаны. Ожидаем файлы ЛВС.\n")
						})
					} else {
						//appendOutput(outputText, "Все операции для ЦОД выполнены!\n")
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЦОД выполнены!\n")
							NotifySuccess("Ура!", "Конфиги обновлены и отсортированы!")
						})
					}
				}

			}
			if dcCheck.Checked && lanCheck.Checked {
			} else {
				if manualRun == false {
					cfg.LastRun = time.Now().Format(time.RFC3339)
				} else {
					manualRun = false
				}
				fyne.Do(func() {
					scroll.ScrollToBottom()
					scroll.Refresh()
					allUnblock()
				})
			}
		}

		if lanCheck.Checked {

			repoURL := "https://configs.net.rt.ru/lan/configs.git"
			authURL := repoURL
			dstDir := filepath.Join(path, targetDir, "ЛВС")

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
					appendOutput(outputText, "Ошибка: неверный логин или пароль GitLab\n")

				case strings.Contains(string(outputBytes), "not found"):
					appendOutput(outputText, "Ошибка: git не найден в PATH\n")

				case strings.Contains(string(outputBytes), "Could not resolve host"):
					appendOutput(outputText, "Ошибка: нет интернета или сервер недоступен\n")

				case strings.Contains(string(outputBytes), "Repository not found"):
					appendOutput(outputText, "Ошибка: репозиторий не найден или нет доступа\n")

				case err != nil && (os.IsPermission(err) ||
					strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
					strings.Contains(strings.ToLower(err.Error()), "access denied") ||
					strings.Contains(strings.ToLower(err.Error()), "read-only file system")):
					appendOutput(outputText, "Ошибка: нет прав на запись в целевую папку\n")

				default:
					appendOutput(outputText, "Ошибка git clone: "+err.Error()+"\n")
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
				//appendOutput(outputText, "Режим: Сохранение истории (архив по дням)\n")
				if err := progdl.RunProgressMode(dstDir, path, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, manualRun); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло нет так!")
					})
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						//appendOutput(outputText, "Все операции для ЛВС и ЦОД выполнены!\n")
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС и ЦОД выполнены!\n")
							NotifySuccess("Ура!", "Конфиги обновлены и отсортированы!")
						})
					} else {
						//fyne.Do(func() {
						appendOutput(outputText, "✅ Все операции для ЛВС выполнены!\n")
						NotifySuccess("Ура!", "Конфиги обновлены и отсортированы!")
						//})
						//appendOutput(outputText, "Все операции для ЛВС выполнены!\n")

					}
				}
			} else {

				sdDst := filepath.Join(path, sortedDst, "ЛВС")

				//println(dstDir, "-dstdir в gui4", sdDst, "-sdDst в gui4")
				if err := os.MkdirAll(sdDst, 0755); err != nil {
					//appendOutput(outputText, fmt.Sprintf("Ошибка создания подпапки %s: %v\n", sdDst, err))
					//continue
				}
				//appendOutput(outputText, "Режим: Обновление текущих файлов\n")
				if err := progdl.RunUpdateMode(dstDir, sdDst,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcCheck.Checked, lanCheck.Checked, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll); err != nil {
					//fyne.Do(func() {
					appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					NotifyError("Oй!", "Что-то пошло нет так!")
					//})
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС и ЦОД выполнены!\n")
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы.")
						})
						//appendOutput(outputText, "Все операции для ЛВС и ЦОД выполнены!\n")
					} else {
						//fyne.Do(func() {
						appendOutput(outputText, "✅ Все операции для ЛВС выполнены!\n")
						NotifySuccess("Ура!", "Конфиги загружены и отсортированы.")
						//})
						//appendOutput(outputText, "Все операции для ЛВС выполнены!\n")
					}
				}
			}

			if manualRun == false {
				cfg.LastRun = time.Now().Format(time.RFC3339)
			} else {
				manualRun = false
			}

		}

		fyne.Do(func() {
			scroll.ScrollToBottom()
			scroll.Refresh()
			allUnblock()
		})

	}

	cloneBtn.OnTapped = func() {

		if !fileExists(configPath) {
			manualRun = true
		}

		if isAutoRun {
			isAutoRun = false // сбрасываем
			go startDownload()
			return
		}

		// Если config.json существует И расписание настроено → спрашиваем
		if fileExists(configPath) && cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
			dialog.ShowConfirm(
				"Ой!",
				"Загрузить принудительно?\n",
				func(confirmed bool) {
					if confirmed {
						//clearLogKeepHeader(outputText, &output.Entry)
						appendOutput(outputText, "🟢 Запуск по запросу пользователя.\n")
						manualRun = true
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

	sortBox := container.NewHBox(
		container.NewCenter(asIsCheck),
		container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		// ← добавляет отступ вокруг спейсера
		container.NewCenter(platformCheck),
		container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		container.NewCenter(regionCheck),
		container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		container.NewCenter(aclCheck),
		container.NewCenter(parserCheck),
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
		//container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		//container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		//container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		//container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		//container.NewPadded(layout.NewSpacer(), layout.NewSpacer(), layout.NewSpacer()),
		targetBlock,
		//layout.NewSpacer(),
	)
	savePathContainer := container.NewBorder(
		nil, nil, nil, browseBtn, savePathEntry,
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
	//loginLabel := widget.NewLabel("1223")
	// Сборка формы
	locked := container.NewMax(passEntry, blocker)
	form := container.NewVBox(
		//loginLabel,
		loginEntry,
		//passEntry,
		locked,
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
		savePathContainer,
		scheduleEntry,
		timeEntry,
		//testBtn,
	)

	w.SetContent(form)
	w.ShowAndRun()
	form.Refresh()
	w.Canvas().Refresh(form)

}
