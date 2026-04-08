// pause_btn.go
package main

import (
	//"strconv"
	//"strings"
	//"time"

	config "configtool.local/conf"
	"configtool.local/scheduler"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

var (
	//btn              *widget.Button
	//cfg              *config.AppConfig
	schedulerRunning bool
	schedulerCancel  chan struct{}
)

//	func calculateNextRun(days int, timeStr string, lastRunStr string) time.Time {
//		now := time.Now()
//		loc := now.Location()
//
//		// Парсим время HH:MM
//		parts := strings.Split(timeStr, ":")
//		hour, _ := strconv.Atoi(parts[0])
//		minute, _ := strconv.Atoi(parts[1])
//
//		interval := 24 * time.Hour * time.Duration(days)
//
//		// Базовая точка — последний запуск или "давно"
//		var base time.Time
//		if lastRunStr != "" {
//			last, err := time.Parse(time.RFC3339, lastRunStr)
//			if err != nil || last.IsZero() {
//				last = now.Add(-interval * 2)
//			}
//			base = last
//		} else {
//			base = now.Add(-interval)
//		}
//
//		// Следующий запуск после base
//		next := base.Add(interval)
//
//		// Приводим к нужному времени дня
//		next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, loc)
//
//		// Если уже прошло сегодня — переносим на завтра
//		todayTarget := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//		if !now.Before(todayTarget) {
//			next = todayTarget.Add(interval)
//		}
//
//		return next
//	}
func updateButtonAppearance(btn *widget.Button, cfg *config.AppConfig) {
	println("updateButtonAppearance → SchedulerState =", cfg.SchedulerState)

	if cfg.SchedulerState == "running" {
		btn.Importance = widget.DangerImportance
		btn.SetText("Пауза")
	} else {
		btn.Importance = widget.HighImportance
		btn.SetText("Старт")
	}
	btn.Refresh()
}

func PauseUpdateButtonState(btn *widget.Button, cfg *config.AppConfig, configPath string) {
	hasConfig := fileExists(configPath) && cfg.GitLabLogin != ""

	if !hasConfig {
		btn.Disable()
		btn.SetText("Старт")
		return
	}

	btn.Enable()
	updateButtonAppearance(btn, cfg)
}

// Глобальная функция для цвета и текста
//func updateButtonAppearance(btn *widget.Button, isRunning bool, cfg *config.AppConfig, configPath string) {
//	//if firstrun {
//	//	firstrun = false
//	//	isRunning = false
//	//}
//
//	//if cfg.SchedulerState = "running"
//	println("3. updateButtonAppearance. cfg.SchedulerState is ", cfg.SchedulerState)
//	//if !cnf{}
//	if fileExists(configPath) && cfg.SchedulerState == "" {
//		println("4. Ставлю время в pause_btn если SchedulerState пустой")
//		cfg.SchedulerState = "paused"
//
//		//_ = saveConfig(cfg, configPath)
//	}
//	//println("isrunning", isRunning)
//	//println(cfg.SchedulerState)
//	if cfg.SchedulerState == "running" {
//		//println("на старте")
//		btn.Importance = widget.DangerImportance // красная "Пауза"
//		btn.SetText("Пауза")
//	} else if cfg.SchedulerState == "paused" {
//		//println("на паузе")
//		btn.Importance = widget.HighImportance // синяя "Старт"
//		btn.SetText("Старт")
//	}
//	//_ = saveConfig(cfg, configPath)
//	btn.Refresh()
//}

// Глобальная функция для блокировки/разблокировки (используется в gui4.go)
//func PauseUpdateButtonState(btn *widget.Button, cfg *config.AppConfig, configPath string) {
//	hasConfig := fileExists(configPath) && cfg.GitLabLogin != ""
//	println("2. PauseUpdateButtonState. hasConfig is ", hasConfig)
//	if !hasConfig {
//		btn.Disable()
//		btn.SetText("Старт")
//
//		return
//	} else {
//		btn.Enable()
//	}
//	updateButtonAppearance(btn, cfg.SchedulerState == "running", cfg, configPath)
//
//} // ← ПРАВИЛЬНО: true если "running", false если "paused"

func CreateStartPauseButton(
	cfg *config.AppConfig,
	configPath string,
	cloneAction func(),
	output binding.String,
	//firstrun bool,
	scroll *container.Scroll,
	w fyne.Window,
) *widget.Button {
	//firstrun = false
	btn := widget.NewButton("Старт", nil)
	//println("состояние в CreateStartPauseButton", cfg.SchedulerState)
	// Восстановление состояния + уведомление
	if cfg.SchedulerState == "running" && cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
		updateButtonAppearance(btn, cfg)
		schedulerRunning = true
		schedulerCancel = make(chan struct{})
		//isAutoRun = true
		go scheduler.Start(
			configPath,
			cfg.ScheduleDays,
			cfg.ScheduleTime,
			cfg.LastRun,
			func(newLastRun string) {
				cfg.LastRun = newLastRun
				_ = saveConfig(cfg, configPath)
			},
			func() {
				fyne.Do(func() {
					isAutoRun = true // ← обязательно ставим перед cloneAction
					cloneAction()
					isAutoRun = false // ← сбрасываем сразу после
				})
			},
			//cloneAction,
			output,
			schedulerCancel,
			updateHint,
		)

	} else if cfg.SchedulerState == "paused" {
		updateButtonAppearance(btn, cfg)
		fyne.Do(func() {
			appendOutput(output, "⚠️ ВНИМАНИЕ! Планировщик не активен! Нажмите «Старт» для запуска.\n")
		})
	} else {
		updateButtonAppearance(btn, cfg)
	}

	btn.OnTapped = func() {

		if schedulerRunning {
			stopScheduler(cfg, configPath, output, btn, scroll)

		} else {
			startScheduler(cfg, configPath, cloneAction, output, btn, scroll)
		}
		println(cfg.SchedulerState + " ==== в btn.OnTapped")
	}
	return btn
}

