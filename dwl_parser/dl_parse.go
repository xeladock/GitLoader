package dwl_parser

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
)

func DownloadACLParser(targetDir string) error {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // ← вот это главное
		},
	}

	client := &http.Client{Transport: tr}

	url := "https://portal.net.rt.ru/static/downloads/ACL-Parser-AltLinux1.17.3"

	filePath := filepath.Join(targetDir, "..", "..", "ACL-Parser")
	//filepath.Join(targetDir, "..", "..")
	//localSize := int64(0)
	//if info, err := os.Stat(filePath); err == nil {
	//	localSize = info.Size()
	//}
	// Отключаем проверку сертификата

	if _, err := os.Stat(filePath); err == nil {
		//appendOutput(output, "ℹ️ ACL-Parser уже существует в папке. Скачивание пропущено.\n")
		return nil
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("не удалось скачать ACL-Parser: %w", err)
	}
	defer resp.Body.Close()

	// Создаём файл в целевой папке

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка при скачивании ACL-Parser: статус %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл ACL-Parser: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка записи файла ACL-Parser: %w", err)
	}

	if err := os.Chmod(filePath, 0755); err != nil {
		//println("сделали chmod\n")
		//appendOutput(output, fmt.Sprintf("⚠️ Не удалось сделать файл исполняемым: %v\n", err))
	}

	return nil
}

//downloadACLParser — скачивает файл только если он новее (по размеру)

//func DownloadACLParser(targetDir string, output binding.String, scroll *container.Scroll) {
//		tr := &http.Transport{
//			TLSClientConfig: &tls.Config{
//				InsecureSkipVerify: true,
//			},
//		}
//
//		client := &http.Client{Transport: tr}
//
//		const url = "https://portal.net.rt.ru/static/downloads/ACL-Parser-AltLinux1.17.3"
//		localFilePath := filepath.Join(targetDir, "..", "..", "ACL-Parser-AltLinux")
//
//		// Проверяем размер локального файла
//		localSize := int64(0)
//		if info, err := os.Stat(localFilePath); err == nil {
//			localSize = info.Size()
//		}
//		req, _ := http.NewRequest("HEAD", url, nil)
//		resp, err := client.Do(req)
//		if err != nil {
//			appendOutput(output, "❌ Не удалось подключиться к portal.net.rt.ru\n")
//			return
//		}
//		defer resp.Body.Close()
//
//		if resp.StatusCode != http.StatusOK {
//			appendOutput(output, fmt.Sprintf("❌ ACL-parser. Portal.net.rt.ru вернул ошибку: %d\n", resp.StatusCode))
//			return
//		}
//
//		remoteSize := resp.ContentLength
//
//		// Сравниваем размеры
//		if remoteSize == localSize && localSize > 0 {
//			//println("ACL на месте\n")
//			//appendOutput(output, fmt.Sprintf("ℹ️ ACL-Parser уже актуален (%.1f MB)\n", float64(localSize)/1024/1024))
//			return
//		}
//
//		// Скачиваем файл (новый или обновлённый)
//		if localSize == 0 {
//			appendOutput(output, "📥 Скачиваем ACL-Parser...\n")
//		} else {
//			appendOutput(output, "📥🔄 Обнаружена новая версия ACL-Parser. Обновляем...\n")
//		}
//
//		if err := downloadFile(url, localFilePath, output, scroll); err != nil {
//			appendOutput(output, "❌ "+err.Error()+"\n")
//		}
//		if err := os.Chmod(localFilePath, 0755); err != nil {
//			println("сделали chmod\n")
//			//appendOutput(output, fmt.Sprintf("⚠️ Не удалось сделать файл исполняемым: %v\n", err))
//		}
//	}
//func DownloadACLParser(targetDir string, output binding.String, scroll *container.Scroll) {
//	// Один клиент на всю операцию
//	tr := &http.Transport{
//		TLSClientConfig: &tls.Config{
//			InsecureSkipVerify: true,
//		},
//	}
//	client := &http.Client{
//		Transport: tr,
//		Timeout:   10 * time.Second,
//	}
//
//	const url = "https://https://portal.net.rt.ru/static/downloads/ACL-Parser-AltLinux1.17.3"
//	localFilePath := filepath.Join(targetDir, "..", "..", "ACL-Parser-AltLinux")
//
//	// Создаём папку, если её нет
//	if err := os.MkdirAll(filepath.Dir(localFilePath), 0755); err != nil {
//		appendOutput(output, "❌ Не удалось создать папку для ACL-Parser\n")
//		return
//	}
//
//	// Проверяем размер локального файла
//	localSize := int64(0)
//	if info, err := os.Stat(localFilePath); err == nil {
//		localSize = info.Size()
//	}
//
//	// HEAD-запрос
//	req, _ := http.NewRequest("HEAD", url, nil)
//	resp, err := client.Do(req)
//	if err != nil {
//		appendOutput(output, "❌ Не удалось подключиться к site.local\n")
//		return
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		appendOutput(output, fmt.Sprintf("❌ site.local вернул ошибку: %d\n", resp.StatusCode))
//		return
//	}
//
//	remoteSize := resp.ContentLength
//
//	if remoteSize > 0 && localSize >= remoteSize {
//		appendOutput(output, "✅ ACL-Parser уже актуален\n")
//		return
//	}
//
//	if localSize == 0 {
//		appendOutput(output, "📥 Скачиваем ACL-Parser...\n")
//	} else {
//		appendOutput(output, "🔄 Обнаружена новая версия ACL-Parser. Обновляем...\n")
//	}
//
//	// Скачиваем через тот же client!
//	if err := downloadFileWithClient(url, localFilePath, client, output, scroll); err != nil {
//		appendOutput(output, "❌ "+err.Error()+"\n")
//		return
//	}
//
//	if err := os.Chmod(localFilePath, 0755); err != nil {
//		appendOutput(output, "⚠️ Не удалось выполнить chmod +x\n")
//	} else {
//		appendOutput(output, "✅ ACL-Parser скачан и сделан исполняемым (+x)\n")
//	}
//}

