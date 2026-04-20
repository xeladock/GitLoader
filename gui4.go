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
	"sync"
	"syscall"
	"time"

	conf "configtool.local/conf"
	"configtool.local/crypt"
	"configtool.local/gloss"
	"configtool.local/progdl"
	"configtool.local/sound"
	"configtool.local/state_var"
	"fyne.io/fyne/v2/driver/desktop"
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

//go:embed icon/icon_white.png
var white_trayIcon []byte

//var firstrun = false

// var SecretKey []byte
var (
	schedulerRunning bool
	schedulerCancel  chan struct{}
	asIsCheck        *widget.Check
	platformCheck    *widget.Check
	regionCheck      *widget.Check

	progressCheck *widget.Check
	updateCheck   *widget.Check

	dcCheck     *widget.Check
	lanCheck    *widget.Check
	aclCheck    *widget.Check
	parserCheck *widget.Check
	scroll      *container.Scroll
	outputText  binding.String
	//IsDownloading atomic.Bool
	//overridePath bool
	//nextRunTime    time.Time
	//targetTime     time.Duration
	//interval       time.Duration
	//cfg            *conf.AppConfig
	//cfg2           *conf.AppConfig
	//cloneBtn       *widget.Button
	//cancel         chan struct{}
	chosenOverPath string
	stateMutex     sync.Mutex
	//schedulerRunning bool
	// ... другие виджеты, если нужноasIsCheck
)

var isAutoRun = false // ← флаг: запущено ли по расписанию
//var manualRun = false

//var overridePath = false
//state_var.OverridePath = false
// Вызов:
//func SaveDoubleEncryptedConfig(cfg *conf.AppConfig, path string, internalKey, externalKey []byte) error {
//	// 1. Шифруем чувствительные поля (внутренний слой)
//	encPass, err := crypt.Encrypt(cfg.GitLabPass, internalKey)
//	if err != nil {
//		return err
//	}
//	encToken, err := crypt.Encrypt(cfg.NetboxToken, internalKey)
//	if err != nil {
//		return err
//	}
//
//	// Создаём копию с зашифрованными полями
//	encCfg := *cfg
//	encCfg.GitLabPass = encPass
//	encCfg.NetboxToken = encToken
//
//	// 2. Сериализуем в JSON
//	jsonData, err := json.MarshalIndent(encCfg, "", "  ")
//	if err != nil {
//		return err
//	}
//
//	// 3. Шифруем весь JSON (внешний слой)
//	encryptedAll, err := crypt.Encrypt(string(jsonData), externalKey)
//	if err != nil {
//		return err
//	}
//
//	// 4. Сохраняем
//	return os.WriteFile(path, []byte(encryptedAll), 0600)
//}

//	func LoadDoubleEncryptedConfig(path string, internalKey, externalKey []byte) (*conf.AppConfig, error) {
//		data, err := os.ReadFile(path)
//		if err != nil {
//			return nil, err
//		}
//
//		// 1. Расшифровываем внешний слой
//		jsonStr, err := crypt.Decrypt(string(data), externalKey)
//		if err != nil {
//			return nil, err
//		}
//
//		// 2. Парсим JSON
//		cfg := &conf.AppConfig{}
//		if err := json.Unmarshal([]byte(jsonStr), cfg); err != nil {
//			return nil, err
//		}
//
//		// 3. Расшифровываем внутренние поля
//		decPass, err := crypt.Decrypt(cfg.GitLabPass, internalKey)
//		if err != nil {
//			return nil, err
//		}
//		decToken, err := crypt.Decrypt(cfg.NetboxToken, internalKey)
//		if err != nil {
//			return nil, err
//		}
//
//		cfg.GitLabPass = decPass
//		cfg.NetboxToken = decToken
//
//		return cfg, nil
//	}
func openFolderInExplorer(path string) {
	if path == "" {
		//appendOutput(outputText, "⚠️ Путь для открытия не задан.\n")
		return
	}

	// Для Linux используем xdg-open — он открывает папку в проводнике по умолчанию
	cmd := exec.Command("xdg-open", path)

	if err := cmd.Start(); err != nil {
		//appendOutput(outputText, "❌ Не удалось открыть папку: "+err.Error()+"\n")
	}
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
	configPath := getConfigPath()
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
	case "acl":
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

	switch cfg.Parser {
	case "true":
		parserCheck.SetChecked(true)
	default:
		parserCheck.SetChecked(false)

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

type hiddenTheme struct{ fyne.Theme }

func (hiddenTheme) ScrollBarSize() int { return 0 }

// 1000
func NotifySuccess(title, message string) {
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "10000", title, message).Run()
	sound.PlayYes()
}

func NotifyError(title, message string) {
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "10000", title, message).Run()
	sound.PlayNo()
	//exec.Command("paplay", "./icon/no.mp3").Run()
	//cmd.Run()
}

// 1014
func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Если не удалось получить домашнюю папку — fallback в текущую директорию
		return "config.json"
	}

	configDir := filepath.Join(home, ".config", "gittornado")

	// Создаём папку, если её нет
	if err := os.MkdirAll(configDir, 0755); err != nil {
		// Если не смогли создать — fallback
		return "config.json"
	}

	return filepath.Join(configDir, "config.json")
}

