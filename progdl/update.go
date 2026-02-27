// progdl/update.go
package progdl

import (
	"fmt"
	"time"

	"configtool.local/acl"
	"configtool.local/asis"
	config "configtool.local/conf"
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
		return nil
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

	// Только для режимов сортировки (не «Как есть») — копируем в config_files_clear
	//currentDir := filepath.Join(sortedDst, "config_files_clear")
	currentDir := sortedDst

	if err := copyDir(sortedDst, currentDir); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	if manualRun == false {
		nextTime := calculateNextRun(cfg.ScheduleDays, cfg.ScheduleTime, cfg.LastRun)
		delay := time.Until(nextTime)
		hours := int(delay.Hours())
		minutes := int(delay.Minutes()) - hours*60

		appendOutput(output, scroll, fmt.Sprintf("🔄 Следующий запуск: %s в %s. (через %d ч. %d мин.)\n",
			nextTime.Format("02.01.2006"),
			nextTime.Format("15:04"),
			hours, minutes))
	}

	//Append(output, "Текущие файлы успешно обновлены.\n")
	return nil
}

// RunUpdateMode — текущий режим: просто обновляет ./config_files_clear
//func RunUpdateMode(
//	targetDir, sortedDst string,
//	asIs, platformMode, regionMode, dcCheck, lanCheck bool,
//	netboxToken string,
//	output binding.String,
//	scroll *container.Scroll,
//) error {
//	//wantDC := dcCheck.Checked
//	//wantLAN := lanCheck.Checked
//
//	//_ = RemoveGitFolder(targetDir, output)
//
//	// === Сортировка (одинаковая для обоих режимов) ===
//	if platformMode {
//		if err := platform.SortFilesByPlatform(targetDir, sortedDst, netboxToken, output, scroll, dcCheck, lanCheck); err != nil {
//			return err
//		}
//	}
//	if asIs {
//		if err := asis.MoveAsIs(targetDir, sortedDst, output, dcCheck, lanCheck); err != nil {
//			return err
//		}
//	}
//	if regionMode {
//		if err := region.SortByRegion(targetDir, sortedDst, output, scroll, dcCheck, lanCheck); err != nil {
//			return err
//		}
//	}
//
//	// === Перезапись в актуальную папку ===
//	currentDir := "config_files_clear"
//
//	//startDir := "./configs"
//	hasDC := filepath.Join(targetDir, "ЦОД")
//	hasLAN := filepath.Join(targetDir, "ЛВС")
//	println(hasDC, hasLAN)
//
//	if dirExists(hasDC) {
//		currentDir = hasDC
//	}
//
//	if dirExists(hasLAN) {
//		currentDir = hasLAN
//	}
//	//_ = appendOutput(output, "Обновление текущих файлов...\n")
//
//	// Просто перезаписываем содержимое — без удаления папки целиком
//	if err := copyDir(sortedDst, currentDir); err != nil {
//		return err
//	}
//
//	//_ = appendOutput(output, "Текущие файлы успешно обновлены.\n")
//	return nil
//}

//func dirExists(path string) bool {
//	info, err := os.Stat(path)
//	if err != nil {
//		return false
//	}
//	return info.IsDir()
//}
