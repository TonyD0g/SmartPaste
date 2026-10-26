# Smart Paste

**Smart Paste 是一款本地优先的智能剪贴板。** 它记住你复制过的每一条内容。聚焦输入框、按下一个快捷键，它就读懂这个输入框、从历史里挑出最合适的一条直接粘贴 —— 用的是*你自己的* TypeSafe Key，走的也是*你自己的*额度。

**[下载](#下载)** · [为什么](#为什么是-smart-paste) · [快速上手](#快速上手) · [功能](#功能) · [语言](#语言) · [开发者](#开发者) · [社区](#社区) · [English](../README.md)

[![最新版本](https://img.shields.io/github/v/release/OWNER/r-jaste?label=release&color=1a1816)](https://github.com/OWNER/r-jaste/releases/latest)
![平台](https://img.shields.io/badge/平台-macOS%20%7C%20Windows%20%7C%20Linux-1a1816)
![许可](https://img.shields.io/badge/许可-见许可章节-1a1816)

## 为什么是 Smart Paste

剪贴板管理工具确实存下了你复制的内容，但你还是得自己去翻。

Smart Paste 在本地记录你复制的每一条文本和图片，然后读取你当前聚焦的输入框，直接粘贴匹配的那一条 —— 不用搜索、不用开面板、不用来回切换。所有数据都在你自己的磁盘上，唯一的网络请求是用**你自己的** API Key 直连 TypeSafe 的 Direct Jev：没有云端后端、没有邀请码、中间不经过我们的任何服务器。

## 功能

### 一个快捷键，一次粘贴

在任意输入框按下快捷键，Smart Paste 通过系统无障碍接口读取焦点输入框，对复制历史排序，并把最合适的一条直接写入。建议只在置信度 ≥ 0.4 时采纳；否则打开历史面板让你自己选。

- Swift 版：**⌘V**（可切换为 **⌘J**），与原版一样拦截该按键
- Go 版：**⌘⇧V** / **Ctrl+Shift+V**（另有 ⇧J 变体），不会占用普通粘贴

### 一眼扫得完的历史

固定在 3 行的紧凑列表：滚轮按行吸附，↑/↓ 选择，回车粘贴，搜索过滤，右键或 `Ctrl+Backspace`（macOS 也可 `⌘⌫`）删除。文本与图片条目都有缩略图、相对时间与来源应用；图片以 PNG 存储，长边不超过 4096 像素。

### 除了一个请求，什么都不离开本机

历史、图片、设置都是磁盘上的普通文件夹。唯一的网络请求发生在触发智能粘贴的那一刻：焦点字段信息与候选值直接发往 `https://api.typesafe.ai/v1/systemone`，使用你自己的 Key。银行卡号、SSN、私钥以及密码 / 验证码类字段会被过滤，永不出现在建议里。

### 任意语言

内置 English 与简体中文，首次启动跟随系统语言。**加载语言文件…** 可立即安装并切换 JSON 语言包，无需重新编译就能把界面换成你自己的语言。

## 下载

安装包发布在 [Releases](https://github.com/OWNER/r-jaste/releases/latest) 页面，请按系统选择：

| 系统 | 架构 | 下载 |
|---|---|---|
| macOS | Apple Silicon | `SmartPaste-mac-arm64.dmg` |
| macOS | Intel | `SmartPaste-mac-intel.dmg` |
| Windows | Windows 10 / 11 x64 | `SmartPaste-windows-setup.exe` |
| Linux | X11 / Wayland | `smartpaste`（发布 AppImage 后提供） |

如果你的系统还没有对应安装包，或者你想要 macOS 上拦截 **⌘V** 的原版体验，请自行构建 —— 见[开发者](#开发者)。两种二进制均为 **ad-hoc 签名**，未经 Developer ID 公证。

### macOS 首次启动

由于未经公证，macOS 可能拦截。把 **SmartPaste.app** 拖入**应用程序**后打开；若提示无法打开，请到**系统设置 → 隐私与安全性**，滚动到**安全性**，点击该提示旁的**仍要打开**。

随后授予应用请求的两项权限（系统设置 → 隐私与安全性）：

- **辅助功能（Accessibility）**：读取当前聚焦的输入框
- **输入监控（Input Monitoring）**：监听全局快捷键

ad-hoc 签名每次重新编译都会变化，因此重新构建后通常需要重新勾选这两项权限。应用只驻留菜单栏，不显示 Dock 图标。

仅当 macOS 提示应用"已损坏"时：先确认安装包来自本仓库，再执行

```bash
xattr -dr com.apple.quarantine "/Applications/SmartPaste.app"
```

请勿全局关闭 Gatekeeper。

### Windows 首次启动

安装包没有 Authenticode 签名。在 SmartScreen 提示中选择**更多信息** → **仍要运行**。无需其他权限；若目标应用以管理员身份运行，请同样以管理员身份运行 Smart Paste，否则读不到它的输入框。

### Linux 首次启动

至少安装 `xdotool`、`wtype`、`ydotool` 之一，Smart Paste 用它们模拟粘贴按键。Wayland 下全局快捷键可能注册失败，可改用托盘图标打开历史面板。若读不到焦点输入框，请在系统设置中开启无障碍（AT-SPI2）。

## 快速上手

1. **填入 TypeSafe API Key。** 点击菜单栏 / 托盘图标进入设置。密钥保存在系统凭据存储（Keychain / 凭据管理器 / Secret Service），也可用环境变量 `SMARTPASTE_DIRECT_JEV_KEY` 直接提供。
2. **照常复制。** 文本和图片会被自动记录；需要时可在托盘菜单暂停捕获。
3. **聚焦输入框并按快捷键。** 有可信匹配时立即粘贴；否则打开面板 —— ↑/↓ 后回车选择，按 Esc 关闭。

更多：[Go 版说明](../go/README.md) · [语言包](#语言) · [数据位置](#数据与隐私)

## 语言

菜单栏 / 托盘 → **Language / 语言**：

- 选择 **English** 或 **简体中文**；首次启动跟随系统语言。
- **加载语言文件…** 立即安装并切换 JSON 语言包；**打开语言文件夹**可直接进入该目录：
  - macOS：`~/Library/Application Support/Smart Paste/Languages/`
  - Windows：`%APPDATA%\Smart Paste\Languages\`
  - Linux：`~/.local/share/Smart Paste/Languages/`
- 首次启动会在该目录生成 `example.json`，内含全部英文原文与写作说明：复制它，修改 `code` 与 `name`，翻译 `strings`，再加载即可。缺失的键自动回退英文；请保留 `%@` / `%ld` 占位符及其顺序。

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

## 数据与隐私

默认数据目录（可用 `SMARTPASTE_DATA_DIRECTORY` 覆盖）：

```text
Smart Paste/
  clips.json       历史记录（Swift 版与 Go 版互相兼容）
  settings.json    快捷键 / 语言 / 捕获暂停 / 面板宽度
  Images/          图片历史（PNG）
  Languages/       语言包（自动生成 example.json）
```

| 环境变量 | 作用 |
|---|---|
| `SMARTPASTE_DATA_DIRECTORY` | 覆盖数据目录 |
| `SMARTPASTE_DIRECT_JEV_KEY` | 直接提供 TypeSafe Key，优先于系统凭据存储 |
| `SMARTPASTE_KEYCHAIN_SERVICE` | 覆盖凭据存储的 service 名（默认 `app.smartpaste.mac.typesafe`） |
| `SMARTPASTE_TYPESAFE_ENDPOINT` | 覆盖 Direct Jev 地址（便于代理或测试） |

在面板中清空历史即删除本地文件；没有任何云同步，也不采集任何统计数据。

## 开发者

**Swift 版** —— 需要 macOS 14+、Xcode Command Line Tools（Swift 6），无第三方依赖：

```bash
git clone https://github.com/your-name/r-jaste.git
cd r-jaste
make app          # 生成 build/SmartPaste.app（ad-hoc 签名）
open build/SmartPaste.app
```

`make build` 只编译二进制。`Scripts/build-binary.sh` 会处理一种 Command Line Tools 问题：`module.modulemap` 与 `bridging.modulemap` 同时声明 `SwiftBridging` 导致模块重复定义；工具链正常的机器也可直接用 `Package.swift` 执行 `swift build`。

**Go 版** —— 需要 Go 1.24+ 和 C 编译器（Fyne 依赖 CGO，无法在 macOS 上交叉编译出 Windows/Linux 桌面二进制，请在目标系统或对应平台 CI 上构建）：

```bash
cd go && make app                                   # macOS
# Windows（PowerShell，需 mingw-w64）
$env:CGO_ENABLED = "1"; go build -trimpath -ldflags "-s -w" -o build\SmartPaste.exe .
# Linux（Debian/Ubuntu）
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o build/smartpaste .
```

```text
Sources/SmartPaste/   Swift 源码：菜单栏、剪贴板监听、AX 焦点读取、建议客户端、面板、多语言
go/                   Go + Fyne 重写：main.go 与 internal/{app,platform,store,suggest,i18n,ui,config,secret,model}
Resources/            来自原始应用的图标与菜单栏模板图
Scripts/              构建脚本
```

## 许可

mit