//func NextTime(cfg *conf.AppConfig) string {
//	if cfg.ScheduleDays > 31 || cfg.ScheduleTime == "" || cfg.ScheduleDays <= 0 {
//		return "❌Ошибка: Нарушение интервала загрузки."
//	}
//
//	parts := strings.Split(cfg.ScheduleTime, ":")
//	if len(parts) != 2 {
//		return "❌Ошибка: Неверный формат времени."
//	}
//
//	hour, _ := strconv.Atoi(parts[0])
//	minute, _ := strconv.Atoi(parts[1])
//
//	now := time.Now()
//	loc := now.Location()
//
//	var nextRun time.Time
//
//	// === Определяем, первый это запуск или повторный ===
//	isFirstRun := cfg.LastRun == "" || cfg.LastRun == "0001-01-01T00:00:00Z"
//
//	if !isFirstRun {
//		// Просто считаем от последнего запуска
//		last, _ := time.Parse(time.RFC3339, cfg.LastRun)
//		nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
//		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)
//
//		for !nextRun.After(now) {
//			nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
//		}
//	} else {
//		// Первый запуск
//		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//		if now.Before(today) {
//			nextRun = today
//		} else {
//			nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
//		}
//	}
//
//	println(isFirstRun)
//	println(cfg.LastRun)
//	// Форматирование задержки
//	delay := time.Until(nextRun)
//	totalMinutes := int(delay.Minutes())
//
//	var delayText string
//	if totalMinutes < 1 {
//		delayText = "меньше минуты"
//	} else {
//		days := totalMinutes / (24 * 60)
//		remaining := totalMinutes % (24 * 60)
//		hours := remaining / 60
//		minutes := remaining % 60
//
//		var parts []string
//		if days > 0 {
//			parts = append(parts, fmt.Sprintf("%d д.", days))
//		}
//		if hours > 0 || days > 0 {
//			parts = append(parts, fmt.Sprintf("%d ч.", hours))
//		}
//		if minutes > 0 || len(parts) == 0 {
//			parts = append(parts, fmt.Sprintf("%d мин.", minutes))
//		}
//		delayText = strings.Join(parts, " ")
//	}
//	//appendOutput(outputText, "▶ Планировщик запущен.\n")
//	return fmt.Sprintf("▶ Планировщик запущен.\n🔄 Следующий запуск: %s в %s. (через %s)",
//		nextRun.Format("02.01.2006"),
//		nextRun.Format("15:04"),
//		delayText)
//}

//func NextTime(cfg *conf.AppConfig) string {
//	if cfg.ScheduleDays <= 0 || cfg.ScheduleDays > 31 || cfg.ScheduleTime == "" {
//		return "❌ Планировщик не настроен"
//	}
//
//	// Парсим время запуска (HH:MM)
//	parts := strings.Split(cfg.ScheduleTime, ":")
//	if len(parts) != 2 {
//		return "❌ Неверный формат времени"
//	}
//
//	hour, _ := strconv.Atoi(parts[0])
//	minute, _ := strconv.Atoi(parts[1])
//
//	now := time.Now()
//	loc := now.Location()
//
//	var nextRun time.Time
//
//	// === Основная надёжная логика ===
//	if cfg.LastRun == "" || cfg.LastRun == "0001-01-01T00:00:00Z" || time.Since(parseLastRun(cfg.LastRun)) < 10*time.Minute {
//		// Первый запуск вообще
//		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//
//		if now.Before(today) {
//			nextRun = today
//		} else {
//			nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
//		}
//	} else {
//		// Повторный запуск — считаем от LastRun
//		last, err := time.Parse(time.RFC3339, cfg.LastRun)
//		if err != nil {
//			// Если LastRun повреждён — считаем как первый запуск
//			today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//			if now.Before(today) {
//				nextRun = today
//			} else {
//				nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
//			}
//		} else {
//			// Нормальный расчёт от последнего запуска
//			nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
//			nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)
//
//			// Если рассчитанное время уже в прошлом — добавляем интервал
//			for !nextRun.After(now) {
//				nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
//			}
//		}
//	}
//
//	// === Форматирование задержки ===
//	delay := time.Until(nextRun)
//	totalMinutes := int(delay.Minutes())
//
//	var timeStr string
//
//	if totalMinutes < 1 {
//		timeStr = "(До запуска меньше минуты)"
//	} else {
//		days := totalMinutes / (24 * 60)
//		remaining := totalMinutes % (24 * 60)
//		hours := remaining / 60
//		minutes := remaining % 60
//
//		var parts []string
//		if days > 0 {
//			parts = append(parts, fmt.Sprintf("%d д.", days))
//		}
//		if hours > 0 || days > 0 {
//			parts = append(parts, fmt.Sprintf("%d ч.", hours))
//		}
//		if minutes > 0 || len(parts) == 0 {
//			parts = append(parts, fmt.Sprintf("%d мин.", minutes))
//		}
//		timeStr = strings.Join(parts, " ")
//	}
//
//	return fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %s)",
//		nextRun.Format("02.01.2006"),
//		nextRun.Format("15:04"),
//		timeStr)
//}

