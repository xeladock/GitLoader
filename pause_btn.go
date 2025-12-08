// pause_btn.go
package main

import (
	"configtool.local/scheduler"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

var (
	schedulerRunning bool
	schedulerCancel  chan struct{}
)

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
func PauseUpdateButtonState(btn *widget.Button, cfg *AppConfig, configPath string) {
	hasConfig := fileExists(configPath) && cfg.GitLabLogin != ""

	if !hasConfig {
		btn.Disable()
		btn.SetText("Старт")
		return
	}

	btn.Enable()
	updateButtonAppearance(btn, cfg.SchedulerState == "running") // ← ПРАВИЛЬНО: true если "running", false если "paused"
}

func CreateStartPauseButton(
	cfg *AppConfig,
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
		go startScheduler(cfg, configPath, cloneAction, output, btn)
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

func startScheduler(cfg *AppConfig, configPath string, cloneAction func(), output binding.String, btn *widget.Button) {
	if schedulerRunning {
		return
	}
	schedulerRunning = true
	cfg.SchedulerState = "running"
	_ = saveConfig(*cfg, configPath)

	schedulerCancel = make(chan struct{})
	go scheduler.Start(
		configPath,
		cfg.ScheduleDays,
		cfg.ScheduleTime,
		cfg.LastRun,
		func(newLastRun string) {
			cfg.LastRun = newLastRun
			_ = saveConfig(*cfg, configPath)
		},
		cloneAction,
		output,
		schedulerCancel,
	)

	fyne.Do(func() {
		updateButtonAppearance(btn, true)
	})
	//appendOutput(output, "Планировщик запущен\n")
}

func stopScheduler(cfg *AppConfig, configPath string, output binding.String, btn *widget.Button) {
	if !schedulerRunning || schedulerCancel == nil {
		return
	}
	close(schedulerCancel)
	schedulerRunning = false
	cfg.SchedulerState = "paused"
	_ = saveConfig(*cfg, configPath)

	fyne.Do(func() {
		updateButtonAppearance(btn, false)
	})
	appendOutput(output, "Планировщик на паузе.\n")
}
