//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func startServer(state *AppState) {
	binaryName := "mlcembedder.exe"
	foundPath := filepath.Join(state.BaseDir, binaryName)
	if _, err := os.Stat(foundPath); err != nil {
		foundPath = "./" + binaryName
	}

	logFile := filepath.Join(state.BaseDir, "server.log")

	// Windows specific: Start without window and redirect output
	cmdLine := fmt.Sprintf("start /B %s > \"%s\" 2>&1", binaryName, logFile)
	cmd := exec.Command("cmd", "/C", cmdLine)
	cmd.Dir = state.BaseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}
