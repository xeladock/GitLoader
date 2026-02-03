// progdl/progress.go
package progdl

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"configtool.local/asis"
	config "configtool.local/conf"
	"configtool.local/platform"
	"configtool.local/region"
	//"configtool.local/main"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

const configPath = "config.json"

func calculateNextRun(days int, timeStr string, lastRunStr string) time.Time {
	now := time.Now()

	if timeStr == "" || !strings.Contains(timeStr, ":") {
		// Если время не задано — возвращаем завтра 00:00
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}

	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}

	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}

	loc := now.Location()

	// Парсим время HH:MM
	parts = strings.Split(timeStr, ":")
	hour, _ = strconv.Atoi(parts[0])
	minute, _ = strconv.Atoi(parts[1])

	interval := 24 * time.Hour * time.Duration(days)

	// Базовая точка — последний запуск или "давно"
	var base time.Time
	if lastRunStr != "" {
		last, err := time.Parse(time.RFC3339, lastRunStr)
		if err != nil || last.IsZero() {
			last = now.Add(-interval * 2)
		}
		base = last
	} else {
		base = now.Add(-interval)
	}

	// Следующий запуск после base
	next := base.Add(interval)

	// Приводим к нужному времени дня
	next = time.Date(next.Year(), next.Month(), next.Day(), hour, minute, 0, 0, loc)

	// Если уже прошло сегодня — переносим на завтра
	todayTarget := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
	if !now.Before(todayTarget) {
		next = todayTarget.Add(interval)
	}

	return next
}

// RunProgressMode — новый режим: сохраняет копию с датой
func RunProgressMode(
	targetDir, path string,
	cfg *config.AppConfig,
	asIs, platformMode, regionMode, dcCheck, lanCheck bool,
	netboxToken string,
	output binding.String,
	scroll *container.Scroll,
	manualRun bool,
) error {
	// ← ПАПКА С ДАТОЙ — сразу пишем туда!
	//parent := filepath.Dir(targetDir)
	dateStr := time.Now().Format("02-01-06")
	folderName := filepath.Base(targetDir)
	datedDir := filepath.Join(path, dateStr, "config_files_clear", folderName)
	//println(targetDir, "target в progress.go")
	//println(datedDir, "datedDir в progress.go")
	if err := os.MkdirAll(datedDir, 0777); err != nil {
		return err
	}
	//_ = RemoveGitFolder(targetDir, output)

	// === Та же сортировка ===
	if platformMode {
		if err := platform.SortFilesByPlatform(targetDir, datedDir, netboxToken, output, scroll, dcCheck, lanCheck); err != nil {
			return err
		}
	}
	if asIs {
		if err := asis.MoveAsIs(targetDir, datedDir, output, dcCheck, lanCheck); err != nil {
			return err
		}
	}
	if regionMode {
		if err := region.SortByRegion(targetDir, datedDir, output, scroll, dcCheck, lanCheck); err != nil {
			return err
		}
	}

	_ = appendOutput(
		output,
		fmt.Sprintf("\n📣Файлы %s сохранены в папку: %s\n", folderName, dateStr),
	)
	time.Sleep(500 * time.Millisecond)

	// === Сохранение в папку с датой ===
	//dateStr := time.Now().Format("02-01-06") // 27-11-25
	//datedDir := filepath.Join(".", dateStr, "config_files_clear")
	//switch {
	//case dcCheck && lanCheck:
	//	_ = appendOutput(output,
	//		"Файлы ЛВС и ЦОД сохранены в папку: "+dateStr)
	//case dcCheck:
	//	_ = appendOutput(output,
	//		"Файлы ЦОД сохранены в папку: "+dateStr)
	//case lanCheck:
	//	_ = appendOutput(output,
	//		"Файлы ЛВС сохранены в папку: "+dateStr)
	//}

	//_ = appendOutput(output, "Сохранение архивной копии («прогресс»)...\n")
	//if dcCheck {
	//_ = appendOutput(output, "Файлы ЦОД сохранены в папку: "+dateStr+"")
	//}
	//if lanCheck {
	//	_ = appendOutput(output, "Файлы ЛВС сохранены в папку: "+dateStr+"")
	//}

	//if err := os.MkdirAll(datedDir, 0755); err != nil {
	//	return err
	//}
	//
	//if err := copyDir(sortedDst, datedDir); err != nil {
	//	return err
	//}

	//_ = appendOutput(output, "Архивная копия сохранена успешно.")
	if manualRun == false {
		nextTime := calculateNextRun(cfg.ScheduleDays, cfg.ScheduleTime, cfg.LastRun)
		delay := time.Until(nextTime)
		hours := int(delay.Hours())
		minutes := int(delay.Minutes()) - hours*60

		appendOutput(output, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
			nextTime.Format("02.01.2006"),
			nextTime.Format("15:04"),
			hours, minutes))
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // ошибка или файла нет
	}
	return !info.IsDir() // существует и это файл
}

//scheduler.ScheduleNext(cfgPath, targetTime, interval, newLastRun, onUpdateLastRun, cloneAction, output, cancel, false) // ← true: печатаем в следующем цикле

//delay := time.Until(nextRun)
//hours := int(delay.Hours())
//minutes := int(delay.Minutes()) % 60
//appendLog(output, fmt.Sprintf("Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
//	nextRun.Format("02.01.2006"),
//	nextRun.Format("15:04"),
//	hours, minutes))

//func RemoveGitFolder(dir string, output binding.String) error {
//	gitPath := filepath.Join(dir, ".git")
//
//	// Проверяем, существует ли .git
//	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
//		//appendOutput(output, "Папка .git не найдена (уже удалена или clone прошёл без неё).\n")
//		return nil
//	}
//
//	err := os.Chmod(gitPath, 0777)
//	if err != nil {
//		return err
//	}
//
//	// Удаляем полностью
//	if err := os.RemoveAll(gitPath); err != nil {
//		appendOutput(output, fmt.Sprintf("Ошибка удаления .git: %v\n", err))
//		return err
//	}
//
//	//appendOutput(output, "Папка .git удалена.\n")
//	return nil
//}

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

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