// const configPath = "config.json"
//
//	func NextTime(cfg *conf.AppConfig) string {
//		if cfg == nil {
//			return "Конфигурация не загружена"
//		}
//		if cfg.ScheduleDays <= 0 || cfg.ScheduleDays > 31 || cfg.ScheduleTime == "" {
//			return "❌ Ошибка: Нарушение интервала загрузки."
//		}
//
//		parts := strings.Split(cfg.ScheduleTime, ":")
//		if len(parts) != 2 {
//			return "❌ Ошибка: Неверный формат времени в настройках"
//		}
//
//		hour, _ := strconv.Atoi(parts[0])
//		minute, _ := strconv.Atoi(parts[1])
//
//		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
//
//			return "❌ Ошибка: Некорректное время в настройках."
//		}
//
//		now := time.Now()
//		loc := now.Location()
//
//		//fmt.Printf("DEBUG NextTime: now=%s, ScheduleTime=%s, LastRun=%s\n",
//		//	now.Format("15:04"), cfg.ScheduleTime, cfg.LastRun)
//
//		var nextRun time.Time
//
//		// ЖЁСТКОЕ условие: считаем первый запуск, если LastRun пустой ИЛИ очень свежий (меньше 30 минут)
//		if cfg.LastRun == "" ||
//			cfg.LastRun == "0001-01-01T00:00:00Z" {
//
//			today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//
//			if now.Before(today) {
//				nextRun = today
//				//fmt.Println("DEBUG: Первый запуск → СЕГОДНЯ")
//			} else {
//				nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
//				//fmt.Println("DEBUG: Первый запуск → ЗАВТРА")
//			}
//		} else {
//			// Повторный запуск
//			last, _ := time.Parse(time.RFC3339, cfg.LastRun)
//			nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
//			nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)
//
//			for !nextRun.After(now) {
//				nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
//			}
//			//fmt.Println("DEBUG: Повторный запуск по LastRun")
//		}
//
//		//fmt.Printf("DEBUG: nextRun = %s\n", nextRun.Format("02.01.2006 15:04"))
//
//		delay := time.Until(nextRun)
//		totalMinutes := int(delay.Minutes())
//
//		if totalMinutes < 1 {
//			return fmt.Sprintf("▶ Планировщик запущен.\n"+
//				"🔄 Следующий запуск: %s в %s. (До запуска меньше минуты)",
//				nextRun.Format("02.01.2006"),
//				nextRun.Format("15:04"),
//			)
//		}
//
//		days := totalMinutes / (24 * 60)
//		remainingMinutes := totalMinutes % (24 * 60)
//		hours := remainingMinutes / 60
//		minutes := remainingMinutes % 60
//
//		var timeParts []string
//
//		if days > 0 {
//			timeParts = append(timeParts, fmt.Sprintf("%d д.", days))
//		}
//
//		if hours > 0 {
//			timeParts = append(timeParts, fmt.Sprintf("%d ч.", hours))
//		}
//
//		if minutes > 0 || len(timeParts) == 0 {
//			timeParts = append(timeParts, fmt.Sprintf("%d мин.", minutes))
//		}
//
//		timeStr := strings.Join(timeParts, " ")
//
//		return fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %s)",
//			nextRun.Format("02.01.2006"),
//			nextRun.Format("15:04"),
//			timeStr,
//		)
//	}
func NextTime(cfg *conf.AppConfig) string {
	if cfg.ScheduleDays > 31 || cfg.ScheduleTime == "" || cfg.ScheduleDays <= 0 {
		return "❌ Ошибка: Нарушение интервала загрузки."
	}

	parts := strings.Split(cfg.ScheduleTime, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	//parts := strings.Split(cfg.ScheduleTime, ":")
	if len(parts) != 2 {
		return "❌ Ошибка: Неверный формат времени в настройках."
	}

	//hour, _ := strconv.Atoi(parts[0])
	//minute, _ := strconv.Atoi(parts[1])

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return "❌ Ошибка: Некорректное время в настройках."
	}

	now := time.Now()
	loc := now.Location()

	var nextRun time.Time

	if cfg.RunCount == 0 {
		// Первый запуск в истории
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if now.Before(today) {
			nextRun = today
		} else {
			nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
		}
	} else {
		// Обычный повторный запуск
		last, err := time.Parse(time.RFC3339, cfg.LastRun)
		if err != nil {
			last = now.Add(-365 * 24 * time.Hour) // fallback
		}
		nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)

		for !nextRun.After(now) {
			nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
		}
	}

	// Форматирование задержки (твой текущий код)
	delay := time.Until(nextRun)
	totalMinutes := int(delay.Minutes())

	var delayText string
	if totalMinutes <= 1 {
		delayText = "(До запуска меньше минуты)"
	} else {
		days := totalMinutes / (24 * 60)
		remaining := totalMinutes % (24 * 60)
		hours := remaining / 60
		minutes := remaining % 60

		var parts []string
		if days > 0 {
			parts = append(parts, fmt.Sprintf("%d д.", days))
		}
		if hours > 0 || days > 0 {
			parts = append(parts, fmt.Sprintf("%d ч.", hours))
		}
		if minutes > 0 || len(parts) == 0 {
			parts = append(parts, fmt.Sprintf("%d мин.", minutes))
		}
		delayText = strings.Join(parts, " ")
	}

	return fmt.Sprintf("▶ Планировщик запущен.\n"+"🔄 Следующий запуск: %s в %s. (через %s)",
		nextRun.Format("02.01.2006"),
		nextRun.Format("15:04"),
		delayText)
}

// 10111
// Вспомогательная функция для безопасного парсинга LastRun
func parseLastRun(lastRunStr string) time.Time {
	t, err := time.Parse(time.RFC3339, lastRunStr)
	if err != nil {
		return time.Time{} // нулевое время
	}
	return t
}

