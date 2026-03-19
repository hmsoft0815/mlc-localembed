// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"net/http"
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
	systray.SetTooltip("Local Text Embedding Engine")

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
	if runtime.GOOS == "windows" {
		// Try to start as a service or direct process
		exec.Command("cmd", "/C", "start", "mlcembedder.exe").Run()
	}
}

func stopServer() {
	if runtime.GOOS == "windows" {
		exec.Command("taskkill", "/F", "/IM", "mlcembedder.exe").Run()
	}
}

func runPreloader() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "localembed-preloader.exe").Run()
	}
}

func openLogs() {
	// Simple log opening for now, assuming server.log exists in the same folder
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", "server.log").Run()
	}
}
