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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ── Build-time variables (set via -ldflags) ───────────────────────────────────

var (
	AppName      = "sheafbase" // overridden at build time
	LauncherUUID = ""          // must be set at build time — fails fast if missing
	Version      = "dev"
	BuildTime    = "unknown"
	GitCommit    = "unknown"
)

// ── Errors ────────────────────────────────────────────────────────────────────

var ErrPortUnavailable = errors.New("port unavailable")

// ── Ready signal (matches emitReadySignal() in serverInit.ts) ─────────────────

type readySignal struct {
	Status string `json:"status"`
	Port   int    `json:"port"`
}

// ── Engine state ──────────────────────────────────────────────────────────────

var engineCmd *exec.Cmd
var enginePort int

// ── Entry point ───────────────────────────────────────────────────────────────

func main() {
	if LauncherUUID == "" {
		log.Fatal("[sheaflauncher] LauncherUUID not set — build with -ldflags -X main.LauncherUUID=...")
	}

	cfg := loadConfig()

	log.Printf("[sheaflauncher] %s %s (%s) built %s", AppName, Version, GitCommit, BuildTime)

	password := resolvePassword()

	port, err := launchBunWithRetry(password, cfg)
	if err != nil {
		log.Fatalf("[sheaflauncher] failed to start engine: %v", err)
	}
	enginePort = port

	defer shutdownEngine()

	url := fmt.Sprintf(
		"http://127.0.0.1:%d/sheaflauncher-control?uuid=%s&password=%s",
		port, LauncherUUID, password,
	)

	log.Printf("[sheaflauncher] engine ready on port %d, opening webview", port)

	openWindow(url, cfg)
}

// ── Shutdown ──────────────────────────────────────────────────────────────────

func shutdownEngine() {
	if engineCmd == nil || engineCmd.Process == nil {
		return
	}
	log.Printf("[sheaflauncher] sending shutdown signal to engine")
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf(
		"http://127.0.0.1:%d/sheaflauncher-control?uuid=%s",
		enginePort, LauncherUUID,
	)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err == nil {
		client.Do(req) //nolint:errcheck
	}
	// Give engine a moment to shut down gracefully before force killing
	time.Sleep(200 * time.Millisecond)
	if err := engineCmd.Process.Kill(); err == nil {
		log.Printf("[sheaflauncher] engine process killed")
	}
}

// ── Password ──────────────────────────────────────────────────────────────────

func resolvePassword() string {
	pw := uuid.New().String()
	log.Printf("[sheaflauncher] generated session password")
	return pw
}

// ── Engine paths ──────────────────────────────────────────────────────────────

func launcherDir() string {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("[sheaflauncher] could not determine executable path: %v", err)
	}
	return filepath.Dir(exe)
}

func bunExePath() string {
	name := "bun"
	if runtime.GOOS == "windows" {
		name = "bun.exe"
	}
	return filepath.Join(launcherDir(), name)
}

func engineScriptPath() string {
	return filepath.Join(launcherDir(), "build", "index.js")
}

// ── Port helpers ──────────────────────────────────────────────────────────────

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