//func NextTime(cfg *conf.AppConfig) string {
//	if cfg == nil {
//		return "Конфигурация не загружена"
//	}
//	if cfg.ScheduleDays <= 0 || cfg.ScheduleTime == "" {
//		return "Планировщик не настроен"
//	}
//
//	// Парсим scheduleTime (формат "HH:MM")
//	parts := strings.Split(cfg.ScheduleTime, ":")
//
//	if len(parts) != 2 {
//		return "Неверный формат времени в настройках"
//	}
//
//	hour, errH := strconv.Atoi(parts[0])
//	minute, errM := strconv.Atoi(parts[1])
//	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
//		//cfg.SchedulerState = "paused"
//		//_ = saveConfig(cfg, configPath)
//		return "Некорректное время в настройках"
//	}
//
//	now := time.Now()
//	loc := now.Location()
//
//	// Рассчитываем время следующего запуска
//	var nextRun time.Time
//
//	if cfg.LastRun == "" {
//		// Первый запуск — сегодня в указанное время
//		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//		if now.After(today) || now.Equal(today) {
//			nextRun = today.AddDate(0, 0, cfg.ScheduleDays)
//		} else {
//			nextRun = today
//		}
//	} else {
//		// Есть LastRun — считаем от него
//		last, err := time.Parse(time.RFC3339, cfg.LastRun)
//		if err != nil {
//			return "Ошибка парсинга LastRun"
//		}
//
//		nextRun = last.AddDate(0, 0, cfg.ScheduleDays)
//		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)
//
//		// Если nextRun в прошлом — добавляем дни
//		for !nextRun.After(now) {
//			nextRun = nextRun.AddDate(0, 0, cfg.ScheduleDays)
//		}
//	}
//
//	delay := time.Until(nextRun)
//
//	// Расчёт в днях, часах и минутах
//	totalMinutes := int(delay.Minutes())
//
//	if totalMinutes < 1 {
//		return fmt.Sprintf("▶ Планировщик запущен.\n"+
//			"🔄 Следующий запуск: %s в %s. ( До запуска меньше минуты. )",
//			nextRun.Format("02.01.2006"),
//			nextRun.Format("15:04"),
//		)
//	}
//
//	days := totalMinutes / (24 * 60)
//	remainingMinutes := totalMinutes % (24 * 60)
//	hours := remainingMinutes / 60
//	minutes := remainingMinutes % 60
//
//	// Формируем строку "через X д. Y ч. Z мин."
//	var timeParts []string
//	if days > 0 {
//		timeParts = append(timeParts, fmt.Sprintf("%d д.", days))
//	}
//	if hours > 0 || days > 0 { // показываем часы, если есть дни или часы > 0
//		timeParts = append(timeParts, fmt.Sprintf("%d ч.", hours))
//	}
//	if minutes > 0 || len(timeParts) == 0 { // минуты всегда, если ничего другого нет
//		timeParts = append(timeParts, fmt.Sprintf("%d мин.", minutes))
//	}
//
//	timeStr := strings.Join(timeParts, " ")
//	//println(timeStr)
//	return fmt.Sprintf(
//		"🔄 Следующий запуск: %s в %s. (через %s)",
//		nextRun.Format("02.01.2006"),
//		nextRun.Format("15:04"),
//		timeStr,
//	)
//}

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

//	func updateTrayIcon() {
//		if fyne.CurrentApp().Settings().Theme() == theme.DarkTheme() {
//			systray.SetIcon(darkIcon.StaticContent())
//			.SetIcon(fyne.NewStaticResource("icon.png", trayIcon))
//		} else {
//			systray.SetIcon(lightIcon.StaticContent())
//		}
//	}
//
// 1001
type NoContextMenuEntry struct {
	widget.Entry
}

func NewNoContextMenuEntry() *NoContextMenuEntry {
	e := &NoContextMenuEntry{}
	e.ExtendBaseWidget(e)
	//e.MultiLine = true
	//e.Wrapping = fyne.TextWrapWord
	//e.TextStyle = fyne.TextStyle{Monospace: true}
	return e
}

// Основной способ отключения контекстного меню в v2.7.2
func (e *NoContextMenuEntry) TappedSecondary(pe *fyne.PointEvent) {
	// Ничего не делаем — меню не появляется
}

// На всякий случай перехватываем и MouseDown
//
//	func (e *NoContextMenuEntry) MouseDown(ev *desktop.MouseEvent) {
//		if ev.Button == desktop.RightMouseButton {
//			return
//		}
//		e.Entry.MouseDown(ev)
//	}

// подбери под ширину твоего окна (примерно 120-150)

func errorlog(output binding.String, scroll *container.Scroll, text string) {

	const maxLineWidth = 115

	fyne.Do(func() {
		current, _ := output.Get()

		lines := strings.Split(text, "\n")
		var sb strings.Builder

		for _, line := range lines {
			if len(line) > maxLineWidth {
				// Переносим по словам, стараясь не разрывать слово
				for len(line) > maxLineWidth {
					// Ищем последний пробел в пределах допустимой длины
					cut := line[:maxLineWidth]
					lastSpace := strings.LastIndex(cut, " ")

					if lastSpace > 10 { // если нашли хороший пробел
						sb.WriteString(cut[:lastSpace] + "\n")
						line = cut[lastSpace+1:] + line[maxLineWidth:]
					} else {
						// Нет хорошего пробела — режем жёстко с дефисом
						sb.WriteString(line[:maxLineWidth-1] + "-\n")
						line = line[maxLineWidth-1:]
					}
				}
				sb.WriteString(line + "\n")
			} else {
				sb.WriteString(line + "\n")
			}
		}

		newText := current + sb.String()
		output.Set(newText)

		scroll.ScrollToBottom()
		scroll.Refresh()
	})
}

