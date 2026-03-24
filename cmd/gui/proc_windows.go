//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func startServer(state *AppState) {
	binaryName := "mlcembedder.exe"
	logFile := "server.log"

	// Use cmd /C start /B to run detached and redirect both stdout and stderr
	// Important: We use double quotes for paths with potential spaces
	cmdLine := fmt.Sprintf("start /B %s > %s 2>&1", binaryName, logFile)
	cmd := exec.Command("cmd", "/C", cmdLine)
	cmd.Dir = state.BaseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
