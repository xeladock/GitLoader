// scheduler/schedule.go
package scheduler

import (
	"fmt"
	"math"
	//"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

// Start запускает планировщик
// cfgPath — путь к config.json (чтобы сохранить LastRun)
// scheduleDays, scheduleTime — из твоего config.json
// lastRun — текущая дата из config.json (может быть пустой)
// onUpdateLastRun — callback, который ты вызываешь после успешного скачивания
func Start(
	cfgPath string,
	scheduleDays int,
	scheduleTime string,
	lastRun string,
	onUpdateLastRun func(string), // ← сохраняет LastRun в config.json
	cloneAction func(),
	output binding.String,
) {
	if scheduleDays <= 0 || scheduleTime == "" {
		return
	}

	parts := strings.Split(scheduleTime, ":")
	if len(parts) != 2 {
		appendLog(output, "Ошибка: неверный формат времени в config.json\n")
		return
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		appendLog(output, "Ошибка: некорректное время в config.json\n")
		return
	}

	targetTime := time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
	interval := 24 * time.Hour * time.Duration(scheduleDays)

	appendLog(output, fmt.Sprintf("Планировщик запущен.\n"))

	time.AfterFunc(3*time.Second, func() {
		scheduleNext(cfgPath, targetTime, interval, lastRun, onUpdateLastRun, cloneAction, output)
	})
}

func scheduleNext(
	cfgPath string,
	targetTime, interval time.Duration,
	lastRun string,
	onUpdateLastRun func(string),
	cloneAction func(),
	output binding.String,
) {
	now := time.Now()
	loc := now.Location()
	hour := int(targetTime / time.Hour)
	minute := int((targetTime % time.Hour) / time.Minute)

	var nextRun time.Time

	if lastRun != "" {
		last, _ := time.Parse(time.RFC3339, lastRun)
		if last.IsZero() {
			last = now.Add(-365 * 24 * time.Hour)
		}
		nextRun = last.Add(interval)
		nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)
		for !nextRun.After(now) {
			nextRun = nextRun.Add(interval)
		}
	} else {
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if now.After(today) || now.Equal(today) {
			nextRun = today.Add(interval)
		} else {
			nextRun = today
		}
	}

	delay := time.Until(nextRun)
	appendLog(output, fmt.Sprintf("Следующий запуск: %s в %s. (через %.0f ч. %.0f мин.)\n",
		nextRun.Format("02.01.2006"),
		nextRun.Format("15:04"),
		delay.Hours(), math.Mod(delay.Minutes(), 60)))

	time.AfterFunc(delay, func() {
		fyne.Do(func() {
			appendLog(output, fmt.Sprintf("Скачивание по расписанию: %s\n", time.Now().Format("15:04 02.01.2006")))
			cloneAction()
		})

		// Говорим главному коду: "обнови LastRun в config.json"
		newLastRun := time.Now().Format(time.RFC3339)
		onUpdateLastRun(newLastRun)

		// Следующий запуск
		scheduleNext(cfgPath, targetTime, interval, newLastRun, onUpdateLastRun, cloneAction, output)
	})
}

func appendLog(output binding.String, text string) {
	if output == nil {
		return
	}
	current, _ := output.Get()
	_ = output.Set(current + text)
}
