// scheduler/schedule.go
package scheduler

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"configtool.local/state_var"
	//"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	//"fyne.io/fyne/v2/widget"
	//"fyne.io/fyne/v2/widget"
)

type UpdateHintFunc func()

var (
	NextScheduledRun time.Time // ← добавь эту строку
)

//type stoprun bool

// Start запускает планировщик
// cfgPath — путь к config.json (чтобы сохранить LastRun)
// scheduleDays, scheduleTime — из твоего config.json
// lastRun — текущая дата из config.json (может быть пустой)
// onUpdateLastRun — callback, который ты вызываешь после успешного скачивания
func Start(
	cfgPath string,
	scheduleDays int,
	scheduleTimeStr string,
	lastRun string,
	onUpdateLastRun func(string), // ← сохраняет LastRun в config.json
	cloneAction func(),
	output binding.String,
	cancel <-chan struct{},
	updateHint UpdateHintFunc,
	countRun int,
) {
	if scheduleDays <= 0 || scheduleDays > 31 || scheduleTimeStr == "" {
		appendLog(output, "❌ Ошибка: Нарушение интервала загрузки.\n")
		return
	}

	parts := strings.Split(scheduleTimeStr, ":")
	if len(parts) != 2 {
		appendLog(output, "❌ Ошибка: Неверный формат времени.\n")
		return
	}
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		appendLog(output, "❌ Ошибка: Некорректное время в настройках.\n")
		//cfg.SchedulerState = "paused"
		//_ = saveConfig(cfg, configPath)
		return
	}

	//targetTime := time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
	interval := 24 * time.Hour * time.Duration(scheduleDays)
	//println("time.hour is ", time.Hour)
	//println("duration is ", time.Duration(scheduleDays))
	//println("interval is ", interval)

	output.Set("")
	updateHint()
	appendLog(output, fmt.Sprintf("▶ Планировщик запущен.\n"))

	//time.AfterFunc(1*time.Second, func() {
	ScheduleNext(cfgPath, scheduleTimeStr, interval, lastRun, onUpdateLastRun, cloneAction, output, cancel, true, countRun) // ← true: печатаем сообщение сразу
	//})
	if !NextScheduledRun.IsZero() {
		fmt.Println("=== ПЛАНИРОВЩИК ЗАПУЩЕН ===")
		fmt.Printf("Следующий запуск: %s\n", NextScheduledRun.Format("02.01.2006 15:04:05"))
		fmt.Printf("До запуска: %v\n", time.Until(NextScheduledRun))
	}
}

func ScheduleNext(
	cfgPath string,
	scheduleTimeStr string,
	interval time.Duration,
	lastRun string,
	onUpdateLastRun func(string),
	cloneAction func(),
	output binding.String,
	cancel <-chan struct{},
	printNext bool,
	countRun int,
) {
	now := time.Now()
	loc := now.Location()

	parts := strings.Split(scheduleTimeStr, ":")
	if len(parts) != 2 {
		appendLog(output, "Ошибка: неверный формат времени.\n")
		return
	}

	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

	var nextRun time.Time

	if countRun == 0 {
		// Это первый запуск в истории программы
		today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)

		if now.Before(today) {
			nextRun = today
		} else {
			nextRun = today.Add(interval)
		}
	} else {
		// Обычный повторный запуск
		if lastRun != "" {
			last, err := time.Parse(time.RFC3339, lastRun)
			if err == nil && !last.IsZero() {
				nextRun = last.Add(interval)
				nextRun = time.Date(nextRun.Year(), nextRun.Month(), nextRun.Day(), hour, minute, 0, 0, loc)

				// Если время уже прошло — прибавляем интервалы
				for !nextRun.After(now) {
					nextRun = nextRun.Add(interval)
				}
			} else {
				// LastRun повреждён — fallback на первый запуск
				today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
				nextRun = today.Add(interval)
			}
		} else {
			// LastRun пустой — считаем как первый запуск
			today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
			nextRun = today.Add(interval)
		}
	}
	NextScheduledRun = nextRun
	delay := time.Until(nextRun)
	totalMinutes := int(delay.Minutes())

	if totalMinutes < 1 {
		appendLog(output, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (До запуска меньше минуты)\n",
			nextRun.Format("02.01.2006"),
			nextRun.Format("15:04"),
		))
	} else if printNext {
		days := totalMinutes / (24 * 60)
		remaining := totalMinutes % (24 * 60)
		hours := remaining / 60
		minutes := remaining % 60

		var parts []string
		if days > 0 {
			parts = append(parts, fmt.Sprintf("%d д.", days))
		}
		if hours > 0 {
			parts = append(parts, fmt.Sprintf("%d ч.", hours))
		}
		if minutes > 0 || len(parts) == 0 {
			parts = append(parts, fmt.Sprintf("%d мин.", minutes))
		}

		timeStr := strings.Join(parts, " ")
		appendLog(output, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %s)\n",
			nextRun.Format("02.01.2006"),
			nextRun.Format("15:04"),
			timeStr,
		))
		//var NextRunTime time.Time
		//NextRunTime = nextRun
		//println(nextRun)
	}

	time.AfterFunc(delay, func() {
		select {
		case <-cancel:
			return
		default:
		}

		if state_var.ManRun.Load() {
			appendLog(output, "⚠️ Ручная загрузка активна. Запуск по расписанию отменён.\n")
		} else {
			NotifySuccess("Внимание!", "Запуск загрузки по расписанию!")
			go cloneAction()
		}

		newLastRun := time.Now().Format(time.RFC3339)
		onUpdateLastRun(newLastRun)

		ScheduleNext(cfgPath, scheduleTimeStr, interval, newLastRun, onUpdateLastRun, cloneAction, output, cancel, false, countRun)
	})
}

func NotifySuccess(title, message string) {
	//exec.Command("paplay", "./icon/yes.mp3").Run()
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "6000", title, message).Run()
	//cmd.Run() // ошибки молча игнорируем
}

func appendLog(output binding.String, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text)
}
