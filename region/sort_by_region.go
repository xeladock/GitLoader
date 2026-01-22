// region/sort_by_region.go
package region

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func SortByRegion(srcDir, dstBase string, output binding.String, scroll *container.Scroll, dcCheck, lanCheck bool) error {
	entries, err := os.ReadDir(srcDir)

	println(dstBase, "- dstbase", srcDir, "- srcdir")

	_ = os.RemoveAll(dstBase)

	//if dcCheck {
	//	if dirExists(dstBase) {
	//		os.RemoveAll(dstBase)
	//	}
	//}
	//
	//if lanCheck {
	//	if dirExists(dstBase) {
	//		os.RemoveAll(dstBase)
	//	}
	//} //types = append(types, "ЛВС")

	if err != nil {
		Append(output, scroll, fmt.Sprintf("Ошибка чтения папки %s: %v\n", srcDir, err))
		return err
	}

	// Создаём целевую папку dstBase, если её нет
	if err := os.MkdirAll(dstBase, 0755); err != nil {
		Append(output, scroll, fmt.Sprintf("Ошибка создания папки %s: %v\n", dstBase, err))
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // пропускаем файлы в корне (если есть)
		}

		folderName := entry.Name()
		regionName := GetRegionName(folderName)
		targetDir := filepath.Join(dstBase, regionName) // ← ГЛАВНОЕ ИЗМЕНЕНИЕ: используем dstBase

		if err := os.MkdirAll(targetDir, 0777); err != nil {
			Append(output, scroll, fmt.Sprintf("Ошибка создания папки %s: %v\n", regionName, err))
			continue
		}

		// Копируем всё содержимое папки
		srcPath := filepath.Join(srcDir, folderName)
		if err := copyDirContents(srcPath, targetDir, output); err != nil {
			Append(output, scroll, fmt.Sprintf("Ошибка копирования %s: %v\n", folderName, err))
		} else {
			Append(output, scroll, fmt.Sprintf("[%s] ← %s\n", regionName, folderName))
		}
	}
	if err == nil {
		os.RemoveAll(filepath.Dir(srcDir))
		//Append(output, "Исходная папка configs удалена.\n")
	}

	//Append(output, "\nСортировка по регионам завершена успешно.\n")

	return nil
}

//21212

//	func SortByRegion(srcDir, dstBase string, output binding.String, dcCheck, lanCheck bool) error {
//		println(dstBase, "- dstbase", srcDir, "- srcdir")
//		types := []string{}
//
//		_ = os.RemoveAll(dstBase)
//
//		if dcCheck {
//			//os.RemoveAll(filepath.Join(dstBase, "ЦОД"))
//			types = append(types, filepath.Base(srcDir))
//		}
//		if lanCheck {
//			//os.RemoveAll(filepath.Join(dstBase, "ЛВС"))
//			types = append(types, filepath.Base(srcDir))
//		}
//
//		for _, t := range types {
//			println(t, "t")
//			Append(output, fmt.Sprintf("Начинаю сортировку для папки **%s** \n", t))
//			typeSrc := filepath.Join(filepath.Dir(srcDir), t)
//			println(typeSrc, "typesrc")
//			if !dirExists(typeSrc) {
//				continue
//			}
//
//			typeDst := dstBase
//
//			entries, err := os.ReadDir(typeSrc)
//			if err != nil {
//				return err
//			}
//
//			for _, entry := range entries {
//				if !entry.IsDir() {
//					continue
//				}
//
//				folderName := entry.Name()
//				regionName := GetRegionName(folderName)
//
//				targetDir := filepath.Join(typeDst, regionName)
//				if err := os.MkdirAll(targetDir, 0755); err != nil {
//					Append(output, fmt.Sprintf("Ошибка создания %s: %v\n", targetDir, err))
//					continue
//				}
//
//				srcPath := filepath.Join(typeSrc, folderName)
//				if err := copyDirContents(srcPath, targetDir, output); err != nil {
//					Append(output, fmt.Sprintf("Ошибка копирования %s: %v\n", folderName, err))
//				} else {
//					Append(output, fmt.Sprintf("[%s/%s] ← %s\n", t, regionName, folderName))
//				}
//			}
//		}
//
//		os.RemoveAll(srcDir)
//		return nil
//	}
//func SortByRegion(srcDir, dstBase string, output binding.String, dcCheck, lanCheck bool) error {
//	println(srcDir, dstBase)
//	entries, err := os.ReadDir(srcDir)
//	if err != nil {
//		Append(output, fmt.Sprintf("Ошибка чтения папки %s: %v\n", srcDir, err))
//		return err
//	}
//
//	// Полностью очищаем целевую папку
//	if err := os.RemoveAll(dstBase); err != nil {
//		Append(output, fmt.Sprintf("Ошибка очистки %s: %v\n", dstBase, err))
//		return err
//	}
//
//	if err := os.MkdirAll(dstBase, 0755); err != nil {
//		Append(output, fmt.Sprintf("Ошибка создания папки %s: %v\n", dstBase, err))
//		return err
//	}
//
//	for _, entry := range entries {
//		if !entry.IsDir() {
//			continue
//		}
//
//		folderName := entry.Name()
//		regionName := GetRegionName(folderName)
//		targetDir := filepath.Join(dstBase, regionName)
//
//		if err := os.MkdirAll(targetDir, 0755); err != nil {
//			Append(output, fmt.Sprintf("Ошибка создания %s: %v\n", targetDir, err))
//			continue
//		}
//
//		srcPath := filepath.Join(srcDir, folderName)
//		if err := copyDirContents(srcPath, targetDir, output); err != nil {
//			Append(output, fmt.Sprintf("Ошибка копирования %s: %v\n", folderName, err))
//		} else {
//			Append(output, fmt.Sprintf("[%s] ← %s\n", regionName, folderName))
//		}
//	}
//
//	// Удаляем исходную папку ПОСЛЕ успешной обработки
//	if err := os.RemoveAll(srcDir); err != nil {
//		Append(output, fmt.Sprintf("Не удалось удалить %s: %v\n", srcDir, err))
//	}
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

func Append(output binding.String, scroll *container.Scroll, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text)
	fyne.Do(func() {
		scroll.ScrollToBottom()
		//scroll.Refresh()
	})
}
