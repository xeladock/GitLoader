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

func calculateNextRun(days int, timeStr string, lastRunStr string) time.Time {
	now := time.Now()
	loc := now.Location()

	// Парсим время HH:MM
	parts := strings.Split(timeStr, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])

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
	targetDir, sortedDst string,
	cfg *config.AppConfig,
	asIs, platformMode, regionMode bool,
	netboxToken string,
	output binding.String,
	scroll *container.Scroll,
) error {
	// ← ПАПКА С ДАТОЙ — сразу пишем туда!
	dateStr := time.Now().Format("02-01-06")
	datedDir := filepath.Join(".", dateStr, "config_files_clear")
	if err := os.MkdirAll(datedDir, 0777); err != nil {
		return err
	}
	_ = RemoveGitFolder(targetDir, output)

	// === Та же сортировка ===
	if platformMode {
		if err := platform.SortFilesByPlatform(targetDir, datedDir, netboxToken, output, scroll); err != nil {
			return err
		}
	}
	if asIs {
		if err := asis.MoveAsIs(targetDir, datedDir, output); err != nil {
			return err
		}
	}
	if regionMode {
		if err := region.SortByRegion(targetDir, datedDir, output); err != nil {
			return err
		}
	}

	// === Сохранение в папку с датой ===
	//dateStr := time.Now().Format("02-01-06") // 27-11-25
	//datedDir := filepath.Join(".", dateStr, "config_files_clear")

	//_ = appendOutput(output, "Сохранение архивной копии («прогресс»)...\n")
	_ = appendOutput(output, "Сохранено в папку: "+dateStr+"\n")

	//if err := os.MkdirAll(datedDir, 0755); err != nil {
	//	return err
	//}
	//
	//if err := copyDir(sortedDst, datedDir); err != nil {
	//	return err
	//}

	//_ = appendOutput(output, "Архивная копия сохранена успешно.")

	nextTime := calculateNextRun(cfg.ScheduleDays, cfg.ScheduleTime, cfg.LastRun)
	delay := time.Until(nextTime)
	hours := int(delay.Hours())
	minutes := int(delay.Minutes()) - hours*60

	appendOutput(output, fmt.Sprintf("Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
		nextTime.Format("02.01.2006"),
		nextTime.Format("15:04"),
		hours, minutes))

	return nil
}

//scheduler.ScheduleNext(cfgPath, targetTime, interval, newLastRun, onUpdateLastRun, cloneAction, output, cancel, false) // ← true: печатаем в следующем цикле

//delay := time.Until(nextRun)
//hours := int(delay.Hours())
//minutes := int(delay.Minutes()) % 60
//appendLog(output, fmt.Sprintf("Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
//	nextRun.Format("02.01.2006"),
//	nextRun.Format("15:04"),
//	hours, minutes))

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

	//appendOutput(output, "Папка .git удалена.\n")
	return nil
}

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