//func downloadFileWithClient(url, destPath string, client *http.Client, output binding.String, scroll *container.Scroll) error {
//	resp, err := client.Get(url)
//	if err != nil {
//		return fmt.Errorf("не удалось скачать файл: %w", err)
//	}
//	defer resp.Body.Close()
//
//	out, err := os.Create(destPath)
//	if err != nil {
//		return fmt.Errorf("не удалось создать файл: %w", err)
//	}
//	defer out.Close()
//
//	_, err = io.Copy(out, resp.Body)
//	if err != nil {
//		return fmt.Errorf("ошибка записи файла: %w", err)
//	}
//
//	appendOutput(output, "✅ ACL-Parser успешно скачан.\n")
//	return nil
//}

//func downloadFile(url, destPath string, output binding.String, scroll *container.Scroll) error {
//	tr := &http.Transport{
//		TLSClientConfig: &tls.Config{
//			InsecureSkipVerify: true,
//		},
//	}
//	client := &http.Client{Transport: tr}
//	resp, err := client.Get(url)
//	if err != nil {
//		return fmt.Errorf("не удалось скачать файл: %w", err)
//	}
//	defer resp.Body.Close()
//
//	out, err := os.Create(destPath)
//	if err != nil {
//		return fmt.Errorf("не удалось создать файл: %w", err)
//	}
//	defer out.Close()
//
//	_, err = io.Copy(out, resp.Body)
//	if err != nil {
//		return fmt.Errorf("ошибка записи файла: %w", err)
//	}
//
//	appendOutput(output, "✅ ACL-Parser успешно скачан.\n")
//	return nil
//}

func appendOutput(output binding.String, msg string) {
	fyne.Do(func() {
		current, _ := output.Get()
		_ = output.Set(current + msg)
	})
}