func findFreePort(preferred int) (int, error) {
	if isPortFree(preferred) {
		return preferred, nil
	}
	for port := preferred + 1; port < preferred+20; port++ {
		if isPortFree(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found near %d", preferred)
}

// ── Engine health check ───────────────────────────────────────────────────────

func checkEngineAlive(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf(
		"http://127.0.0.1:%d/sheaflauncher-control?uuid=%s",
		port, LauncherUUID,
	))
	if err != nil {
		return false
	}
	err = resp.Body.Close()
	if err != nil {
		return false
	}
	// 404 means wrong UUID or wrong app — not our engine
	return resp.StatusCode != 404
}

// ── Launch ────────────────────────────────────────────────────────────────────

func launchBun(password string, port int, cfg LauncherConfig) (int, error) {
	bunPath := bunExePath()
	scriptPath := engineScriptPath()

	if _, err := os.Stat(bunPath); err != nil {
		return 0, fmt.Errorf("bun not found at %s: %w", bunPath, err)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		return 0, fmt.Errorf("engine script not found at %s: %w", scriptPath, err)
	}

	cmd := exec.Command(bunPath, scriptPath)
	env := os.Environ()
	env = setEnvVar(env, "SHEAF_LAUNCHER_PASSWORD", password)
	env = setEnvVar(env, "PORT", fmt.Sprintf("%d", port))
	env = setEnvVar(env, "HOST", "127.0.0.1")
	cmd.Env = env

	for _, e := range cmd.Env {
		if strings.HasPrefix(e, "PORT=") ||
			strings.HasPrefix(e, "HOST=") ||
			strings.HasPrefix(e, "SHEAF_") {
			log.Printf("[sheaflauncher] env: %s", e)
		}
	}
	log.Printf("[sheaflauncher] running: %s %s", bunPath, scriptPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("stdout pipe failed: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("stderr pipe failed: %w", err)
	}

	if err := cmd.Start(); err != nil {
		if isPortError(err) {
			return 0, ErrPortUnavailable
		}
		return 0, fmt.Errorf("failed to start engine: %w", err)
	}

	// Store for graceful shutdown
	engineCmd = cmd

	ready := make(chan readySignal, 1)
	startErr := make(chan error, 1)

	// Scan stdout for ready signal
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			var sig readySignal
			if err := json.Unmarshal([]byte(line), &sig); err == nil && sig.Status == "ready" {
				ready <- sig
				return
			}
			fmt.Println(line)
		}
		startErr <- fmt.Errorf("engine stdout closed without ready signal")
	}()

	// Scan stderr — pass through to console but watch for port errors
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Fprintln(os.Stderr, line)
			if err != nil {
				return
			}
			if isPortError(fmt.Errorf("%s", line)) {
				startErr <- ErrPortUnavailable
				return
			}
		}
	}()

	timeout := time.Duration(cfg.ReadyTimeout) * time.Second

	select {
	case sig := <-ready:
		return sig.Port, nil
	case err := <-startErr:
		err = cmd.Process.Kill()
		if err != nil {
			return 0, err
		}
		if isPortError(err) {
			return 0, ErrPortUnavailable
		}
		return 0, err
	case <-time.After(timeout):
		// Before giving up check if engine is actually running on expected port
		log.Printf("[sheaflauncher] ready signal timeout — checking port %d directly", port)
		if checkEngineAlive(port) {
			log.Printf("[sheaflauncher] engine alive on port %d (no ready signal)", port)
			return port, nil
		}
		// Also scan nearby ports in case adapter-node shifted
		for p := port + 1; p < port+20; p++ {
			if checkEngineAlive(p) {
				log.Printf("[sheaflauncher] engine found on port %d (shifted from %d)", p, port)
				return p, nil
			}
		}
		err := cmd.Process.Kill()
		if err != nil {
			return 0, err
		}
		return 0, ErrPortUnavailable
	}
}

func launchBunWithRetry(password string, cfg LauncherConfig) (int, error) {
	for attempt := range cfg.PortRetries {
		port, err := findFreePort(cfg.PreferredPort)
		if err != nil {
			return 0, err
		}

		actualPort, err := launchBun(password, port, cfg)
		if err == nil {
			return actualPort, nil
		}

		if !errors.Is(err, ErrPortUnavailable) {
			return 0, err
		}

		log.Printf("[sheaflauncher] port busy, retry %d/%d", attempt+1, cfg.PortRetries)
	}

	return 0, fmt.Errorf("could not bind to a port after %d attempts", cfg.PortRetries)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func isPortError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "bind") ||
		strings.Contains(msg, "is port") ||
		strings.Contains(msg, "eaddrinuse")
}

// setEnvVar sets or replaces an environment variable in a slice.
// Prevents duplicate keys which can cause the wrong value to be used.
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
