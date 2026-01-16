package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"configtool.local/nb"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

func SortFilesByPlatform(
	srcDir, dstBase, netboxToken string,
	output binding.String,
	scroll *container.Scroll,
	dcCheck, lanCheck bool,
) error {

	typeDirs := []string{}

	if dcCheck {
		os.RemoveAll(filepath.Join(dstBase, "ЦОД"))
		typeDirs = append(typeDirs, "ЦОД")
	}
	if lanCheck {
		os.RemoveAll(filepath.Join(dstBase, "ЛВС"))
		typeDirs = append(typeDirs, "ЛВС")
	}

	allowedRoots := []string{"PR", "pr", "DV", "dv", "SZ", "sz", "CE", "ce", "UR", "ur", "UK", "uk", "SI", "si"}

	for _, t := range typeDirs {
		AppendToOutput(output, scroll, fmt.Sprintf("Начинаю сортировку для папки **%s** \n", t))
		typeSrc := filepath.Join(srcDir, t)
		if !dirExists(typeSrc) {
			continue
		}

		err := filepath.Walk(typeSrc, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			fileName := info.Name()
			upper := strings.ToUpper(fileName)

			// фильтр по корням
			valid := false
			for _, root := range allowedRoots {
				if strings.HasPrefix(upper, root) {
					valid = true
					break
				}
			}
			if !valid {
				return nil
			}

			deviceName := strings.Split(fileName, ".")[0]
			platform, err := nb.GetDevicePlatform(deviceName, netboxToken)
			if err != nil || platform == "" {
				platform = "Unknown"
			}

			targetDir := filepath.Join(dstBase, t, platform)
			if err := os.MkdirAll(targetDir, 0777); err != nil {
				return err
			}

			targetFile := filepath.Join(targetDir, fileName)
			if err := copyFile(path, targetFile); err != nil {
				return err
			}

			AppendToOutput(
				output,
				scroll,
				fmt.Sprintf("[%s/%s] ← %s", t, platform, fileName),
			)

			return nil
		})

		if err != nil {
			return err
		}
	}

	os.RemoveAll(srcDir)
	return nil
}

// SortFilesByPlatform рекурсивно обходит srcDir,
// определяет платформу каждого файла и копирует в dstDir/<Платформа>/,
// при этом обновляет bindableText (например, для поля вывода в GUI).
//func SortFilesByPlatform(srcDir, dstDir, netboxToken string, output binding.String, scroll *container.Scroll, dcCheck, lanCheck bool) error {
//	allowedPrefixes := []string{
//		"PRNG-DC", "DVPR-DC", "SZSP-DC", "CEMO-DC",
//		"CEMS-DC", "UREK-DC", "UKFR-DC", "SINO-DC",
//	}
//	//очистить папку config_files_clear
//	if err := ClearDestination(dstDir); err != nil {
//		return err
//	}
//
//	//очистить папку configs в конце процесса
//	success := false
//	defer func() {
//		if success {
//			RemoveSourceAfterSuccess(srcDir, output)
//		}
//	}()
//
//	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			return err
//		}
//
//		if info.IsDir() {
//			return nil
//		}
//
//		fileName := info.Name()
//
//		valid := false
//		for _, prefix := range allowedPrefixes {
//			if strings.HasPrefix(strings.ToUpper(fileName), prefix) {
//				valid = true
//				break
//			}
//		}
//		if !valid {
//			return nil
//		}
//
//		deviceName := strings.Split(fileName, ".")[0]
//		platform, err := nb.GetDevicePlatform(deviceName, netboxToken)
//		if err != nil {
//			AppendToOutput(output, scroll, "[ERROR] "+deviceName+": "+err.Error())
//			return nil
//		}
//		if platform == "" {
//			platform = "Unknown"
//
//		}
//
//		targetDir := filepath.Join(dstDir, platform)
//		os.MkdirAll(targetDir, 0777)
//
//		targetFile := filepath.Join(targetDir, fileName)
//		if err := copyFile(path, targetFile); err != nil {
//			AppendToOutput(output, scroll, "[ERROR] copy "+fileName+": "+err.Error())
//			return nil
//		}
//
//		AppendToOutput(output, scroll, fmt.Sprintf("[%s] >> %s", platform, fileName))
//		time.Sleep(20 * time.Millisecond)
//		return nil
//	})
//
//	return err
//}

// appendToOutput безопасно добавляет строку в binding.String (для GUI)
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func scr(*container.Scroll) {
	scroll := container.Scroll{}
	scroll.ScrollToBottom()
}

var t = 0

func AppendToOutput(output binding.String, scroll *container.Scroll, text string) {

	current, _ := output.Get()

	_ = output.Set(current + text + "\n")
	t++

	if t > 50 {
		_ = output.Set("Продолжаем сортировку...\n")
		t = 0
	}
	fyne.Do(func() {
		scroll.ScrollToBottom()
		//scroll.Refresh()
	})

	//time.AfterFunc(20*time.Millisecond, func() {
	//	scroll.ScrollToBottom()
}

func AppendToOutput2(output binding.String, scroll *container.Scroll, msg string) {
	current, _ := output.Get()
	output.Set(current + msg + "\n")

	fyne.Do(func() {
		scroll.ScrollToBottom()
		scroll.Refresh()
	})
}

// безопасная автопрокрутка
//go func() {
//	time.Sleep(80 * time.Millisecond)
//	if scroll != nil {
//		scroll.ScrollToBottom()
//	}
//}()

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

//func ClearDestination(dstDir string) error {
//	// Удаляем всё содержимое (или всю папку)
//	if err := os.RemoveAll(dstDir); err != nil {
//		return fmt.Errorf("не удалось очистить целевую папку: %w", err)
//	}
//
//	// Создаём чистую папку заново
//	if err := os.MkdirAll(dstDir, 0777); err != nil {
//		return fmt.Errorf("не удалось создать целевую папку: %w", err)
//	}
//	return nil
//}
//
//func RemoveSourceAfterSuccess(srcDir string, output binding.String) {
//	if err := os.RemoveAll(srcDir); err != nil {
//		AppendToOutput(output, nil, fmt.Sprintf("Предупреждение: не удалось удалить папку %s: %v\n", srcDir, err))
//		//} else {
//		//	AppendToOutput(output, nil, "Исходная папка configs удалена.\n")
//	}
//}

//
//_ = AppendToOutput(output, nil, "Очищена папка с предыдущими результатами.\n")
//return nil
