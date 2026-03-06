//go:build windows && !lorca

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
	"log"

	webview2 "github.com/jchv/go-webview2"
)

// openWindow opens the app using WebView2 (built into Windows 10/11).
// No extra dependencies required.
func openWindow(url string, cfg LauncherConfig) {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  cfg.WindowTitle,
			Width:  uint(cfg.WindowWidth),
			Height: uint(cfg.WindowHeight),
		},
	})
	if w == nil {
		log.Fatal("[sheaflauncher] failed to create WebView2 window — is WebView2 runtime installed?")
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run()
}
