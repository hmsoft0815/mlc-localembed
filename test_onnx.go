// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

func main() {
	onnxPath := os.Getenv("ONNX_PATH")
	fmt.Printf("Testing ONNX_PATH: %s\n", onnxPath)

	ort.SetSharedLibraryPath(onnxPath)
	err := ort.InitializeEnvironment()
	if err != nil {
		fmt.Printf("Initialization failed: %v\n", err)
		os.Exit(1)
	}
	defer ort.DestroyEnvironment()
	fmt.Println("ONNX Runtime initialized successfully!")
}
