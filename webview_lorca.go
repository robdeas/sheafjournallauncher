//go:build linux || darwin || (windows && lorca)

// Copyright 2026 Rob Deas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"

	"github.com/zserge/lorca"
)

// openWindow opens the app using lorca (Chrome/Chromium as the webview).
// Requires Chrome or Chromium to be installed on the system.
func openWindow(url string, cfg LauncherConfig) {
	if !isChromeAvailable() {
		showNoChromeError()
		return
	}

	ui, err := lorca.New(url, "", cfg.WindowWidth, cfg.WindowHeight)
	if err != nil {
		log.Printf("[sheaflauncher] lorca failed to open: %v", err)
		showNoChromeError()
		return
	}
	defer ui.Close()
	<-ui.Done()
}

func isChromeAvailable() bool {
	for _, candidate := range chromeCandidates() {
		if _, err := exec.LookPath(candidate); err == nil {
			return true
		}
	}
	return false
}

func chromeCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"chrome", "chromium", "msedge", "brave"}
	case "darwin":
		return []string{"chromium", "google-chrome", "brave-browser", "microsoft-edge"}
	default: // linux
		return []string{
			"chromium",
			"chromium-browser",
			"google-chrome",
			"google-chrome-stable",
			"brave-browser",
			"microsoft-edge",
			"vivaldi",
			"opera",
		}
	}
}

func showNoChromeError() {
	fmt.Println()
	fmt.Printf("❌  %s could not start — Chrome or Chromium is required on %s.\n", AppName, runtime.GOOS)
	fmt.Println()
	fmt.Println("Please install a Chromium-based browser:")
	fmt.Println(installInstructions())
	fmt.Println()
	fmt.Printf("Chrome, Chromium, Brave, Edge, Vivaldi, and Opera will all work.\n")
	fmt.Println("Safari is not supported.")
	fmt.Println()
}

func installInstructions() string {
	switch runtime.GOOS {
	case "windows":
		return "  https://www.chromium.org/getting-involved/download-chromium"
	case "darwin":
		return "  brew install chromium"
	default: // linux
		return `  Ubuntu/Debian:  sudo apt install chromium-browser
  Fedora:         sudo dnf install chromium
  Arch:           sudo pacman -S chromium`
	}
}
