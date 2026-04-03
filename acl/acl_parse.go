package acl

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

//var aclPlatforms = []string{
//	"Cisco ASA",
//	"Cisco FXOS",
//	"Cisco IOS",
//	"Cisco IOS XE",
//	"Cisco NX-OS",
//	"FortiOS",
//	"B4COM BCOM-OS-DC",
//	"EdgeCore",
//	"IBM/Lenovo Network OS",
//	"HP ProCurve",
//	"Dell Networking OS",
//	"Juniper Junos",
//	"Eltex",
//	"Cisco IOS XR",
//	"Cisco PIX",
//	"QTECH NOS",
//	"Raisecom",
//	"ELTEX ESR",
//}

var aclPlatforms2 = map[string]struct{}{
	"Cisco ASA":             {},
	"Cisco IOS":             {},
	"Cisco IOS XE":          {},
	"Cisco NX-OS":           {},
	"Cisco FXOS":            {},
	"Eltex":                 {},
	"FortiOS":               {},
	"Huawei VRP":            {},
	"Eltex ESR":             {},
	"B4COM BCOM-OS-DC":      {},
	"EdgeCore":              {},
	"IBM/Lenovo Network OS": {},
	//"IBM_Lenovo Network OS": {},
	"HP ProCurve":        {},
	"HPE OfficeConnect":  {},
	"HPE Comware 1910":   {},
	"HPE Comware":        {},
	"3Com Comware 1910":  {},
	"Huawei VRP 2403":    {},
	"Dell Networking OS": {},
	"Juniper Junos":      {},
	"Cisco IOS XR":       {},
	"Cisco PIX":          {},
	"QTECH NOS":          {},
	"Raisecom":           {},
}

func SortFilesByACL(
	srcDir, dstBase, netboxToken string,
	output binding.String,
	scroll *container.Scroll,
	dcbool, lanbool bool,
) error {

	typeDirs := []string{}
	//println(dstBase, "- dstbase", srcDir, "- srcdir")
	if dcbool {
		os.RemoveAll(dstBase)
		os.MkdirAll(dstBase, 0755)
		//os.RemoveAll(filepath.Join(dstBase, "ЦОД"))
		typeDirs = append(typeDirs, "ЦОД")
	}
	if lanbool {
		//println(dstBase, "очистка lan для платформы")
		os.RemoveAll(dstBase)
		os.MkdirAll(dstBase, 0755)
		//os.RemoveAll(filepath.Join(dstBase, "ЛВС"))
		typeDirs = append(typeDirs, "ЛВС")
	}

	allowedRoots := []string{"CE", "SI", "SZ", "UR", "PR", "UK", "DV"}
	//allowedRoots := []string{"SI"}
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
			//println(fileName)
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
			//println(platform)
			if err != nil || platform == "" {
				return nil
			} else if IsACLPlatform(platform) {
				platform = strings.ReplaceAll(platform, "/", "_")
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

			}
			//println()

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

func IsACLPlatform(platform string) bool {
	_, exists := aclPlatforms2[platform]
	return exists
}