func main() {
	//println(state_var.ManRun.Load())
	configPath := getConfigPath()
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
		os.Exit(1)
	}

	//}
	//1005
	if oldPID > 0 {
		if !isProcessRunning(oldPID) {
			//println("Найден старый lock от мёртвого процесса — удаляем")
			os.Remove(lockPath)
		} else {
			exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "2000", "Oй!", "Рестарт процесса!").Run()
			killProcess(oldPID)
			time.Sleep(1500 * time.Millisecond)
			os.Remove(lockPath)
			//println("Программа уже запущена (PID:", oldPID, ")")
			//os.Exit(1)
		}
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

	//fmt.Println("Программа запущена успешно (PID:", os.Getpid(), ")")
	a := app.NewWithID("gittornado")
	a.Settings().SetTheme(theme.LightTheme())

	a.SetIcon(fyne.NewStaticResource("icon_white.png", white_trayIcon))
	w := a.NewWindow("GitTornado")
	//860
	w.Resize(fyne.NewSize(760, 250))
	w.SetFixedSize(true)
	w.SetOnClosed(nil)
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
			//if len(trayIcon) > 0 {
			systray.SetIcon(trayIcon)
			systray.SetTitle("GitTornado")
			systray.SetTooltip("GitTornado v1.2")
			//999
			open := systray.AddMenuItem("Открыть", "Показать программу")
			quit := systray.AddMenuItem("Выход", "Закрыть программу")

			go func() {
				for range open.ClickedCh {
					//1100
					fyne.Do(func() {
						//println(state_var.IsActive.Load())
						if !state_var.IsActive.Load() {
							loadConfigFromFile()
							outputText.Set("")
							updateHint()
							if fileExists(configPath) && cfg.SchedulerState == "running" {
								appendOutput(outputText, NextTime(cfg))
							} else if fileExists(configPath) && cfg.SchedulerState == "paused" {
								appendOutput(outputText, "⚠️ ВНИМАНИЕ! Планировщик не активен. Нажмите «Старт» для возобновления.")

							}

							w.Show()
							w.RequestFocus()
							w.Canvas().Focus(nil)
							scroll.Refresh()
						} else {
							w.Show()
						}

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
						a.Quit() // ← завершаем приложение (БЕЗ ОШИБКИ!)
					})
					systray.Quit() // ← это можно вне UI-потока
				}
			}()
		}, func() {})
	}()

	// === END SYSTRAY ===

	// Поля ввода
	//output := NewReadOnlyEntry()
	//4444
	//loginEntry := widget.NewEntry()
	//loginEntry := NewReadOnlyEntry2()
	loginEntry := NewNoContextMenuEntry()
	loginEntry.SetPlaceHolder("Введите логин GitLab")
	//loginEntry.SetText("") // очищаем возможный мусор
	//loginEntry.Refresh()
	//loginEntry.Bind(loginBinding)
	//loginEntry.TextStyle = fyne.TextStyle{}
	//blocker := widget.NewLabel("") // перехватывает мышь
	//blocker.Resize(loginEntry.Size())

	//passEntry := widget.NewEntry()
	passEntry := NewNoContextMenuEntry()
	passEntry.Password = true
	passEntry.SetPlaceHolder("Введите пароль GitLab")
	//passEntry.TextStyle = fyne.TextStyle{}

	netboxEntry := NewNoContextMenuEntry()
	netboxEntry.SetPlaceHolder("Введите API NetBox Token")
	netboxEntry.Password = true
	//netboxEntry.Disable()

	//выбор папки

	//savePathEntry := widget.NewEntry()
	savePathEntry := NewNoContextMenuEntry()
	savePathEntry.SetPlaceHolder("Папка сохранения. По-умолчанию - текущая")
	//888
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
	scheduleEntry := NewNoContextMenuEntry()
	//scheduleEntry := widget.NewEntry()
	scheduleEntry.SetPlaceHolder("Интервал (дней, 1–31). По-умолчанию - 1")
	scheduleEntry.Resize(fyne.NewSize(140, 40))
	// Поле выбора времени
	timeEntry := NewNoContextMenuEntry()
	//timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("Время обновления (HH:MM). По-умолчанию - 00:00")
	//aaaa
	outputText = binding.NewString()
	//outputText.Wrapping = fyne.TextWrapWord
	//output := widget.NewMultiLineEntry()
	//1012
	//output := widget.NewMultiLineEntry()
	output := NewReadOnlyEntry()
	output.MultiLine = true
	//output.Disable()

	output.Bind(outputText)
	//output.Wrapping = fyne.TextWrapWord
	//output.TextStyle = fyne.TextStyle{Monospace: false}
	//output.Wrapping = fyne.TextWrapWord
	//output.SetMinRowsVisible(15)
	//scroll := container.NewVScroll(output)
	scroll = container.NewVScroll(output)
	//fixed := container.NewMax(scroll)
	//fixed := container.NewBorder(nil, nil, nil, nil, scroll)
	//fixed.SetMinSize(fyne.NewSize(0, 250))
	//scroll.SetHorizontalScroll(false)
	//scroll.Direction = container.ScrollVerticalOnly
	//scroll.SetMinSize(fyne.NewSize(600, 300))
	//scroll.SetMinSize(fyne.NewSize(600, 300))  // твой размер
	//scroll.Resize(fyne.NewSize(600, 300))
	//scroll.SetOverlayScrollbars(false)
	//1025
	scroll.SetMinSize(fyne.NewSize(0, 250))
	//scroll.Size()

	//fixed := container.NewBorder(nil, nil, nil, nil, scroll)
	//fixed.SetMinSize(fyne.NewSize(760, 250))
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
	//if fileExists(configPath) && cfg.SchedulerState == "paused" {
	//	appendOutput(outputText, "⚠️ Планировщик не активен! Нажмите «Старт» для запуска.")
	//}

	//сюда блок обзора
	//PauseUpdateButtonState(startPauseBtn, cfg, configPath)
	//9999
	//записываем настройки
	setConfig := func() {
		stateMutex.Lock()
		defer stateMutex.Unlock()
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
			//savePathEntry.SetPlaceHolder(" ✅ Текущая папка сохранения") // показываем пользователю
		} else if chosenPath[0] != '/' {
			dialog.ShowInformation(
				"Ой!",
				"Укажите корректный путь загрузки.\n",
				w,
			)
			return // или continue — в зависимости от контекста
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
				"Введите корректное время в формате ЧЧ:ММ.\n",
				w,
			)

			return

		} else {
			cfg.ScheduleTime = strings.TrimSpace(timeEntry.Text)
		}

		cfg.LastRun = time.Now().Format("2006-01-02T15:04:05Z07:00")
		cfg.RunCount = 0
		configPath := getConfigPath()
		if err := saveConfig(cfg, configPath); err != nil {
			errorlog(outputText, scroll, fmt.Sprintf("❌ Ошибка сохранения: %v", err))
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
			println(cfg.SchedulerState + " первый save")
			//time.Sleep(150 * time.Millisecond)
			cfg.SchedulerState = "paused" // явно ставим "paused" после сохранения настроек
			schedulerRunning = false

			if err := saveConfig(cfg, configPath); err != nil {
				appendOutput(outputText, "❌ Ошибка сохранения: "+err.Error()+"\n")
				return
			}
			println(" 0. SetConfig. cfg.SchedulerState is " + cfg.SchedulerState)
			fyne.Do(func() {
				PauseUpdateButtonState(startPauseBtn, cfg, configPath)
			})
			//_ = saveConfig(cfg, configPath)
			println(" 1. SetConfig. cfg.SchedulerState is " + cfg.SchedulerState)

			browseBtn.Disable()
			//}
			outputText.Set("")
			updateHint()
			appendOutput(outputText, "✅ Настройки сохранены. Нажмите «Старт» для запуска планировщика.")
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
	//outputView := widget.NewLabelWithData(outputText)
	//сбросить конфигу
	resetConfig := func() {
		stateMutex.Lock()
		defer stateMutex.Unlock()

		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			errorlog(outputText, scroll, "❌ Ошибка сброса настроек: "+err.Error()+"\n")
			//outputView.SetText(outputView.Text + "\n❌ Ошибка сброса настроек: " + err.Error())
			return
		}
		cfg.SchedulerState = "paused"
		//cfg.LastRun = ""
		schedulerRunning = false // ← обязательно сбрасываем флаг
		schedulerCancel = nil
		//cfg := &conf.AppConfig{}
		//loginEntry.SetEditable(true)
		//println("login editable in reset")
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

		//println("first run в reset", firstrun)

		//firstrun = true

		//cfg = &conf.AppConfig{}
		fyne.Do(func() {
			//time.Sleep(80 * time.Millisecond) // даём время на запись и стабилизацию
			PauseUpdateButtonState(startPauseBtn, cfg, configPath)
		})
		//println("5. resetConfig. cfg.SchedulerState is ", cfg.SchedulerState)
		updateButtonState()
		//println(cfg.SchedulerState, "на что смотрю")
		//unblockInputs()
	}
	//1028
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
			setConfig()
			//time.Sleep(500 * time.Millisecond)
			//функция установки
			//blockInputs()

		}
	}
	//1027
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
	//111
	//modeRow := container.NewHBox(
	//	label,
	//	layout.NewSpacer(),
	//	modeRadio, // справа — радиокнопки
	//)
	//modeCard2 := widget.NewCard("", "", modeRow)
	//modeRadioContainer := container.NewCenter(modeCard2)
	//1003
	println("manrun is ", state_var.ManRun.Load())
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
		//
		//if manualRun && isAutoRun {
		//	println("ЗАПУСК ОБОИХ")
		//	//cfg.LastRun = time.Now().Format(time.RFC3339)
		//	return
		//}

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
		if state_var.ManRun.Load() {
			appendOutput(outputText, "🟢 Запуск по запросу пользователя\n")
			time.Sleep(500 * time.Millisecond)
			appendOutput(outputText, "🔥 Подключаемся и загружаем файлы из gitlab...\n")
			//NotifySuccess("раз!", "два")
			//exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "10000", "раз", "два").Run()

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
		if state_var.OverridePath {
			state_var.OverridePath = false
			path = chosenOverPath
		} else if path == "" && cfg.SavedPlace != "" {
			path = cfg.SavedPlace
		}
		//555

		os.RemoveAll(filepath.Join(path, targetDir))

		dcbool := dcCheck.Checked
		lanbool := lanCheck.Checked
		//println("данные на цикл: ", dcbool, lanbool)
		//!загрузка ЦОД
		state_var.IsActive.Store(true)
		defer state_var.IsActive.Store(false)

		if dcbool {
			//dc := true
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
					appendOutput(outputText, "Ошибка: неверный логин или пароль GitLab.\n")
					os.RemoveAll(filepath.Join(path, targetDir))
				case strings.Contains(string(outputBytes), "not found"):
					appendOutput(outputText, "Ошибка: git не найден.\n")

				case strings.Contains(string(outputBytes), "Could not resolve host"):
					appendOutput(outputText, "Ошибка: нет интернета или сервер недоступен.\n")

				case strings.Contains(string(outputBytes), "Repository not found"):
					appendOutput(outputText, "Ошибка: репозиторий не найден или нет доступа.\n")

				case err != nil && (os.IsPermission(err) ||
					strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
					strings.Contains(strings.ToLower(err.Error()), "access denied") ||
					strings.Contains(strings.ToLower(err.Error()), "read-only file system")):
					appendOutput(outputText, "Ошибка: нет прав на запись в целевую папку.\n")

				default:
					errorlog(outputText, scroll, "Ошибка git clone: "+err.Error()+"\n")
				}
				fyne.Do(allUnblock)

				return
			}
			//1007
			if err := RemoveGitFolder(dstDir, outputText); err != nil {
				return
			}

			// ВСЁ НОРМА — запускаем нужный режим
			isProgressMode := progressCheck.Checked

			if isProgressMode {
				//appendOutput(outputText, "Режим: Сохранение истории (архив по дням)\n")
				if err := progdl.RunProgressMode(dstDir, path, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcbool, false, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, state_var.ManRun.Load()); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						errorlog(outputText, scroll, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло не так!")
					})
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "🟢 Файлы ЦОД обработаны. Ожидаем файлы ЛВС.\n")
							scroll.ScrollToBottom()
						})
						//fyne.Do(func() {
						//	scroll.ScrollToBottom()
						//	scroll.Refresh()
						//})
					} else {
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЦОД выполнены!\n")
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы!")
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
				println("manrun in update mode", state_var.ManRun.Load())
				//appendOutput(outputText, "Режим: Обновление текущих файлов\n")
				if err := progdl.RunUpdateMode(dstDir, sdDst, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, dcbool, false, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, state_var.ManRun.Load()); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						errorlog(outputText, scroll, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло не так!")
					})
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "🟢 Файлы ЦОД обработаны. Ожидаем файлы ЛВС.\n")
							scroll.ScrollToBottom()
						})
					} else {
						//appendOutput(outputText, "Все операции для ЦОД выполнены!\n")
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЦОД выполнены!\n")
							scroll.ScrollToBottom()
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы!")
						})
					}
				}

			}
			if dcCheck.Checked && lanCheck.Checked {
			} else {
				if state_var.ManRun.Load() == false {
					cfg.RunCount++
					cfg.LastRun = time.Now().Format(time.RFC3339)
					_ = saveConfig(cfg, configPath)
				} else {
					state_var.ManRun.Store(false)
					//state_var.ManRun.CompareAndSwap(true, false)
				}
				fyne.Do(func() {
					scroll.ScrollToBottom()
					scroll.Refresh()
					allUnblock()
				})
			}
			dcbool = false
		}

		if lanbool {

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
					os.RemoveAll(filepath.Join(path, targetDir))
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
					errorlog(outputText, scroll, "Ошибка git clone: "+err.Error()+"\n")
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
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, false, lanbool, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, state_var.ManRun.Load()); err != nil {
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
					fyne.Do(func() {
						errorlog(outputText, scroll, "Ошибка выполнения: "+err.Error()+"\n")
						NotifyError("Oй!", "Что-то пошло не так!")
					})
				} else {
					//1002
					if dcCheck.Checked && lanCheck.Checked {
						//appendOutput(outputText, "Все операции для ЛВС и ЦОД выполнены!\n")
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС и ЦОД выполнены!\n")
							scroll.ScrollToBottom()
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы!")
						})
					} else {
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС выполнены!\n")
							scroll.ScrollToBottom()
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы!")
						})
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
				if err := progdl.RunUpdateMode(dstDir, sdDst, cfg,
					asIsCheck.Checked, platformCheck.Checked, regionCheck.Checked, false, lanbool, aclCheck.Checked, parserCheck.Checked,
					token, outputText, scroll, state_var.ManRun.Load()); err != nil {
					//fyne.Do(func() {
					errorlog(outputText, scroll, "Ошибка выполнения: "+err.Error()+"\n")
					NotifyError("Oй!", "Что-то пошло не так!")
					//})
					//appendOutput(outputText, "Ошибка выполнения: "+err.Error()+"\n")
				} else {
					if dcCheck.Checked && lanCheck.Checked {
						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС и ЦОД выполнены!\n")
							scroll.ScrollToBottom()
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы.")
						})
						//appendOutput(outputText, "Все операции для ЛВС и ЦОД выполнены!\n")
					} else {

						fyne.Do(func() {
							appendOutput(outputText, "✅ Все операции для ЛВС выполнены!\n")
							scroll.ScrollToBottom()
							NotifySuccess("Ура!", "Конфиги загружены и отсортированы.")
							//exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "2000", "1", "2").Run()
							//NotifySuccess("Ура!", "Конфиги загружены и отсортированы.")
						})
						//appendOutput(outputText, "Все операции для ЛВС выполнены!\n")
					}
				}
			}

			//1002
			if state_var.ManRun.Load() == false {
				cfg.RunCount++
				cfg.LastRun = time.Now().Format(time.RFC3339)
				_ = saveConfig(cfg, configPath)
			} else {
				state_var.ManRun.Store(false)
				//state_var.ManRun.CompareAndSwap(true, false)
			}
			lanbool = false
		}

		fyne.Do(func() {
			//println(state_var.ManRun.Load())
			scroll.ScrollToBottom()
			scroll.Refresh()
			allUnblock()
		})
	}

	cloneBtn.OnTapped = func() {
		//println(manualRun, "manualrun в начале процесса")
		if !fileExists(configPath) {
			//manualRun = true
			state_var.ManRun.Store(true)

		}

		if isAutoRun {
			isAutoRun = false // сбрасываем
			go startDownload()
			return
		}

		//if isAutoRun {
		//	println(manualRun, "manualrun в авторан")
		//	if manualRun {
		//		println("запущено в ручную")
		//		cfg.LastRun = time.Now().Format(time.RFC3339)
		//		return
		//	} else {
		//		isAutoRun = false
		//		println(manualRun, "manualrun в конце выбора") // сбрасываем
		//		go startDownload()
		//		return
		//	}
		//}
		//1009
		// Если config.json существует И расписание настроено → спрашиваем
		if fileExists(configPath) && cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
			dialog.ShowConfirm(
				"Ой!",
				"Планировщик настроен. Загрузить принудительно?\n",

				func(confirmed bool) {
					state_var.ManRun.Store(true)
					//state_var.ManRun.CompareAndSwap(false, true)
					//println(manualRun, "manualrun перед выбором")
					//manualRun = true
					//println(manualRun, "manualrun после выбора")
					if confirmed {
						// Пользователь сказал "Да" → спрашиваем про перезапись
						dialog.ShowConfirm(
							"Ой!",
							"Сохранить файлы в назначенной папке планировщика?\n",
							//"Да → перезаписать\n"+
							//"Нет → выбрать другую папку",
							func(overwrite bool) {

								if overwrite {
									go startDownload()
								} else {
									state_var.OverridePath = true
									//state_var.ManRun.Store(false)
									//overridePath = true
									// Нет — выбираем папку
									dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
										if err != nil || uri == nil {
											//state_var.ManRun.CompareAndSwap(true, false)
											state_var.ManRun.Store(false)
											state_var.OverridePath = false
											//appendOutput(outputText, "Выбор папки отменён.\n")
											//state_var.ManRun.Store(false)
											//manualRun = false
											//println(manualRun, "manualrun в конце выбора")
											return
										}

										chosenOverPath = uri.Path()
										//println(chosenOverPath)
										//newPathEntry.SetText(chosenPath)
										//chosenPath := uri.Path()

										//444
										// ← Здесь запускаем с кастомным путём
										//manualRun = true
										//println(manualRun, "manualRun из второго выбора")
										go startDownload()
									}, w)
								}
							},
							w,
						)
					}
				},
				w,
			)
		} else {
			// Если планировщика нет — просто запускаем
			go startDownload()
		}
	}

	cloneBtn.Resize(fyne.NewSize(140, 40))
	cloneButtonContainer := container.NewHBox(layout.NewSpacer(), cloneBtn, layout.NewSpacer())

	saveBtn.Resize(fyne.NewSize(140, 40))
	startPauseBtn = CreateStartPauseButton(cfg, configPath, func() { cloneBtn.OnTapped() }, outputText, scroll, w)
	//schedulerRunning = false
	fyne.Do(func() {
		time.Sleep(80 * time.Millisecond) // даём время на запись и стабилизацию
		PauseUpdateButtonState(startPauseBtn, cfg, configPath)
	})
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
	//locked := container.NewMax(passEntry, blocker)
	form := container.NewVBox(

		//loginLabel,
		loginEntry,
		passEntry,
		//locked,
		netboxEntry,
		sortBlock,
		passtargetButtonContainer,
		//passBlock,
		//centeredModeBox,
		//modeRadioContainer,

		cloneButtonContainer,
		scroll,
		//fixed,
		//saveButtonContainer := container.NewHBox(layout.NewSpacer(),  saveBtn,  startPauseBtn  )
		saveButtonContainer,
		savePathContainer,
		scheduleEntry,
		timeEntry,
		//testBtn,
	)
	//root := container.NewBorder(nil, nil, nil, nil, form)
	//form = container.NewBorder(nil, nil, nil, nil, scroll)
	w.SetContent(form)
	shortcut := &desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
	}

	w.Canvas().AddShortcut(shortcut, func(s fyne.Shortcut) {
		//config_files_clear /
		openFolderInExplorer(cfg.SavedPlace)
		//println("проверка")
	})

	closeshortcut := &desktop.CustomShortcut{
		KeyName:  fyne.KeyEscape,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
	}

	w.Canvas().AddShortcut(closeshortcut, func(s fyne.Shortcut) {
		w.Hide()
	})

	openshortcut := &desktop.CustomShortcut{
		KeyName:  fyne.KeyF1,
		Modifier: fyne.KeyModifierControl | fyne.KeyModifierShift,
	}

	w.Canvas().AddShortcut(openshortcut, func(s fyne.Shortcut) {
		gloss.ShowGlossary(w)
	})

	//w.SetFixedSize(true)
	w.ShowAndRun()
	form.Refresh()
	w.Canvas().Refresh(form)
	//	e4c732fd39ceed92b1e87931e78db912d71c33d3

}
