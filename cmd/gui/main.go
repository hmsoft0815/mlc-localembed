// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

//go:build windows || darwin

package main

import (
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"
)

const (
	apiURL = "http://localhost:9142/api/health"
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
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
			if isOnline {
				mStatus.SetTitle("🟢 API Online")
				mStart.Hide()
				mStop.Show()
			} else {
				mStatus.SetTitle("🔴 Offline / Loading")
				mStart.Show()
				mStop.Hide()
			}
			<-ticker.C
		}
	}()

	// Event Loop
	for {
		select {
		case <-mStart.ClickedCh:
			startServer()
		case <-mStop.ClickedCh:
			stopServer()
		case <-mDownload.ClickedCh:
			runPreloader()
		case <-mLogs.ClickedCh:
			openLogs()
		case <-mQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func onExit() {
	// Cleanup if needed
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

func startServer() {
	binaryName := "mlcembedder"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	paths := []string{"./" + binaryName, "../MacOS/" + binaryName, "/usr/local/bin/" + binaryName}
	var foundPath string
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			foundPath = p
			break
		}
	}

	if foundPath == "" {
		foundPath = binaryName // Fallback to PATH
	}

	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", foundPath).Run()
	} else {
		exec.Command(foundPath).Start()
	}
}

func stopServer() {
	if runtime.GOOS == "windows" {
		exec.Command("taskkill", "/F", "/IM", "mlcembedder.exe").Run()
	} else {
		exec.Command("pkill", "-f", "mlcembedder").Run()
	}
}

func runPreloader() {
	if runtime.GOOS == "darwin" {
		script := "tell application \"Terminal\" to do script \"/usr/local/bin/localembed-preloader\""
		exec.Command("osascript", "-e", script).Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "localembed-preloader.exe").Run()
	}
}

func openLogs() {
	if runtime.GOOS == "darwin" {
		exec.Command("open", "/usr/local/var/log/localembed.log").Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "server.log").Run()
	}
}
