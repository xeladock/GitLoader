package asis

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/widget"
)

// CopyAsIs — копирует ВСЕ файлы из srcDir в dstDir без сортировки.
// В окно вывода попадают только ошибки.
func CopyAsIs(srcDir, dstDir string, output *widget.Entry) error {

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			output.SetText(output.Text + fmt.Sprintf("❌ Ошибка чтения %s: %v\n", path, err))
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// создаём целевой каталог
		if mkErr := os.MkdirAll(dstDir, 0777); mkErr != nil {
			output.SetText(output.Text + fmt.Sprintf("❌ Ошибка создания каталога: %v\n", mkErr))
			return nil
		}

		dstFile := filepath.Join(dstDir, info.Name())

		// копирование файла
		if err := copyFile(path, dstFile); err != nil {
			output.SetText(output.Text + fmt.Sprintf("❌ Ошибка копирования %s: %v\n", info.Name(), err))
			return nil
		}

		return nil
	})

	return err
}

// copyFile — простое копирование файла
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
