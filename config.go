// Copyright 2026 Rob Deas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LauncherConfig holds settings read from sheafgate-launcher.env at build time
// via -ldflags, with sensible defaults for local dev builds.
type LauncherConfig struct {
	// Window
	WindowTitle  string
	WindowWidth  int
	WindowHeight int

	// Engine
	PreferredPort int
	PortRetries   int
	ReadyTimeout  int // seconds
}

// Build-time variables from sheafgate-launcher.env via -ldflags
var (
	CfgWindowTitle   = "" // defaults to AppName
	CfgWindowWidth   = "1200"
	CfgWindowHeight  = "800"
	CfgPreferredPort = "49200"
	CfgPortRetries   = "3"
	CfgReadyTimeout  = "15"
)

func loadConfig() LauncherConfig {
	title := CfgWindowTitle
	if title == "" {
		title = AppName
	}
	return LauncherConfig{
		WindowTitle:   title,
		WindowWidth:   parseInt(CfgWindowWidth, 1200),
		WindowHeight:  parseInt(CfgWindowHeight, 800),
		PreferredPort: parseInt(CfgPreferredPort, 5173),
		PortRetries:   parseInt(CfgPortRetries, 3),
		ReadyTimeout:  parseInt(CfgReadyTimeout, 15),
	}
}

func parseInt(s string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

// parseEnvFile parses KEY=VALUE format, ignoring comments and blank lines.
// Used by the Makefile to extract values — mirrored here for completeness.
func parseEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer f.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if key != "" {
			result[key] = val
		}
	}
	return result, scanner.Err()
}
