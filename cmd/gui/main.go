// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	apiURL = "http://localhost:9142/api/health"
)

type AppState struct {
	IsServerReachable bool
	ServerProcess     *exec.Cmd
}

func main() {
	app := application.New(application.Options{
		Name:        "LocalEmbed",
		Description: "Local Text Embedding Engine Manager",
		Assets: application.AssetOptions{
			Handler: nil, // We don't need a main window for now
		},
	})

	state := &AppState{
		IsServerReachable: false,
	}

	// Define the tray menu
	menu := app.NewMenu()
	statusItem := menu.Add("Status: Checking...")
	statusItem.Enabled(false)

	menu.AddSeparator()

	startItem := menu.Add("Start Server").OnClick(func(ctx *application.Context) {
		startServer(state)
	})

	stopItem := menu.Add("Stop Server").OnClick(func(ctx *application.Context) {
		stopServer(state)
	})

	menu.AddSeparator()

	menu.Add("Download Models...").OnClick(func(ctx *application.Context) {
		runPreloader()
	})

	menu.Add("Open Logs").OnClick(func(ctx *application.Context) {
		openLogs()
	})

	menu.AddSeparator()

	menu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	// Create the system tray
	tray := app.NewSystemTray()
	// tray.SetIcon(iconData) // Set icon here later

	tray.SetMenu(menu)

	// Health check loop
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for {
			reachable := checkHealth()
			if reachable != state.IsServerReachable {
				state.IsServerReachable = reachable
				app.InvokeMainThread(func() {
					if reachable {
						statusItem.SetLabel("🟢 API Online")
						startItem.SetHidden(true)
						stopItem.SetHidden(false)
					} else {
						statusItem.SetLabel("🔴 Offline / Loading")
						startItem.SetHidden(false)
						stopItem.SetHidden(true)
					}
					// menu.Update() // Wails v3 handles menu updates dynamically
				})
			}
			<-ticker.C
		}
	}()

	err := app.Run()
	if err != nil {
		fmt.Printf("Wails app error: %v\n", err)
	}
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

	// Try to find the binary relative to current path or in standard locations
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

	cmd := exec.Command(foundPath)
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	state.ServerProcess = cmd
}

func stopServer(state *AppState) {
	if state.ServerProcess != nil && state.ServerProcess.Process != nil {
		state.ServerProcess.Process.Kill()
		state.ServerProcess = nil
	} else {
		// Fallback for Windows: Kill by name
		if runtime.GOOS == "windows" {
			exec.Command("taskkill", "/F", "/IM", "mlcembedder.exe").Run()
		}
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
	logPath := "server.log"
	if runtime.GOOS == "darwin" {
		exec.Command("open", "/usr/local/var/log/localembed.log").Run()
	} else if runtime.GOOS == "windows" {
		exec.Command("cmd", "/C", "start", logPath).Run()
	}
}
