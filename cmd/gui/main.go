// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

//go:build windows || darwin

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/getlantern/systray"
)

const (
	apiURL = "http://localhost:9142/api/health"
)

type AppState struct {
	IsServerReachable bool
	BaseDir           string
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)
	
	state := &AppState{
		IsServerReachable: false,
		BaseDir:           baseDir,
	}

	systray.SetIcon(iconData)
	systray.SetTitle("LocalEmbed")
	systray.SetTooltip("Local Text Embedding Engine Manager")

	mStatus := systray.AddMenuItem("Status: Checking...", "")
	mStatus.Disable()
	systray.AddSeparator()

	mStart := systray.AddMenuItem("Start Server", "Start the local embedding server")
	mStop := systray.AddMenuItem("Stop Server", "Stop the local embedding server")
	systray.AddSeparator()

	mDownload := systray.AddMenuItem("Download Models...", "Run the model preloader")
	mLogs := systray.AddMenuItem("Open Logs", "View server logs")
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Exit the application")

	// Start health check loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			isOnline := checkHealth()
			if isOnline != state.IsServerReachable {
				state.IsServerReachable = isOnline
				if isOnline {
					mStatus.SetTitle("🟢 API Online")
					mStart.Hide()
					mStop.Show()
				} else {
					mStatus.SetTitle("🔴 Offline / Loading")
					mStart.Show()
					mStop.Hide()
				}
			}
			<-ticker.C
		}
	}()

	// Event Loop
	for {
		select {
		case <-mStart.ClickedCh:
			startServer(state)
		case <-mStop.ClickedCh:
			stopServer()
		case <-mDownload.ClickedCh:
			runPreloader(state)
		case <-mLogs.ClickedCh:
			openLogs(state)
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func onExit() {
}

func checkHealth() bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func startServer(state *AppState) {
	binaryName := "mlcembedder"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	foundPath := filepath.Join(state.BaseDir, binaryName)
	if _, err := os.Stat(foundPath); err != nil {
		// Fallback to relative
		foundPath = "./" + binaryName
	}

	logFile := filepath.Join(state.BaseDir, "server.log")

	if runtime.GOOS == "windows" {
		// Windows specific: Start without window and redirect output
		// We use cmd /C to handle the redirection correctly
		cmdLine := fmt.Sprintf("start /B %s > \"%s\" 2>&1", binaryName, logFile)
		cmd := exec.Command("cmd", "/C", cmdLine)
		cmd.Dir = state.BaseDir
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		cmd.Run()
	} else {
		cmd := exec.Command(foundPath)
		cmd.Dir = state.BaseDir
		cmd.Start()
	}
}

func stopServer() {
	if runtime.GOOS == "windows" {
		exec.Command("taskkill", "/F", "/IM", "mlcembedder.exe").Run()
	} else {
		exec.Command("pkill", "-f", "mlcembedder").Run()
	}
}

func runPreloader(state *AppState) {
	if runtime.GOOS == "darwin" {
		script := "tell application \"Terminal\" to do script \"/usr/local/bin/localembed-preloader\""
		exec.Command("osascript", "-e", script).Run()
	} else if runtime.GOOS == "windows" {
		preloader := filepath.Join(state.BaseDir, "preloader.exe")
		// Start in a new visible console window
		exec.Command("cmd", "/C", "start", preloader).Run()
	}
}

func openLogs(state *AppState) {
	logFile := filepath.Join(state.BaseDir, "server.log")
	
	// Create empty log if not exists to prevent "file not found" errors
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		os.WriteFile(logFile, []byte("--- LocalEmbed Log Started ---\n"), 0644)
	}

	if runtime.GOOS == "darwin" {
		exec.Command("open", logFile).Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "notepad.exe", logFile).Run()
	}
}
