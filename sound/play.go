package sound

import (
	"bytes"
	"embed"
	"os/exec"
)

//go:embed *.mp3

var sounds embed.FS
var yesSound []byte
var noSound []byte
var banSound []byte

func init() {
	var err error
	yesSound, err = sounds.ReadFile("yes.mp3")
	noSound, err = sounds.ReadFile("no.mp3")
	banSound, err = sounds.ReadFile("ban.mp3")
	if err != nil {
		println(err.Error())
		// можно логгировать или игнорировать
	}
}

func PlayYes() {
	//println(len(yesSound))
	//if len(yesSound) == 0 {
	//	return
	//}
	cmd := exec.Command("paplay")
	cmd.Stdin = bytes.NewReader(yesSound)
	_ = cmd.Run()
}

func PlayNo() {
	//if len(noSound) == 0 {
	//	return
	//}
	cmd := exec.Command("paplay")
	cmd.Stdin = bytes.NewReader(noSound)
	_ = cmd.Run()
}

func PlayBan() {
	//if len(noSound) == 0 {
	//	return
	//}
	cmd := exec.Command("paplay")
	cmd.Stdin = bytes.NewReader(banSound)
	_ = cmd.Run()
}
