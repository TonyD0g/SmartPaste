# Smart Paste

**Smart Paste is a local-first smart clipboard.** It remembers everything you copy. When you focus an input field and hit one hotkey, it reads that field, picks the best match from your history, and pastes it for you — with *your own* TypeSafe key, at the price *your key* already pays.

**[Download](#download)** · [Why](#why-smart-paste) · [Quick start](#quick-start) · [Features](#features) · [Language](#language) · [Developers](#developers) · [Community](#community) · [简体中文](docs/README_zh.md)

[![Latest release](https://img.shields.io/github/v/release/OWNER/r-jaste?label=release&color=1a1816)](https://github.com/OWNER/r-jaste/releases/latest)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-1a1816)
![License](https://img.shields.io/badge/license-see%20License-1a1816)

## Why Smart Paste

Clipboard managers store what you copied. They still make you hunt for it.

Smart Paste keeps a local history of every text and image you copy, then reads the input field you are focused on and pastes the one entry that fits — no searching, no panel, no round trip. Everything stays on your disk, and the only network call is the Direct Jev request to TypeSafe with **your** API key: no cloud backend, no invite code, no server of ours in between.

## Features

### One hotkey, one paste

Press the hotkey in any input field. Smart Paste reads the focused field through the OS accessibility API, ranks your copy history, and writes the best match directly into the field. A suggestion is adopted only when confidence ≥ 0.4; otherwise the history panel opens so you can pick yourself.

- Swift version: **⌘V** (switchable to **⌘J**), intercepting the key like the original app
- Go version: **⌘⇧V** / **Ctrl+Shift+V** (⇧J variant), so plain paste is never taken over

### A history you can actually scan

A compact list pinned to three rows — mouse wheel snaps row by row, ↑/↓ select, Enter pastes, search filters, right-click or `Ctrl+Backspace` (⌘⌫ on macOS) deletes. Text and image entries alike, with thumbnails, relative timestamps, and the source app. Images are stored as PNG, capped at 4096 px on the long edge.

### Nothing leaves your machine except one request

History, images, and settings live in a plain folder on your disk. The one network call happens when you trigger Smart Paste: focused-field context plus candidate values go straight to `https://api.typesafe.ai/v1/systemone` with your own key. Card numbers, SSNs, private keys, and password / one-time-code fields are filtered out and never suggested.

### Any language you want

English and Simplified Chinese are built in, and the UI follows your system language on first launch. **Load Language File…** installs a JSON language pack immediately, so you can translate the app into your own language without rebuilding it.

## Download

Installers are published on the [Releases](https://github.com/OWNER/r-jaste/releases/latest) page. Pick the build for your system:

| System | Build | Download |
|---|---|---|
| macOS | Apple Silicon | `SmartPaste-mac-arm64.dmg` |
| macOS | Intel | `SmartPaste-mac-intel.dmg` |
| Windows | Windows 10 / 11 x64 | `SmartPaste-windows-setup.exe` |
| Linux | X11 / Wayland | `smartpaste` (AppImage when published) |

If a build for your system is not published yet, or you want the macOS version that intercepts **⌘V**, build from source — see [Developers](#developers). Both binaries are **ad-hoc signed**, not Developer ID notarized.

### First launch on macOS

Because the build is not notarized, macOS may block it. Drag **SmartPaste.app** into **Applications**, open it, and if macOS reports it cannot be opened, go to **System Settings → Privacy & Security**, scroll to **Security**, and click **Open Anyway**.

Then grant the two permissions the app asks for (System Settings → Privacy & Security):

- **Accessibility** — read the focused input field
- **Input Monitoring** — listen for the global hotkey

Ad-hoc signatures change on every recompile, so after rebuilding you will likely need to re-check both permissions. The app lives in the menu bar only; there is no Dock icon.

Only if macOS says the app is "damaged", confirm the download came from this repository, then run:

```bash
xattr -dr com.apple.quarantine "/Applications/SmartPaste.app"
```

Do not disable Gatekeeper globally.

### First launch on Windows

The build has no Authenticode signature. In the SmartScreen prompt, choose **More info** → **Run anyway**. No further permissions are required; if the target app runs as administrator, run Smart Paste as administrator too, otherwise it cannot read that app's input fields.

### First launch on Linux

Install at least one of `xdotool`, `wtype`, or `ydotool` — Smart Paste uses them to simulate the paste keystroke. On Wayland, global hotkeys may fail to register; use the tray icon to open the history panel instead. Enable accessibility in your desktop settings if focused-field reading returns nothing (AT-SPI2).

## Quick start

1. **Add your TypeSafe API key.** Click the menu bar / tray icon and open settings. The key is stored in your system credential store (Keychain / Credential Manager / Secret Service), or provide it via `SMARTPASTE_DIRECT_JEV_KEY`.
2. **Copy things as usual.** Text and images are captured automatically; pause capture from the tray menu if you need to.
3. **Focus an input field and press the hotkey.** A confident match is pasted on the spot; otherwise the panel opens — ↑/↓ then Enter, or Esc to dismiss.

More: [Go version guide](go/README.md) · [language packs](#language) · [data locations](#data--privacy)

## Language

Menu bar / tray → **Language / 语言**:

- Pick **English** or **简体中文**; first launch follows the system language.
- **Load Language File…** installs and switches to a JSON language pack immediately. **Open Languages Folder** reveals it:
  - macOS: `~/Library/Application Support/Smart Paste/Languages/`
  - Windows: `%APPDATA%\Smart Paste\Languages\`
  - Linux: `~/.local/share/Smart Paste/Languages/`
- `example.json` is generated there on first launch with every English string and writing notes. Copy it, change `code` and `name`, translate `strings`, then load it. Missing keys fall back to English; keep `%@` / `%ld` placeholders and their order.

```json
{
  "code": "fr",
  "name": "Français",
  "strings": {
    "menu.open": "Ouvrir Smart Paste (%@)",
    "timeline.search": "Rechercher dans l'historique"
  }
}
```

## Data & privacy

Default data folder — override it with `SMARTPASTE_DATA_DIRECTORY`:

```text
Smart Paste/
  clips.json       history (Swift and Go builds are compatible)
  settings.json    shortcut / language / capture pause / panel width
  Images/          image history (PNG)
  Languages/       language packs (example.json is auto-generated)
```

| Variable | Effect |
|---|---|
| `SMARTPASTE_DATA_DIRECTORY` | Override the data directory |
| `SMARTPASTE_DIRECT_JEV_KEY` | Provide the TypeSafe key directly, ahead of the credential store |
| `SMARTPASTE_KEYCHAIN_SERVICE` | Override the credential service name (default `app.smartpaste.mac.typesafe`) |
| `SMARTPASTE_TYPESAFE_ENDPOINT` | Override the Direct Jev endpoint (for proxying or testing) |

Clearing history in the panel deletes the local files. Nothing is synced anywhere, and no analytics are collected.

## Developers

**Swift version** — needs macOS 14+, Xcode Command Line Tools (Swift 6), no third-party dependencies:

```bash
git clone https://github.com/your-name/r-jaste.git
cd r-jaste
make app          # build/SmartPaste.app, ad-hoc signed
open build/SmartPaste.app
```

`make build` compiles only the binary. `Scripts/build-binary.sh` works around a Common Language Tools issue where `module.modulemap` and `bridging.modulemap` both declare `SwiftBridging`; a clean toolchain can also `swift build` via `Package.swift`.

**Go version** — needs Go 1.24+ and a C compiler (Fyne uses CGO, so cross-compiling a desktop binary from macOS is not possible; build on the target OS or in per-platform CI):

```bash
cd go && make app                                   # macOS
# Windows (PowerShell, needs mingw-w64)
$env:CGO_ENABLED = "1"; go build -trimpath -ldflags "-s -w" -o build\SmartPaste.exe .
# Linux (Debian/Ubuntu)
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o build/smartpaste .
```

```text
Sources/SmartPaste/   Swift app: menu bar, clipboard watcher, AX reader, suggestion client, panel, i18n
go/                   Go + Fyne rewrite: main.go + internal/{app,platform,store,suggest,i18n,ui,config,secret,model}
Resources/            Icon and menu bar template images from the original app
Scripts/              Build script
```

## License

mit
