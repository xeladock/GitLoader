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
	// Удаляем старую папку назначения, если она есть
	if _, err := os.Stat(dstDir); err == nil {
		if removeErr := os.RemoveAll(dstDir); removeErr != nil {
			//appendOutput(output, fmt.Sprintf("Предупреждение: не удалось удалить старую папку %s: %v\n", dstDir, removeErr))
			// Не прерываем — попробуем перезаписать
		}
	}

	// Атомарное перемещение (переименование)
	if err := os.Rename(srcDir, dstDir); err != nil {
		//appendOutput(output, fmt.Sprintf("Ошибка перемещения %s → %s: %v\n", srcDir, dstDir, err))
		return err
	}
	Append(output, fmt.Sprintf("Завершено успешно."))

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
