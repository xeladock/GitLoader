package asis

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2/data/binding"
)

//type TextOutput interface {
//	SetText(string)
//}

// MoveAsIs — просто переименовывает папку configs → configs_file_clear
func MoveAsIs(srcDir, dstDir string, output binding.String, dcCheck, lanCheck bool) error {
	println(dstDir)

	//hasDC := filepath.Join(dstDir, "ЦОД")
	//hasLAN := filepath.Join(dstDir, "ЛВС")
	//dcName := "ЦОД"
	//lanName := "ЛВС"
	//
	//srcDC := filepath.Join(srcDir, dcName)
	//srcLAN := filepath.Join(srcDir, lanName)
	//
	//dstDC := filepath.Join(dstDir, dcName)
	//dstLAN := filepath.Join(dstDir, lanName)
	//
	//hasDst := dirExists(dstDir)
	//
	//hasDC := dirExists(dstDC)
	//hasLAN := dirExists(dstLAN)
	if !dirExists(dstDir) {
		return os.Rename(srcDir, dstDir)
	}

	// 2. иначе — синхронизация подпапок
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		src := filepath.Join(srcDir, e.Name())
		dst := filepath.Join(dstDir, e.Name())

		// заменить существующую подпапку
		if dirExists(dst) {
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
		}

		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}

	// 3. удалить пустой /configs

	//if dcCheck {
	//	Append(output, fmt.Sprintf("Папка ЦОД загружена успешно (Как есть)!\n"))
	//}
	//if lanCheck {
	//	Append(output, fmt.Sprintf("Папка ЛВС загружена успешно (Как есть)!\n"))
	//}
	println(srcDir)
	parent := filepath.Dir(srcDir)
	println(parent)
	return os.RemoveAll(parent)
}

//parent := filepath.Dir(srcDir)
//child := filepath.Base(srcDir)
//// Удаляем старую папку назначения, если она есть
//dstDir = "collected_files_clear"
//lanDir := "ЛВС"
//dcDir := "ЦОД"
//srcDir = filepath.Dir(srcDir)
////srcLan := filepath.Join(srcDir, lanDir)
////srcDc := filepath.Join(srcDir, dcDir)
//
//dstLan := filepath.Join(dstDir, lanDir)
//dstDc := filepath.Join(dstDir, dcDir)
//
////if dirExists(pa) {}
//
//if child == "ЛВС" {
//	if !dirExists(dstLan) && !dirExists(dstDc) {
//		os.Rename(srcDir, dstLan)
//	} else if dirExists(dstLan) && !dirExists(dstDc) {
//		os.RemoveAll(dstDir)
//		time.Sleep(1)
//		os.Rename(srcDir, dstLan)
//	} else if dirExists(dstLan) && dirExists(dstDc) {
//		os.RemoveAll(dstLan)
//		time.Sleep(1)
//		os.Rename(srcDir, dstLan)
//	}
//}
//
//// ЦОД
//if child == "ЦОД" {
//	if !dirExists(dstLan) && !dirExists(dstDc) {
//		os.Rename(srcDir, dstDc)
//	} else if dirExists(dstDc) && !dirExists(dstLan) {
//		os.RemoveAll(dstDir)
//		time.Sleep(1)
//		os.Rename(srcDir, dstDc)
//	} else if dirExists(dstLan) && dirExists(dstDc) {
//		os.RemoveAll(dstDc)
//		time.Sleep(1)
//		os.Rename(srcDir, dstDc)
//	}
//}

//if os.Stat(filepath.Join(srcDir, lanDir) && os.Stat(dstDir, lanDir) && !os.Stat(dstDir, dcDir){
//	os.RemoveAll(dstDir, lanDir)
//} else if os.Stat(filepath.Join(srcDir, lanDir) && os.Stat(dstDir, lanDir) && os.Stat(dstDir, dcDir) {
//	os.Rename(filepath.Join(srcDir, lanDir), filepath.Join(dstDir, lanDir))
//} else if os.Stat(filepath.Join(srcDir, dcDir) && os.Stat(dstDir, dcDir) && !os.Stat(dstDir, lanDir) {
//	os.RemoveAll(dstDir, dcDir)
//} else if os.Stat(filepath.Join(srcDir, dcDir) && os.Stat(dstDir, dcDir) && os.Stat(dstDir, lanDir){
//	os.Rename(filepath.Join(srcDir, dcDir), filepath.Join(dstDir, dcDir)) }

//if _, err := os.Stat(dstDir); err == nil {
//	if removeErr := os.RemoveAll(dstDir); removeErr != nil {
//		//appendOutput(output, fmt.Sprintf("Предупреждение: не удалось удалить старую папку %s: %v\n", dstDir, removeErr))
//		// Не прерываем — попробуем перезаписать
//	}
//}

//println(srcDir)
//Атомарное перемещение (переименование)
//if err := os.Rename(srcDir, dstDir); err != nil {
//	//appendOutput(output, fmt.Sprintf("Ошибка перемещения %s → %s: %v\n", srcDir, dstDir, err))
//	return err
//}

//Append(output, fmt.Sprintf("Папка успешно перемещена → %s\n", dstDir))
//return nil

// Вспомогательная функция (можно вынести в общий utils, если используется везде)
//
//	func appendOutput(entry *widget.Entry, text string) {
//		if entry != nil {
//			entry.SetText(entry.Text + text)
//		}
func Append(output binding.String, text string) {
	current, _ := output.Get()
	_ = output.Set(current + text)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

//func canReuseDir(wantDC, wantLAN, hasDC, hasLAN bool) bool {
//
//	// ХОТИМ ОБА — можно ВСЕГДА
//	if wantDC && wantLAN {
//		return true
//	}
//
//	// хотим только ЦОД, но есть ЛВС → нельзя
//	if wantDC && !wantLAN && hasLAN {
//		return false
//	}
//
//	// хотим только ЛВС, но есть ЦОД → нельзя
//	if wantLAN && !wantDC && hasDC {
//		return false
//	}
//
//	return true
//}
