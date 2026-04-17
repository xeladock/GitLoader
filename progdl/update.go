// progdl/update.go
package progdl

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"configtool.local/acl"
	"configtool.local/asis"
	config "configtool.local/conf"
	"configtool.local/dwl_parser"
	"configtool.local/platform"
	"configtool.local/region"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

func RunUpdateMode(
	targetDir, sortedDst string,
	cfg *config.AppConfig,
	asIs, platformMode, regionMode, dcCheck, lanCheck, aclMode, parserCheck bool,
	netboxToken string,
	output binding.String,
	scroll *container.Scroll,
	manualRun bool,
) error {

	if platformMode {
		if err := platform.SortFilesByPlatform(targetDir, sortedDst, netboxToken, output, scroll, dcCheck, lanCheck); err != nil {
			return err
		}
	}

	if asIs {
		//println(targetDir, "--targetdir в update.go", sortedDst, "--sortedDst в update.go")
		if err := asis.MoveAsIs(targetDir, sortedDst, output, dcCheck, lanCheck); err != nil {
			return err
		}
		// ← В режиме «Как есть» — НЕ копируем ничего дополнительно!
		//Append(output, "Режим «Как есть» завершён — файлы уже в нужной папке.\n")
	}

	if aclMode {
		if err := acl.SortFilesByACL(targetDir, sortedDst, netboxToken, output, scroll, dcCheck, lanCheck); err != nil {
			return err
		}

	}

	if regionMode {
		//println(targetDir, "--targetdir в update.go", sortedDst, "--sortedDst в update.go")
		if err := region.SortByRegion(targetDir, sortedDst, output, scroll, dcCheck, lanCheck); err != nil {
			return err
		}
	}

	//if parserCheck {
	//	appendOutput(output, scroll, "\rСкачиваем ACL-Parser...\n")
	//
	//	if err := dwl_parser.DownloadACLParser(sortedDst); err != nil { // targetDir — папка, куда идёт сохранение
	//		appendOutput(output, scroll, "❌ "+err.Error()+"\n")
	//	} else {
	//		appendOutput(output, scroll, "✅ ACL-Parser успешно скачан\n")
	//	}
	//}

	if parserCheck {
		targetDir := filepath.Join(sortedDst, "..", "..")
		//println("targetDir is ", targetDir)
		dwl_parser.DownloadACLParser(targetDir, output, scroll) // или baseDir, если нужно на уровень выше
	}

	// Только для режимов сортировки (не «Как есть») — копируем в config_files_clear
	//currentDir := filepath.Join(sortedDst, "config_files_clear")
	currentDir := sortedDst

	if err := copyDir(sortedDst, currentDir); err != nil {
		return err
	}
	time.Sleep(400 * time.Millisecond)
	if manualRun == false {
		nextTime := calculateNextRun(cfg.ScheduleDays, cfg.ScheduleTime, cfg.LastRun)
		delay := time.Until(nextTime)

		totalMinutes := int(delay.Minutes())

		days := totalMinutes / (24 * 60)
		remaining := totalMinutes % (24 * 60)
		hours := remaining / 60
		minutes := remaining % 60

		var parts []string

		if days > 0 {
			parts = append(parts, fmt.Sprintf("%d д.", days))
		}
		if hours > 0 || days > 0 { // показываем часы, если есть дни
			parts = append(parts, fmt.Sprintf("%d ч.", hours))
		}
		if minutes > 0 || len(parts) == 0 { // минуты всегда, если нет ни дней, ни часов
			parts = append(parts, fmt.Sprintf("%d мин.", minutes))
		}

		timeStr := strings.Join(parts, " ")

		appendOutput(output, scroll, fmt.Sprintf(
			"🔄 Следующий запуск: %s в %s. (через %s)\n",
			nextTime.Format("02.01.2006"),
			nextTime.Format("15:04"),
			timeStr,
		))
	}
	//println("manualrun in update is ", manualRun)
	//if manualRun == false {
	//	nextTime := calculateNextRun(cfg.ScheduleDays, cfg.ScheduleTime, cfg.LastRun)
	//	delay := time.Until(nextTime)
	//	hours := int(delay.Hours())
	//	minutes := int(delay.Minutes()) - hours*60
	//
	//	appendOutput(output, scroll, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
	//		nextTime.Format("02.01.2006"),
	//		nextTime.Format("15:04"),
	//		hours, minutes))
	//}

	//Append(output, "Текущие файлы успешно обновлены.\n")
	return nil
}
