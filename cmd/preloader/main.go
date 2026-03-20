// Copyright (c) 2026 Michael Lechner
// Licensed under the MIT License. See LICENSE file in the project root for full license information.

package main

// note - it is difficult to test this in an automated way, as it depends on external resources (Hugging Face repos) and user input (HF token).
// The main purpose of this preloader is to simplify the initial setup for users by automatically downloading necessary model files based on a config.yaml.
// It can be run manually before starting the main application.

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Storage struct {
		CacheDir string `yaml:"cache_dir"`
	} `yaml:"storage"`
	Models struct {
		Available []struct {
			Name       string `yaml:"name"`
			SourceRepo string `yaml:"source_repo"`
			ModelFile  string `yaml:"model_file"`
		} `yaml:"available"`
	} `yaml:"models"`
}

func downloadFile(url, dest, token string) error {
	fmt.Printf("Downloading %s\n", url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status: %s", resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	configPath := "config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try parent directory (if run from bin/)
		if _, err := os.Stat("../config.yaml"); err == nil {
			configPath = "../config.yaml"
		}
	}

	configFile, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Fehler beim Lesen der %s: %v", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Fehler beim Parsen der YAML: %v", err)
	}

	token := os.Getenv("HF_TOKEN")
	cacheProxy := os.Getenv("MLC_CACHE_PROXY")

	if token == "" {
		fmt.Print("Hugging Face Token (HF_TOKEN) nicht gefunden. Bitte eingeben (oder ENTER für anonym): ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		token = strings.TrimSpace(input)
	}

	// Definition der Repositories
	repos := map[string]string{
		"multilingual-e5-small":  "intfloat/multilingual-e5-small",
		"BAAI/bge-small-en-v1.5": "qdrant/bge-small-en-v1.5-onnx-q",
	}

	for _, m := range config.Models.Available {
		repo := m.SourceRepo
		if repo == "" {
			repo = repos[m.Name]
		}

		if repo == "" {
			fmt.Printf("Kein Repo für %s definiert.\n", m.Name)
			continue
		}

		fmt.Printf("\n--- Lade Modell: %s ---\n", m.Name)

		folderName := "models--" + strings.ReplaceAll(m.Name, "/", "--")
		if !strings.Contains(m.Name, "/") {
			folderName = "models--qdrant--" + m.Name
		}

		destDir := filepath.Join(config.Storage.CacheDir, folderName)
		
		// Handle Cache Proxy
		baseURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main", repo)
		if cacheProxy != "" {
			fmt.Printf("Nutze Cache-Proxy: %s\n", cacheProxy)
			baseURL = fmt.Sprintf("%s/huggingface/%s/resolve/main", strings.TrimSuffix(cacheProxy, "/"), repo)
		}

		// 1. JSON Konfigurationsdateien laden
		jsonFiles := []string{"tokenizer.json", "config.json", "tokenizer_config.json", "special_tokens_map.json"}
		for _, file := range jsonFiles {
			destPath := filepath.Join(destDir, file)
			if _, err := os.Stat(destPath); err == nil {
				fmt.Printf("OK: %s vorhanden.\n", file)
				continue
			}

			urls := []string{
				fmt.Sprintf("%s/%s", baseURL, file),
				fmt.Sprintf("%s/onnx/%s", baseURL, file),
			}

			for _, url := range urls {
				if err := downloadFile(url, destPath, token); err == nil {
					fmt.Printf("OK: %s geladen\n", file)
					break
				}
			}
		}

		// 2. Modellgewichte laden
		onnxFile := m.ModelFile
		if onnxFile == "" {
			onnxFile = "model.onnx"
		}

		// Simple normalization: if we expect it in the root of destDir
		// but it has a path in HF repo (like "onnx/model.onnx")
		localOnnxFile := filepath.Base(onnxFile)
		destPath := filepath.Join(destDir, localOnnxFile)

		// Wenn Datei existiert und > 200MB ist (für E5), wollen wir die Xeon-Version erzwingen
		isLarge := false
		if info, err := os.Stat(destPath); err == nil {
			if info.Size() > 200*1024*1024 && m.Name == "multilingual-e5-small" && m.ModelFile == "" {
				isLarge = true
				fmt.Println("Gefunden: Standard-Modell (groß). Versuche Upgrade auf Xeon-optimierte Version (118MB)...")
			}
		}

		if _, err := os.Stat(destPath); err != nil || isLarge {
			var modelVariants []string
			if m.ModelFile != "" {
				modelVariants = []string{m.ModelFile}
			} else if m.Name == "multilingual-e5-small" {
				modelVariants = []string{
					"onnx/model_qint8_avx512_vnni.onnx",
					"onnx/model_O4.onnx",
					"onnx/model.onnx",
					"model.onnx",
				}
			} else {
				modelVariants = []string{"model_optimized.onnx", "model.onnx"}
			}

			success := false
			for _, variant := range modelVariants {
				url := fmt.Sprintf("%s/%s", baseURL, variant)
				// Temp-Pfad nutzen um Teil-Downloads bei Fehlern zu vermeiden
				tmpPath := destPath + ".tmp"
				if err := downloadFile(url, tmpPath, token); err == nil {
					os.Remove(destPath) // Altes Modell löschen
					os.Rename(tmpPath, destPath)
					fmt.Printf("OK: Modell geladen (%s)\n", variant)
					success = true
					break
				}
				os.Remove(tmpPath)
			}
			if !success && !isLarge {
				fmt.Printf("FEHLER: Konnte kein ONNX Modell für %s finden.\n", m.Name)
			}
		} else {
			fmt.Printf("OK: %s vorhanden.\n", localOnnxFile)
		}
	}
	fmt.Println("\nFertig.")
}
