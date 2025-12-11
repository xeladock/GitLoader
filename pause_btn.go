// pause_btn.go
package main

import (
	//"strconv"
	//"strings"
	//"time"

	config "configtool.local/conf"
	"configtool.local/scheduler"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

var (
	//btn              *widget.Button
	//cfg              *config.AppConfig
	schedulerRunning bool
	schedulerCancel  chan struct{}
)

//func calculateNextRun(days int, timeStr string, lastRunStr string) time.Time {
//	now := time.Now()
//	loc := now.Location()
//
//	// Парсим время HH:MM
//	parts := strings.Split(timeStr, ":")
//	hour, _ := strconv.Atoi(parts[0])
//	minute, _ := strconv.Atoi(parts[1])
//
//	interval := 24 * time.Hour * time.Duration(days)
//
//	// Базовая точка — последний запуск или "давно"
//	var base time.Time
//	if lastRunStr != "" {
//		last, err := time.Parse(time.RFC3339, lastRunStr)
//		if err != nil || last.IsZero() {
//			last = now.Add(-interval * 2)
//		}
//		base = last
//	} else {
//		base = now.Add(-interval)
//	}
//
//	// Следующий запуск после base
//	next := base.Add(interval)
//
//	// Приводим к нужному времени дня
//	next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, loc)
//
//	// Если уже прошло сегодня — переносим на завтра
//	todayTarget := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
//	if !now.Before(todayTarget) {
//		next = todayTarget.Add(interval)
//	}
//
//	return next
//}

// Глобальная функция для цвета и текста
func updateButtonAppearance(btn *widget.Button, isRunning bool) {
	if isRunning {
		btn.Importance = widget.DangerImportance // красная "Пауза"
		btn.SetText("Пауза")
	} else {
		btn.Importance = widget.HighImportance // зелёная "Старт"
		btn.SetText("Старт")
	}
	btn.Refresh()
}

// Глобальная функция для блокировки/разблокировки (используется в gui4.go)
func PauseUpdateButtonState(btn *widget.Button, cfg *config.AppConfig, configPath string) {
	hasConfig := fileExists(configPath) && cfg.GitLabLogin != ""

	if !hasConfig {
		btn.Disable()
		btn.SetText("Старт")
		return
	} else {
		btn.Enable()
	}

	updateButtonAppearance(btn, cfg.SchedulerState == "running") // ← ПРАВИЛЬНО: true если "running", false если "paused"
}

func CreateStartPauseButton(
	cfg *config.AppConfig,
	configPath string,
	cloneAction func(),
	output binding.String,
	w fyne.Window,
) *widget.Button {

	btn := widget.NewButton("Старт", nil)

	// Восстановление состояния + уведомление
	if cfg.SchedulerState == "running" && cfg.ScheduleDays > 0 && cfg.ScheduleTime != "" {
		updateButtonAppearance(btn, true)
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
			cloneAction,
			output,
			schedulerCancel,
		)

	} else if cfg.SchedulerState == "paused" {
		updateButtonAppearance(btn, false)
		fyne.Do(func() {
			appendOutput(output, "ВНИМАНИЕ! Планировщик на ПАУЗЕ. Нажмите «Старт» для возобновления.\n")
		})
	} else {
		updateButtonAppearance(btn, false)
	}

	btn.OnTapped = func() {
		if schedulerRunning {
			stopScheduler(cfg, configPath, output, btn)
		} else {
			startScheduler(cfg, configPath, cloneAction, output, btn)
		}
	}

	return btn
}

func startScheduler(cfg *config.AppConfig, configPath string, cloneAction func(), output binding.String, btn *widget.Button) {
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
			//_ = saveConfig(cfg, configPath)
		},

		func() {
			fyne.Do(func() {
				isAutoRun = true // ← пропускаем диалог
				cloneAction()
				isAutoRun = false // ← ЭТО ЗАПУСКАЕТ СКАЧИВАНИЕ!
			})
		},
		//cloneAction,
		output,
		schedulerCancel,
	)

	fyne.Do(func() {
		//isAutoRun = true
		updateButtonAppearance(btn, true)
	})
	//appendOutput(output, "Планировщик запущен\n")
}

func stopScheduler(cfg *config.AppConfig, configPath string, output binding.String, btn *widget.Button) {
	if !schedulerRunning || schedulerCancel == nil {
		return
	}
	close(schedulerCancel)
	schedulerRunning = false
	cfg.SchedulerState = "paused"
	_ = saveConfig(cfg, configPath)

	fyne.Do(func() {
		updateButtonAppearance(btn, false)
	})
	appendOutput(output, "Планировщик на паузе.\n")
}
