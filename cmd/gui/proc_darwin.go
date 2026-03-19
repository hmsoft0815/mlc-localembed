//go:build darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func startServer(state *AppState) {
	binaryName := "mlcembedder"
	foundPath := filepath.Join(state.BaseDir, binaryName)
	if _, err := os.Stat(foundPath); err != nil {
		foundPath = "./" + binaryName
	}

	cmd := exec.Command(foundPath)
	cmd.Dir = state.BaseDir
	cmd.Start()
}
