// progdl/progress.go
package progdl

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"configtool.local/asis"
	"configtool.local/platform"
	"configtool.local/region"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

// RunProgressMode — новый режим: сохраняет копию с датой
func RunProgressMode(
	targetDir, sortedDst string,
	asIs, platformMode, regionMode bool,
	netboxToken string,
	output binding.String,
	scroll *container.Scroll,
) error {

	_ = RemoveGitFolder(targetDir, output)

	// === Та же сортировка ===
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

	// === Сохранение в папку с датой ===
	dateStr := time.Now().Format("02-01-06") // 27-11-25
	datedDir := filepath.Join(".", dateStr, "config_files_clear")

	_ = appendOutput(output, "Сохранение архивной копии («прогресс»)...\n")
	_ = appendOutput(output, "Папка: "+dateStr+"\n")

	if err := os.MkdirAll(datedDir, 0755); err != nil {
		return err
	}

	if err := copyDir(sortedDst, datedDir); err != nil {
		return err
	}

	_ = appendOutput(output, "Архивная копия сохранена успешно.\n")
	return nil
}

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
