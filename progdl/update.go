// progdl/update.go
package progdl

import (
	"configtool.local/asis"
	"configtool.local/platform"
	"configtool.local/region"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

// RunUpdateMode — текущий режим: просто обновляет ./config_files_clear
func RunUpdateMode(
	targetDir, sortedDst string,
	asIs, platformMode, regionMode bool,
	netboxToken string,
	output binding.String,
	scroll *container.Scroll,
) error {

	_ = RemoveGitFolder(targetDir, output)

	// === Сортировка (одинаковая для обоих режимов) ===
	if platformMode {
		if err := platform.SortFilesByPlatform(targetDir, sortedDst, netboxToken, output, scroll); err != nil {
			return err
		}
	}
	if asIs {
		if err := asis.MoveAsIs(targetDir, sortedDst, output); err != nil {
			return err
		}
	}
	if regionMode {
		if err := region.SortByRegion(targetDir, sortedDst, output); err != nil {
			return err
		}
	}

	// === Перезапись в актуальную папку ===
	currentDir := "./config_files_clear"
	//_ = appendOutput(output, "Обновление текущих файлов...\n")

	// Просто перезаписываем содержимое — без удаления папки целиком
	if err := copyDir(sortedDst, currentDir); err != nil {
		return err
	}

	//_ = appendOutput(output, "Текущие файлы успешно обновлены.\n")
	return nil
}
