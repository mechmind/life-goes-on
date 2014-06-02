package main

import (
	"os/exec"
)

func prepareTerminal() {
	cmd := exec.Command("mode", "con:", "cols=140", "lines=78")
	cmd.Run()
}
