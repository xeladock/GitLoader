package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"configtool.local/nb"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

func SortFilesByPlatform(
	srcDir, dstBase, netboxToken string,
	output binding.String,
	scroll *container.Scroll,
	dcbool, lanbool bool,
) error {
	//println("sort by platform", dcbool, lanbool)
	typeDirs := []string{}
	//println(dstBase, "- dstbase", srcDir, "- srcdir")
	if dcbool {
		os.RemoveAll(dstBase)
		time.Sleep(500 * time.Millisecond)
		os.MkdirAll(dstBase, 0755)
		//os.RemoveAll(filepath.Join(dstBase, "ЦОД"))
		typeDirs = append(typeDirs, "ЦОД")
		//dcbool = false

	}
	if lanbool {
		//println(dstBase, "очистка lan для платформы")
		os.RemoveAll(dstBase)
		time.Sleep(500 * time.Millisecond)
		os.MkdirAll(dstBase, 0755)
		//os.RemoveAll(filepath.Join(dstBase, "ЛВС"))
		typeDirs = append(typeDirs, "ЛВС")
		//lanbool = false
	}

	allowedRoots := []string{"PR", "DV", "SZ", "CE", "UR", "UK", "SI"}
	//allowedRoots := []string{"DV"}

	for _, t := range typeDirs {
		//println(t, srcDir)
		AppendToOutput(output, scroll, fmt.Sprintf("\n⏳ Начинаю сортировку для УЭС%s \n", t))
		//typeSrc := filepath.Join(srcDir)
		//println(typeSrc, "- typesrc")
		if !dirExists(srcDir) {
			continue
		}

		err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			fileName := info.Name()
			//upper := strings.ToUpper(fileName)

			// фильтр по корням
			valid := false
			for _, root := range allowedRoots {
				if strings.HasPrefix(fileName, root) {
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
			platform = strings.ReplaceAll(platform, "/", "_")
			//println(dstBase, "- dstbase", platform, "-platform")
			targetDir := filepath.Join(dstBase, platform)
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
	//println(srcDir, "- srcdir удаление")
	os.RemoveAll(filepath.Dir(srcDir))
	AppendToOutput(output, scroll, "\n")
	return nil
}

// appendToOutput безопасно добавляет строку в binding.String (для GUI)
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

var t = 0

func AppendToOutput(output binding.String, scroll *container.Scroll, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text + "\n")
	t++
	if t > 40 {
		_ = output.Set("🔥 Продолжаем сортировку...\n")
		t = 0
	}
	fyne.Do(func() {
		scroll.ScrollToBottom()
		//scroll.Refresh()
	})
}

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

//func AppendToOutput2(output binding.String, scroll *container.Scroll, msg string) {
//	current, _ := output.Get()
//	output.Set(current + msg + "\n")
//
//	fyne.Do(func() {
//		scroll.ScrollToBottom()
//		scroll.Refresh()
//	})
//}

// безопасная автопрокрутка
//go func() {
//	time.Sleep(80 * time.Millisecond)
//	if scroll != nil {
//		scroll.ScrollToBottom()
//	}
//}()

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
