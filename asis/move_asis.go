package asis

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2/data/binding"
)

//type TextOutput interface {
//	SetText(string)
//}

// MoveAsIs — просто переименовывает папку configs → configs_file_clear
func MoveAsIs(srcDir, dstDir string, output binding.String) error {
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

	if _, err := os.Stat(dstDir); err == nil {
		if removeErr := os.RemoveAll(dstDir); removeErr != nil {
			//appendOutput(output, fmt.Sprintf("Предупреждение: не удалось удалить старую папку %s: %v\n", dstDir, removeErr))
			// Не прерываем — попробуем перезаписать
		}
	}

	println(srcDir)
	//Атомарное перемещение (переименование)
	if err := os.Rename(srcDir, dstDir); err != nil {
		//appendOutput(output, fmt.Sprintf("Ошибка перемещения %s → %s: %v\n", srcDir, dstDir, err))
		return err
	}

	Append(output, fmt.Sprintf("Загружено успешно (Как есть)!\n"))

	//Append(output, fmt.Sprintf("Папка успешно перемещена → %s\n", dstDir))
	return nil
}

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
