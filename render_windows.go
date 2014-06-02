package main

import (
	"os/exec"
)


func init() {
	go resizeWinTerminal()
}

func resizeWinTerminal() {
	cmd := exec.Command("mode", "con:", "cols=140", "lines=78")
	cmd.Run()
}
