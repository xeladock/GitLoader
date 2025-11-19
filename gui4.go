package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"configtool.local/asis"
	"configtool.local/platform"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// var (
//
//	outputText     *widget.Entry
//	asIsCheck      *widget.Check
//	platformCheck  *widget.Check
//	regionCheck    *widget.Check
//	saveBtn        *widget.Button
//
// )

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

//func hashString(s string) string {
//	h := sha256.Sum256([]byte(s))
//	return hex.EncodeToString(h[:])
//}

func saveConfig(cfg AppConfig, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}
func loadConfig(path string) (*AppConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg AppConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type AppConfig struct {
	GitLabLogin  string `json:"gitlab_login"`
	GitLabPass   string `json:"gitlab_pass_hash"`
	NetboxToken  string `json:"netbox_token_hash"`
	Mode         string `json:"mode"` // as-is / platform / region
	ScheduleDays int    `json:"schedule_days"`
	ScheduleTime string `json:"schedule_time"` // "HH:MM"
}

func main() {

	const configPath = "config.json"

	var cfg *AppConfig
	cfg, err := loadConfig(configPath)
	if err != nil {
		// файла нет — просим пользователя ввести данные
		cfg = &AppConfig{}
	}

	repoURL := "https://configs.net.rt.ru/dc/configs.git"
	targetDir := "./configs" // куда клонируем репо
	sortedDst := "./config_files_clear"

	a := app.New()
	a.Settings().SetTheme(theme.LightTheme())
	w := a.NewWindow("GitLab Downloader")
	w.Resize(fyne.NewSize(900, 700))

	// Поля ввода
	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("Введите логин GitLab")

	passEntry := widget.NewEntry()
	passEntry.Password = true
	passEntry.SetPlaceHolder("Введите пароль GitLab")

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
	output := widget.NewMultiLineEntry()
	output.Bind(outputText)
	output.SetMinRowsVisible(15)
	scroll := container.NewVScroll(output)
	scroll.SetMinSize(fyne.NewSize(860, 400))
	//scroll.Offset = fyne.NewPos(0, scroll.Content.Size().Height)
	scroll.Refresh()
	scroll.ScrollToBottom()

	asIsCheck := widget.NewCheck("Как есть", nil)
	platformCheck := widget.NewCheck("Платформа", nil)
	regionCheck := widget.NewCheck("Регион", nil)

	updateChecks := func(selected string) {
		asIsCheck.SetChecked(selected == "as-is")
		platformCheck.SetChecked(selected == "platform")
		regionCheck.SetChecked(selected == "region")

		switch selected {
		case "as-is":
			_ = outputText.Set("📂 Режим: хранить файлы как есть (без сортировки)\n")
		case "platform":
			_ = outputText.Set("🔎 Режим: сортировка по платформам (NetBox)\n")
		case "region":
			_ = outputText.Set("📍 Режим: сортировка по регионам\n")
		}
	}

	// Обработчики кликов:
	asIsCheck.OnChanged = func(checked bool) {
		if checked {
			updateChecks("as-is")
		} else {
			_ = outputText.Set("") // все выключены
		}
	}

	platformCheck.OnChanged = func(checked bool) {
		if checked {
			updateChecks("platform")
		} else {
			_ = outputText.Set("") // все выключены
		}
	}

	regionCheck.OnChanged = func(checked bool) {
		if checked {
			updateChecks("region")
		} else {
			_ = outputText.Set("") // все выключены
		}
	}

	//asIsCheck.OnChanged = func(checked bool) {
	//	if checked {
	//		updateChecks("as-is")
	//	} else {
	//		updateChecks("") // все выключены
	//	}
	//}
	//конец нового формата

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
		scheduleEntry.SetText(fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)")
	}
	if cfg.ScheduleTime != "" {
		timeEntry.SetText(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
	}

	saveBtn := widget.NewButton("Сохранить", nil)

	//btn := widget.NewButton("Добавить строку", func() {
	//	platform.AppendToOutput(output, fmt.Sprintf("Лог %v", time.Now().Format("15:04:05")))
	//})
	//w.SetContent(container.NewVBox(scroll, btn))

	setConfig := func() {
		cfg.GitLabLogin = loginEntry.Text
		cfg.GitLabPass = passEntry.Text
		cfg.NetboxToken = netboxEntry.Text
		//cfg.GitLabPass = hashString(passEntry.Text)
		//cfg.NetboxToken = hashString(netboxEntry.Text)

		switch {
		case asIsCheck.Checked:
			cfg.Mode = "as-is"
		case platformCheck.Checked:
			cfg.Mode = "platform"
		case regionCheck.Checked:
			cfg.Mode = "region"
		}

		days, _ := strconv.Atoi(scheduleEntry.Text)
		if days < 1 {
			days = 1
		} else if days > 31 {
			days = 31
		}

		timeValue := strings.TrimSpace(timeEntry.Text)
		if timeValue == "" {
			timeValue = "00:00"
			cfg.ScheduleTime = timeValue
		} else {
			cfg.ScheduleTime = strings.TrimSpace(timeEntry.Text)
		}

		cfg.ScheduleDays = days

		saveConfig(*cfg, configPath)

		if err := saveConfig(*cfg, configPath); err != nil {
			appendOutput(outputText, fmt.Sprintf("❌ Ошибка сохранения: %v", err))
		} else {
			loginEntry.SetText("")
			loginEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			passEntry.SetText("")
			passEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			netboxEntry.SetText("")
			netboxEntry.SetPlaceHolder("✅ Сохранено в файл настроек.")
			scheduleEntry.SetText("")
			scheduleEntry.SetPlaceHolder((fmt.Sprintf("%d", cfg.ScheduleDays) + "    ( ✅ Интервал дней сохранен в файл настроек.)"))
			timeEntry.SetText("")
			timeEntry.SetPlaceHolder(cfg.ScheduleTime + "    ( ✅ Время запуска сохранено в файл настроек.)")
			appendOutput(outputText, "✅ Настройки сохранены.")
		}

	}

	updateButtonState := func() {
		if _, err := os.Stat(configPath); err == nil {
			saveBtn.SetText("Сбросить")
		} else {
			saveBtn.SetText("Сохранить")
		}
	}
	updateButtonState()
	outputView := widget.NewLabelWithData(outputText)
	resetConfig := func() {
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

		scheduleEntry.SetText("")
		scheduleEntry.SetPlaceHolder("Интервал (дней, 1–31)")
		timeEntry.SetText("")
		timeEntry.SetPlaceHolder("Время обновления (HH:MM)")
		appendOutput(outputText, "\n⚙️ Настройки сброшены.\n")
		//outputView.SetText(outputView.Text + "\n⚙️ Настройки сброшены.\n")
		updateButtonState()

	}

	//действие кнопки "скачать/сохранить"
	saveBtn.OnTapped = func() {
		if !checkModeSelected(asIsCheck, platformCheck, regionCheck, w) {
			return
		}
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

		if saveBtn.Text == "Сбросить" {
			dialog.ShowConfirm(
				"Подтверждение",
				"Вы уверены, что хотите удалить все сохранённые данные?",
				func(confirmed bool) {
					if confirmed {
						resetConfig() //функция сброса
						saveBtn.SetText("Сохранить")
					}
				},
				w, // ← окно, к которому относится диалог
			)
			return
		} else {
			setConfig() //функция установки
			saveBtn.SetText("Сбросить")
		}
	}

	//content := container.NewVBox(
	//	passContainer,
	//	saveBtn,
	//)
	//
	//w.SetContent(content)
	//w.ShowAndRun()

	// Output через binding

	// Чекбоксы (создаём один раз)

	// Кнопка "Скачать" — запускает клонирование, затем (опционально) сортировку
	var cloneBtn *widget.Button
	cloneBtn = widget.NewButton("Скачать", func() {
		if !checkModeSelected(asIsCheck, platformCheck, regionCheck, w) {
			return
		}
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

		cloneBtn.Disable() // делаем неактивной пока работает
		_ = outputText.Set("⏳ Начинаю загрузку из GitLab...\n")

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
		//token := strings.TrimSpace(netboxEntry.Text)

		// Сформировать auth URL (если надо)
		authURL := repoURL
		if login != "" {
			authURL = fmt.Sprintf("https://%s:%s@%s", login, pass, strings.TrimPrefix(repoURL, "https://"))
		}

		go func() {
			// запускаем git clone
			cmd := exec.Command("git", "clone", "--depth", "1", authURL, targetDir)
			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

			if err := cmd.Start(); err != nil {
				current, _ := outputText.Get()
				//_ = outputText.Set(current + fmt.Sprintf("❌ Ошибка запуска git: %v\n", err))
				_ = outputText.Set(current + fmt.Sprintf("и тут ошибка", err))
				cloneBtn.Enable()
				return
			}

			// читаем потоки параллельно и собираем краткий лог
			lines := make([]string, 0)
			outDone := make(chan struct{})
			errDone := make(chan struct{})

			go func() {
				sc := bufio.NewScanner(stdout)
				for sc.Scan() {
					// не выводим прогресс; собираем только важные stdout строки
					txt := sc.Text()
					if strings.TrimSpace(txt) != "" {
						lines = append(lines, txt)
					}
				}
				close(outDone)
			}()

			go func() {
				sc := bufio.NewScanner(stderr)
				for sc.Scan() {
					// stderr содержит прогресс и ошибки — здесь можно фильтровать
					txt := sc.Text()
					// фильтруем строки с \r (прогресс) — сохраняем только ошибки
					if strings.Contains(txt, "already exists and is not an empty directory") {
						lines = append(lines, "⚠️ Папка назначения уже существует.")
					} else if strings.TrimSpace(strings.ReplaceAll(txt, "\r", "")) != "" {
						// собираем остальные осмысленные сообщения
						lines = append(lines, strings.ReplaceAll(txt, "\r", ""))
					}
				}
				close(errDone)
			}()

			// ждём чтения потоков
			<-outDone
			<-errDone

			// ждём завершения процесса
			_ = cmd.Wait()

			// Обновляем output один раз кратким результатом
			joined := strings.Join(lines, "\n")
			if joined == "" {
				joined = "Скачивание завершено (без сообщений).\n"
			}
			_ = appendOutput(outputText, fmt.Sprintf("тут ошибка ", err))
			//_ = outputText.Set(current + fmt.Sprintf("❌ Ошибка запуска git: %v\n", err))
			_ = appendOutput(outputText, "\n✅ Репозиторий загружен.\n")
			//_ = outputText.Set(outputString(outputText) + joined + "\n✅ Репозиторий загружен.\n")

			// Если выбран режим Платформа — запускаем сортировку
			if platformCheck.Checked {
				_ = appendOutput(outputText, "🚀 Запуск сортировки по платформам...\n")

				// SortFilesByPlatform должен принимать binding.String, см. файл sort_by_platform.go
				if err := platform.SortFilesByPlatform(targetDir, sortedDst, token, outputText, scroll); err != nil {
					_ = appendOutput(outputText, fmt.Sprintf("❌ Ошибка сортировки: %v\n", err))
					//_ = outputText.Set(outputString(outputText) + fmt.Sprintf("❌ Ошибка сортировки: %v\n", err))
				} else {
					_ = appendOutput(outputText, "\n✅ Завершено успешно.")

					//_ = outputText.Set(outputString(outputText) + "✅ Сортировка завершена.\n")
				}
			}

			if asIsCheck.Checked {

				//_ = appendOutput(outputText, "🚀 Запуск клонирования в режиме 'Как есть'...\n")

				//asis.MoveAsIs(targetDir, sortedDst, output)

				if err := asis.MoveAsIs(targetDir, sortedDst, output); err != nil {
					_ = appendOutput(outputText, fmt.Sprintf("❌ Ошибка копирования: %v\n", err))
				} else {
					_ = appendOutput(outputText, "\n✅ Завершено успешно.")
				}
			}

			//if asIsCheck.Checked {
			//	_ = appendOutput(outputText, "🚀 Скачиание без сортировки...\n")
			//
			//	err = asis.CopyAsIs(targetDir, sortedDst, output, scroll)
			//
			//} else if platformCheck.Checked {
			//	err = platform.SortFilesByPlatform(targetDir, sortedDst, netboxEntry.Text, output, scroll)
			//
			//} else if regionCheck.Checked {
			//	err = region.SortFilesByRegion(targetDir, sortedDst, output, scroll)
			//	}
			//}

			// всё готово — разблокируем кнопку (в главном потоке это безопасно сделать через binding Set)
			cloneBtn.Enable()
		}()
	})

	cloneBtn.Resize(fyne.NewSize(140, 40))
	cloneButtonContainer := container.NewHBox(layout.NewSpacer(), cloneBtn, layout.NewSpacer())

	// Сборка формы
	form := container.NewVBox(
		loginEntry,
		passEntry,
		netboxEntry,
		//passContainer,
		container.NewHBox(asIsCheck, layout.NewSpacer(), platformCheck, layout.NewSpacer(), regionCheck),
		cloneButtonContainer,
		scroll,
		saveBtn,
		scheduleEntry,
		timeEntry,
		//passContainer,
		//maskedPass
	)

	w.SetContent(form)
	w.ShowAndRun()
}

func checkModeSelected(asIsCheck, platformCheck, regionCheck *widget.Check, w fyne.Window) bool {
	if !asIsCheck.Checked && !platformCheck.Checked && !regionCheck.Checked {
		dialog.ShowInformation("Ой!", "⚠️ Выберите режим сохранения.", w)
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

//func ValidateCredentials(login, pass, token string) error {
//	if strings.TrimSpace(login) == "" {
//		return fmt.Errorf("Поле логина не заполнено.")
//	}
//	if strings.TrimSpace(pass) == "" {
//		return fmt.Errorf("Пароль GitLab не заполнен.")
//	}
//	if strings.TrimSpace(token) == "" {
//		return fmt.Errorf("Токен NetBox не заполнен.")
//	}
//	return nil
//}

// helper: получить текущий текст из binding.String (без ошибки)
//func outputString(b binding.String) string {
//	s, _ := b.Get()
//	return s
//}
