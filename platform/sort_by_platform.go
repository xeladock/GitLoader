package platform

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"configtool.local/nb"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

var (
	outputText binding.String
	scroll     *container.Scroll
)

// SortFilesByPlatform рекурсивно обходит srcDir,
// определяет платформу каждого файла и копирует в dstDir/<Платформа>/,
// при этом обновляет bindableText (например, для поля вывода в GUI).
func SortFilesByPlatform(srcDir, dstDir, netboxToken string, output binding.String, scroll *container.Scroll) error {
	allowedPrefixes := []string{
		"PRNG-DC", "DVPR-DC", "SZSP-DC", "CEMO-DC",
		"CEMS-DC", "UREK-DC", "UKFR-DC", "SINO-DC",
	}

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileName := info.Name()

		valid := false
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(strings.ToUpper(fileName), prefix) {
				valid = true
				break
			}
		}
		if !valid {
			return nil
		}

		deviceName := strings.Split(fileName, ".")[0]
		platform, err := nb.GetDevicePlatform(deviceName, netboxToken)
		if err != nil {
			AppendToOutput(output, scroll, "[ERROR] "+deviceName+": "+err.Error())
			return nil
		}
		if platform == "" {
			platform = "Unknown"

		}

		targetDir := filepath.Join(dstDir, platform)
		os.MkdirAll(targetDir, 0777)

		targetFile := filepath.Join(targetDir, fileName)
		if err := copyFile(path, targetFile); err != nil {
			AppendToOutput(output, scroll, "[ERROR] copy "+fileName+": "+err.Error())
			return nil
		}

		AppendToOutput(output, scroll, fmt.Sprintf("[%s] >> %s", platform, fileName))
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	return err
}

// appendToOutput безопасно добавляет строку в binding.String (для GUI)

func AppendToOutput(output binding.String, scroll *container.Scroll, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text + "\n")

	// безопасная автопрокрутка
	//go func() {
	//	time.Sleep(80 * time.Millisecond)
	//	if scroll != nil {
	//		scroll.ScrollToBottom()
	//	}
	//}()
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
