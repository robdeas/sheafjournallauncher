#  Copyright 2026 Rob Deas
#
#  Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0`
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

APP_NAME      := sheafjournal
LAUNCHER_ENV  := sheafgate-launcher.env
ENGINE_ENV    := sheafgate-launcher-engine.env
BUN_APP       := /c/work/typescript/sheafjournal
BUN_URL       := https://github.com/oven-sh/bun/releases/latest/download/bun-windows-x64.zip
BUN_URL_LINUX := https://github.com/oven-sh/bun/releases/latest/download/bun-linux-x64.zip

# ── Dependency checks ─────────────────────────────────────────────────────────
BUN_VERSION := $(shell bun --version 2>/dev/null)
GO_VERSION  := $(shell go version 2>/dev/null)

$(if $(BUN_VERSION),,$(error bun is not installed or not on PATH — see https://bun.sh))
$(if $(GO_VERSION),,$(error go is not installed or not on PATH — see https://golang.org))

# ── Generate env files if missing (runs at parse time) ────────────────────────
$(shell [ -f $(LAUNCHER_ENV) ] || bun -e "const uuid=require('crypto').randomUUID();const lines=['# SheafBase launcher configuration','# Change SHEAFGATE_LAUNCHER_UUID when launcher/engine protocol changes incompatibly','# Both sides must match or auth will fail','','SHEAFGATE_LAUNCHER_UUID='+uuid,'','# Window settings','SHEAFGATE_LAUNCHER_TITLE=$(APP_NAME)','SHEAFGATE_LAUNCHER_WIDTH=1200','SHEAFGATE_LAUNCHER_HEIGHT=800','','# Engine settings','SHEAFGATE_LAUNCHER_PORT_RETRIES=3','SHEAFGATE_LAUNCHER_READY_TIMEOUT=15'];require('fs').writeFileSync('$(LAUNCHER_ENV)',lines.join('\n')+'\n');console.error('[sheafgatelauncher] generated $(LAUNCHER_ENV) uuid='+uuid);")

$(shell [ -f $(ENGINE_ENV) ] || bun -e "const fs=require('fs');const e=fs.readFileSync('$(LAUNCHER_ENV)','utf8');const lines=e.split('\n');const uuidLine=lines.find(l=>l.startsWith('SHEAFGATE_LAUNCHER_UUID='));const uuid=uuidLine?uuidLine.split('=')[1].trim():require('crypto').randomUUID();const out=['# SheafBase engine configuration','# SHEAFGATE_LAUNCHER_UUID must match the Go launcher','# Change when launcher/engine protocol changes incompatibly','# Both sides must match or auth will fail','','SHEAFGATE_LAUNCHER_UUID='+uuid];fs.writeFileSync('$(ENGINE_ENV)',out.join('\n')+'\n');console.error('[sheafgatelauncher] generated $(ENGINE_ENV)');")

# ── Read values from sheafgate-launcher.env ──────────────────────────────────────
getenv = $(shell bun -e "try{const f=require('fs').readFileSync('$(LAUNCHER_ENV)','utf8');const lines=f.split('\n');const l=lines.find(l=>l.startsWith('$1='));if(l)console.log(l.split('=')[1].trim());}catch(e){}" 2>/dev/null)

UUID    := $(call getenv,SHEAFGATE_LAUNCHER_UUID)
TITLE   := $(call getenv,SHEAFGATE_LAUNCHER_TITLE)
WIDTH   := $(call getenv,SHEAFGATE_LAUNCHER_WIDTH)
HEIGHT  := $(call getenv,SHEAFGATE_LAUNCHER_HEIGHT)
RETRIES := $(call getenv,SHEAFGATE_LAUNCHER_PORT_RETRIES)
TIMEOUT := $(call getenv,SHEAFGATE_LAUNCHER_READY_TIMEOUT)

# ── Build metadata ────────────────────────────────────────────────────────────
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell bun -e "console.log(new Date().toISOString())")

# ── Build info header ─────────────────────────────────────────────────────────
$(info )
$(info [sheafgate] build environment)
$(info   app:    $(APP_NAME))
$(info   go:     $(GO_VERSION))
$(info   bun:    $(BUN_VERSION))
$(info   ver:    $(VERSION))
$(info   commit: $(COMMIT))
$(info   date:   $(DATE))
$(info   uuid:   $(UUID))
$(info )

# ── Targets ───────────────────────────────────────────────────────────────────
.PHONY: build build-launcher build-engine sync-env dist update-uuid clean help

## Full release build — builds both launcher and engine
build: build-launcher build-engine

## Build the Go launcher only — no Bun app needed
build-launcher:
	go build -ldflags "-X main.AppName=$(APP_NAME) -X main.LauncherUUID=$(UUID) -X main.Version=$(VERSION) -X main.BuildTime=$(DATE) -X main.GitCommit=$(COMMIT) -X main.CfgWindowTitle=$(TITLE) -X main.CfgWindowWidth=$(WIDTH) -X main.CfgWindowHeight=$(HEIGHT) -X main.CfgPortRetries=$(RETRIES) -X main.CfgReadyTimeout=$(TIMEOUT)" -o $(APP_NAME).exe .

## Build the Bun engine only
build-engine: sync-env
	cd $(BUN_APP) && bun run build

## Sync engine env file to Bun app source tree
sync-env:
	@bun -e "require('fs').copyFileSync('$(ENGINE_ENV)', '$(BUN_APP)/$(ENGINE_ENV)')"
	@echo "[sheafgatelauncher] synced $(ENGINE_ENV) to $(BUN_APP)"

## Assemble full distribution folder — builds everything and downloads bun.exe if needed
dist: build
	@echo "[sheafgatelauncher] assembling dist..."
	@mkdir -p dist
	@cp -r $(BUN_APP)/build/. dist/build/
	@echo "[sheafgatelauncher] copied engine build from $(BUN_APP)/build"
	@cp $(APP_NAME).exe dist/
	@echo "[sheafgatelauncher] copied launcher"
	@if [ ! -f dist/bun.exe ]; then \
        echo "[sheafgatelauncher] downloading bun.exe..."; \
        bun -e "const r=await fetch('$(BUN_URL)');await Bun.write('bun.zip',await r.blob());"; \
        unzip -p bun.zip 'bun-windows-x64/bun.exe' > dist/bun.exe; \
        rm -f bun.zip; \
        echo "[sheafgatelauncher] bun.exe downloaded"; \
    else \
        echo "[sheafgatelauncher] bun.exe already present, skipping download"; \
    fi
	@echo "[sheafgatelauncher] dist ready:"
	@echo "  dist/$(APP_NAME).exe"
	@echo "  dist/bun.exe"
	@echo "  dist/build/"

BUN_URL_LINUX := https://github.com/oven-sh/bun/releases/latest/download/bun-linux-x64.zipB

dist-linux: build
	@echo "[sheafgatelauncher] assembling dist-linux..."
	@mkdir -p dist-linux
	@cp -r $(BUN_APP)/build/. dist-linux/build/
	@go build -GOOS=linux GOARCH=amd64 -ldflags "..." -o dist-linux/sheafjournal .
	@if [ ! -f dist-linux/bun ]; then \
		bun -e " \
			const r = await fetch('$(BUN_URL_LINUX)'); \
			const buf = await r.arrayBuffer(); \
			const {unzipSync} = await import('npm:fflate'); \
			const files = unzipSync(new Uint8Array(buf)); \
			require('fs').writeFileSync('dist-linux/bun', Buffer.from(files['bun-linux-x64/bun'])); \
			require('fs').chmodSync('dist-linux/bun', 0o755); \
		"; \
	fi
	@echo "[sheafgatelauncher] dist-linux ready"

## Rotate UUID in both env files (use when launcher/engine protocol changes)
update-uuid:
	@bun -e "const fs=require('fs');const uuid=require('crypto').randomUUID();['$(LAUNCHER_ENV)','$(ENGINE_ENV)'].forEach(file=>{const lines=fs.readFileSync(file,'utf8').split('\n');const updated=lines.map(l=>l.startsWith('SHEAFGATE_LAUNCHER_UUID=')?'SHEAFGATE_LAUNCHER_UUID='+uuid:l);fs.writeFileSync(file,updated.join('\n'));});console.log('[sheafgatelauncher] rotated UUID to: '+uuid);console.log('[sheafgatelauncher] remember to commit both env files and rebuild both sides');"

## First-time setup: create engine env file to copy to your Bun project
## After this, use update-uuid to keep both files in sync
make-engine-env:
	@bun -e "const fs=require('fs');const e=fs.readFileSync('$(LAUNCHER_ENV)','utf8');const lines=e.split('\n');const uuidLine=lines.find(l=>l.startsWith('SHEAFGATE_LAUNCHER_UUID='));const uuid=uuidLine?uuidLine.split('=')[1].trim():require('crypto').randomUUID();const out=['# SheafBase engine configuration','# SHEAFGATE_LAUNCHER_UUID must match the Go launcher','# Change when launcher/engine protocol changes incompatibly','# Both sides must match or auth will fail','','SHEAFGATE_LAUNCHER_UUID='+uuid];fs.writeFileSync('$(ENGINE_ENV)',out.join('\n')+'\n');console.log('[sheafgatelauncher] created $(ENGINE_ENV) — copy this to your Bun project root');"

## Remove build artifacts and dist folder
clean:
	@bun -e "const fs=require('fs');try{fs.unlinkSync('$(APP_NAME).exe')}catch(e){};try{fs.rmSync('dist',{recursive:true})}catch(e){};"
	@echo "[sheafgatelauncher] cleaned"

## Show this help
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@bun -e "const f=require('fs').readFileSync('Makefile','utf8');f.split('\n').filter(l=>l.startsWith('## ')).forEach(l=>console.log('  '+l.slice(3)));"
	@echo ""