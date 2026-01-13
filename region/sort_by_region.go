// region/sort_by_region.go
package region

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/data/binding"
)

var regionMap = map[string]string{
	"tsentr":        "Центр",
	"ural":          "Урал",
	"dalnij-vostok": "Дальний Восток",
	"yug":           "Юг",
	"volga":         "Волга",
	"severo":        "Северо-Запад",
	"sibir":         "Сибирь",
}

// GetRegionName — определяет название конечной папки по имени исходной папки
func GetRegionName(folderName string) string {
	lower := strings.ToLower(folderName)

	// ПРИОРИТЕТ 1: Москва → всегда "КЦ"
	if strings.Contains(lower, "moskva") {
		return "КЦ"
	}

	// ПРИОРИТЕТ 2: ищем по ключам из карты
	for key, name := range regionMap {
		if strings.Contains(lower, key) {
			return name
		}
	}

	return "Другое" // на всякий случай
}
func SortByRegion(srcDir, dstBase string, output binding.String, dcCheck, lanCheck bool) error {

	types := []string{}

	if dcCheck {
		os.RemoveAll(filepath.Join(dstBase, "ЦОД"))
		types = append(types, "ЦОД")
	}
	if lanCheck {
		os.RemoveAll(filepath.Join(dstBase, "ЛВС"))
		types = append(types, "ЛВС")
	}

	for _, t := range types {

		typeSrc := filepath.Join(srcDir, t)
		if !dirExists(typeSrc) {
			continue
		}

		typeDst := filepath.Join(dstBase, t)

		entries, err := os.ReadDir(typeSrc)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			folderName := entry.Name()
			regionName := GetRegionName(folderName)

			targetDir := filepath.Join(typeDst, regionName)
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				Append(output, fmt.Sprintf("Ошибка создания %s: %v\n", targetDir, err))
				continue
			}

			srcPath := filepath.Join(typeSrc, folderName)
			if err := copyDirContents(srcPath, targetDir, output); err != nil {
				Append(output, fmt.Sprintf("Ошибка копирования %s: %v\n", folderName, err))
			} else {
				Append(output, fmt.Sprintf("[%s/%s] ← %s\n", t, regionName, folderName))
			}
		}
	}

	os.RemoveAll(srcDir)
	return nil
}

// SortByRegion — главная функция для режима "Регион"
//func SortByRegion(srcDir, dstBase string, output binding.String, dcCheck, lanCheck bool) error {
//	// Создаём структуру: collected_files_clear/ЦОД/
//	//dstDir := filepath.Join(dstBase, "ЦОД")
//	//if err := os.MkdirAll(dstDir, 0777); err != nil {
//	//	return fmt.Errorf("не удалось создать папку ЦОД: %w", err)
//	//}
//
//	// Очищаем предыдущие результаты в ЦОД (по желанию — можно вынести в отдельную функцию)
//	os.RemoveAll(dstBase)
//	//os.MkdirAll(dstDir, 0777)
//	//Append(output, "Очищена папка collected_files_clear/ЦОД\n")
//
//	entries, err := os.ReadDir(srcDir)
//	if err != nil {
//		return fmt.Errorf("ошибка чтения папки configs: %w", err)
//	}
//
//	for _, entry := range entries {
//		if !entry.IsDir() {
//			continue // пропускаем файлы в корне (если есть)
//		}
//
//		folderName := entry.Name()
//		regionName := GetRegionName(folderName)
//		targetDir := filepath.Join(regionName)
//
//		if err := os.MkdirAll(targetDir, 0777); err != nil {
//			Append(output, fmt.Sprintf("Ошибка создания папки %s: %v\n", regionName, err))
//			continue
//		}
//
//		// Копируем всё содержимое папки
//		srcPath := filepath.Join(srcDir, folderName)
//		if err := copyDirContents(srcPath, targetDir, output); err != nil {
//			Append(output, fmt.Sprintf("Ошибка копирования %s: %v\n", folderName, err))
//		} else {
//			Append(output, fmt.Sprintf("[%s] ← %s\n", regionName, folderName))
//		}
//	}
//	if err == nil {
//		os.RemoveAll(srcDir)
//		//Append(output, "Исходная папка configs удалена.\n")
//	}
//
//	//Append(output, "\nСортировка по регионам завершена успешно.\n")
//
//	return nil
//}

// copyDirContents — копирует все файлы из src в dst (без самой папки)
func copyDirContents(src, dst string, output binding.String) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Рекурсивно копируем вложенные папки (если вдруг будут)
			if err := os.MkdirAll(dstPath, 0755); err != nil {
				return err
			}
			if err := copyDirContents(srcPath, dstPath, output); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

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

func Append(output binding.String, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text)
}
