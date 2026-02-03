// scheduler/schedule.go
package scheduler

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	//"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	//"fyne.io/fyne/v2/widget"
	//"fyne.io/fyne/v2/widget"
)

type UpdateHintFunc func()

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
	cancel <-chan struct{},
	updateHint UpdateHintFunc,
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
		appendLog(output, "Ошибка: некорректное время in config.json\n")
		return
	}

	targetTime := time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute
	interval := 24 * time.Hour * time.Duration(scheduleDays)

	output.Set("")
	updateHint()
	appendLog(output, fmt.Sprintf("▶ Планировщик запущен.\n"))

	//time.AfterFunc(1*time.Second, func() {
	ScheduleNext(cfgPath, targetTime, interval, lastRun, onUpdateLastRun, cloneAction, output, cancel, true) // ← true: печатаем сообщение сразу
	//})
}

//func clearLogKeepHeader() {
//	outputText := binding.NewString()
//	current, _ := outputText.Get()
//	lines := strings.Split(current, "\n")
//
//	// Оставляем только последние 2 строки (заголовок)
//	if len(lines) > 2 {
//		header := strings.Join(lines[len(lines)-3:], "\n") // -3 потому что последняя пустая
//		outputText.Set(header + "\n")
//	}
//}

func ScheduleNext(
	cfgPath string,
	targetTime, interval time.Duration,
	lastRun string,
	onUpdateLastRun func(string),
	cloneAction func(),
	output binding.String,
	cancel <-chan struct{},
	printNext bool, // ← параметр: печатаем ли "Следующий запуск..." в этом цикле
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

	// Если printNext == true — печатаем "Следующий запуск..." СРАЗУ
	if printNext {
		hours := int(delay.Hours())
		minutes := int(delay.Minutes()) % 60
		appendLog(output, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
			nextRun.Format("02.01.2006"),
			nextRun.Format("15:04"),
			hours, minutes))
	}

	time.AfterFunc(delay, func() {
		select {
		case <-cancel:
			//appendLog(output, "❌ Планировщик остановлен.\n")
			return
		default:
		}
		//fyne.Do(func() {
		//	appendLog(output,
		//		fmt.Sprintf("🚨 Выполняю загрузку по расписанию: %s в %s\n",
		//			time.Now().Format("02.01.2006"),
		//			time.Now().Format("15:04"),
		//		),
		//	)
		//})
		//time.Sleep(2 * time.Second)
		NotifySuccess("Внимание!", "Запуск загрузки по расписанию!")

		go func() {
			cloneAction()
		}()

		newLastRun := time.Now().Format(time.RFC3339)
		onUpdateLastRun(newLastRun)

		ScheduleNext(cfgPath, targetTime, interval, newLastRun, onUpdateLastRun, cloneAction, output, cancel, false) // ← true: печатаем в следующем цикле
	})
}

func NotifySuccess(title, message string) {
	//exec.Command("paplay", "./icon/yes.mp3").Run()
	exec.Command("notify-send", "-u", "normal", "-a", "GitTornado", "-t", "5000", title, message).Run()
	//cmd.Run() // ошибки молча игнорируем
}

func appendLog(output binding.String, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text)
}