func startScheduler(cfg *config.AppConfig, configPath string, cloneAction func(), output binding.String, btn *widget.Button, scroll *container.Scroll) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	if schedulerRunning {
		return
	}
	schedulerRunning = true
	cfg.SchedulerState = "running"
	_ = saveConfig(cfg, configPath)

	schedulerCancel = make(chan struct{})
	go scheduler.Start(
		configPath,
		cfg.ScheduleDays,
		cfg.ScheduleTime,
		cfg.LastRun,
		func(newLastRun string) {
			cfg.LastRun = newLastRun
			_ = saveConfig(cfg, configPath)
		},

		func() {
			fyne.Do(func() {
				isAutoRun = true // ← пропускаем диалог
				cloneAction()
				isAutoRun = false // ← ЭТО ЗАПУСКАЕТ СКАЧИВАНИЕ!
			})
		},
		output,
		schedulerCancel,
		updateHint,
	)

	fyne.Do(func() {
		updateButtonAppearance(btn, cfg)
		scroll.ScrollToBottom()
	})

}

func stopScheduler(cfg *config.AppConfig, configPath string, output binding.String, btn *widget.Button, scroll *container.Scroll) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	if !schedulerRunning || schedulerCancel == nil {
		return
	}
	close(schedulerCancel)
	schedulerRunning = false
	cfg.SchedulerState = "paused"
	_ = saveConfig(cfg, configPath)
	outputText.Set("")
	updateHint()
	if err := saveConfig(cfg, configPath); err != nil {
		appendOutput(output, "❌ Ошибка сохранения состояния: "+err.Error()+"\n")
	}
	fyne.Do(func() {
		//time.Sleep(100 * time.Millisecond)
		updateButtonAppearance(btn, cfg)
		scrollToBottom(scroll)
		//scroll.Refresh()
	})

	appendOutput(output, "⏸ Планировщик на паузе.\n")
}

func scrollToBottom(scroll *container.Scroll) {
	// Принудительно ставим offset на максимум
	fyne.Do(func() {
		scroll.Offset = fyne.NewPos(0, scroll.Content.Size().Height+100) // +100 на всякий случай
		scroll.ScrollToBottom()
		scroll.Refresh()
	})
}
