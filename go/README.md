# Smart Paste · Go 跨平台重写

这是 `r-jaste` 的跨平台实现：同一份 Go 代码运行在 macOS、Windows、Linux。
原版是 Swift + AppKit，只能用于 macOS；本目录是新的 `go/` 版本。

## 先回答：Go 适合写这个项目吗？

**适合，但要把“语言”和“系统能力”分开看。**

Go 擅长的部分（大约 80%）：

- 剪贴板文本 / 图片监听与写回
- `clips.json` 历史持久化、图片压缩、索引值提取
- Direct Jev HTTP 请求与候选排序
- 系统托盘、菜单、多语言、配置
- 单二进制发布，内存占用低

真正难的部分（大约 20%）不是 Go 的问题，而是操作系统差异：

| 能力 | macOS | Windows | Linux |
| --- | --- | --- | --- |
| 剪贴板文本 / 图片 | ✅ 已实测 | ✅ 已实现，待真机验证 | ✅ 已实现，待真机验证（X11 / Wayland data-control） |
| 全局快捷键 | ✅ 已实测 `⌘⇧V` | ✅ `RegisterHotKey` 实现 | ✅ X11 实现；Wayland 取决于桌面 |
| 读取焦点输入框 | ✅ 已实测（AXUIElement） | ✅ UI Automation 实现，待真机验证 | ⚠️ AT-SPI2 实验性实现 |
| 模拟粘贴 | ✅ 已实测（CGEvent） | ✅ `keybd_event` | ✅ `xdotool` / `wtype` / `ydotool` 回退 |
| 系统托盘 | ✅ 已实测 | ✅ Fyne systray | ✅ StatusNotifier / Fyne systray |
| 密钥存储 | ✅ Keychain | ✅ Credential Manager | ✅ Secret Service，缺失时用环境变量 |

如果你最在意**像素级还原**，Go 原生 GUI 不如 Tauri/Electron；如果你最在意**单二进制、低资源、后端逻辑清晰**，Go + Fyne 是合适的。
Linux 的 Wayland 是最大不确定项：全局快捷键和焦点读取都受桌面环境限制，本实现会自动退化为“托盘 + 历史面板”。

## 与 Swift 版的行为差异

### 快捷键：`⌘⇧V` / `Ctrl+Shift+V`

原版直接拦截 `⌘V`，在无法智能粘贴时再把事件放行。要做到这一点，每个平台都需要底层键盘钩子：

- macOS：`CGEventTap`（原 Swift 版就是这么做的）
- Windows：`WH_KEYBOARD_LL`
- Linux X11：`XGrabKey` + 事件重放
- Linux Wayland：通常做不到

Go 版的全局快捷键使用系统热键注册，注册后会“吃掉”该组合键，因此不能直接注册 `⌘V` / `Ctrl+V`，否则普通粘贴会失效。
所以默认改为 `⌘⇧V`（macOS）/ `Ctrl+Shift+V`（Windows/Linux），并提供 `⇧J` 变体。这样普通粘贴始终可用。

> macOS 上切换快捷键后需要**重启应用**才会生效（热键库从非主线程卸载 CGEventTap 会死锁）；Windows/Linux 可即时切换。

如果后续必须完全复刻原版的 `⌘V` 体验，建议按平台补：

1. macOS 移植 Swift 版的 `CGEventTap` 逻辑；
2. Windows 加 `WH_KEYBOARD_LL`；
3. Linux 只在 X11 下做 `XGrabKey` + 重放，Wayland 保持独立快捷键。

### 焦点输入框

- macOS：读取 `AXTextField` / `AXTextArea` / `AXComboBox` / `AXSearchField`，过滤密码框和敏感字段。
- Windows：通过 UI Automation 读取 `Edit` / `Document` / `ComboBox`，使用 `IsPassword` 和关键字过滤。
- Linux：通过 AT-SPI2 在无障碍树中查找 `FOCUSED` 且可编辑的节点，节点预算 600、深度 9、1.2 秒超时。部分应用（未启用无障碍的 Electron、终端等）可能读不到，此时按 Esc 或托盘打开历史面板手动选择。
- 读不到焦点字段时，热键会打开历史面板，而不是直接粘贴。

### UI

Fyne 实现，已包含：

- 固定 3 行、滚轮按行吸附的纵向列表
- 搜索、上下键选择、回车粘贴、`Ctrl+Backspace`（macOS 也可 `⌘⌫`）删除、右键删除
- 文本 / 图片缩略图、相对时间、来源应用、图片尺寸
- 托盘菜单、欢迎页、权限页、密钥配置、语言切换

已知视觉差异：Fyne 窗口是普通的置顶窗口，带标题栏，不是原版的非激活无边框毛玻璃面板；文本预览为单行（原版两行）。功能不受影响，但并非逐像素复刻。

## 构建

要求：Go 1.24+（依赖要求）、C 编译器（Fyne 桌面端需要 CGO）。

> Fyne 使用 CGO，**不能**在 macOS 上直接交叉编译出 Windows/Linux 可用的桌面二进制。
> 请在目标系统上构建，或使用对应平台的 CI。

### macOS

```bash
cd go
make app          # 生成 build/SmartPaste.app（ad-hoc 签名）
open build/SmartPaste.app
```

首次运行需要授予：

- 辅助功能（Accessibility）：读取焦点输入框、注册全局快捷键
- 输入监控（Input Monitoring）：全局快捷键

`SmartPaste.app` 的 `Info.plist` 设置了 `LSUIElement=true`，因此只显示菜单栏图标，不显示 Dock 图标。

> 重新编译后 ad-hoc 签名会变化，macOS 可能要求在系统设置里重新勾选辅助功能 / 输入监控权限。
直接 `go run .` / `./build/SmartPaste` 也可以运行，但会以普通应用形式出现 Dock 图标。

只编译二进制：

```bash
make build        # build/SmartPaste
```

### Windows

安装 Go 和 mingw-w64（提供 gcc），然后在 PowerShell / cmd 中：

```powershell
cd go
$env:CGO_ENABLED = "1"
go build -trimpath -ldflags "-s -w" -o build\SmartPaste.exe .
.\build\SmartPaste.exe
```

可选用 Fyne CLI 打包带图标的 exe（先用一张 256×256 PNG 作为图标）：

```powershell
go install fyne.io/tools/cmd/fyne@latest
fyne package -os windows -icon appicon.png   # 准备一张 256×256 PNG
```

Windows 无需额外权限。若目标应用以管理员身份运行，Smart Paste 可能读不到它的输入框；此时请同样以管理员身份运行 Smart Paste。

### Linux

Ubuntu/Debian 示例依赖：

```bash
sudo apt install build-essential pkg-config \
  libgl1-mesa-dev xorg-dev libxkbcommon-dev \
  libayatana-appindicator3-dev
```

构建：

```bash
cd go
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o build/smartpaste .
./build/smartpaste
```

运行时：

- X11：全局快捷键可用；焦点读取通过 AT-SPI2，若桌面未启用无障碍，请在系统设置中打开。
- Wayland：剪贴板通过 data-control 工作；全局快捷键可能注册失败，请用托盘图标。
- 模拟粘贴会依次尝试 `xdotool`、`wtype`、`ydotool`，至少安装一个。
- 托盘依赖 StatusNotifier / AppIndicator；极简窗口管理器可能不显示托盘。

## 数据目录与环境变量

默认数据目录：

| 系统 | 目录 |
| --- | --- |
| macOS | `~/Library/Application Support/Smart Paste` |
| Windows | `%APPDATA%\Smart Paste` |
| Linux | `$XDG_DATA_HOME/Smart Paste` 或 `~/.local/share/Smart Paste` |

文件：

```
Smart Paste/
  clips.json       历史记录（与 Swift 版兼容）
  settings.json    快捷键 / 语言 / 捕获暂停 / 窗口宽度
  Images/          图片历史（PNG）
  Languages/       语言包（example.json 会自动生成）
```

环境变量：

| 变量 | 作用 |
| --- | --- |
| `SMARTPASTE_DATA_DIRECTORY` | 覆盖数据目录（与 Swift 版同名） |
| `SMARTPASTE_DIRECT_JEV_KEY` | 直接提供 TypeSafe API Key，优先于系统凭据存储 |
| `SMARTPASTE_KEYCHAIN_SERVICE` | 覆盖凭据存储的 service 名（默认 `app.smartpaste.mac.typesafe`（与 Swift 版一致，便于复用已保存的密钥）） |
| `SMARTPASTE_TYPESAFE_ENDPOINT` | 覆盖 Direct Jev 地址（默认 `https://api.typesafe.ai/v1/systemone`，便于代理/测试） |

### 与 Swift 版的数据兼容

- `clips.json` 的 `createdAt` 按 Foundation 参考时间（2001-01-01 起的秒数）写入，Swift 版可以直接读取。
- 读取时同时接受数字和 RFC3339 字符串。
- 图片仍为 `Images/<uuid>.png`。
- 语言包格式不变；新增的界面文案键（如 `setup.*`）在旧语言包中缺失时会回退到英文。

## 代码结构

```
go/
  main.go                          入口、日志、单实例、信号处理
  internal/
    app/controller.go              热键触发、焦点轮询、建议状态机、粘贴/剪贴板恢复
    config/                        跨平台路径、settings.json
    i18n/                          内置中英文、语言包加载/切换/校验
    model/                         Clip / Field / Candidate / Outcome
    platform/
      clipboard.go                 剪贴板监听与写回（golang.design/x/clipboard）
      hotkey.go                    全局快捷键管理
      platform.go                  焦点/前台应用/粘贴/权限抽象
      platform_darwin.go           AXUIElement + CGEvent + CGWindowList
      platform_windows.go          UI Automation（COM vtable）+ keybd_event
      platform_linux.go            AT-SPI2（D-Bus）+ xdotool/wtype/ydotool
    secret/                        系统凭据存储（zalando/go-keyring）
    store/                         历史持久化、图片压缩、敏感过滤、索引值
    suggest/                       Direct Jev 客户端与候选排序
    ui/                            Fyne 托盘、面板、列表、对话框、主题
```

## 安全与隐私

- 历史只保存在本机数据目录。
- 只有触发智能粘贴时，才会把焦点字段信息和候选值发送到 TypeSafe；没有自有服务器。
- 以下内容不会进入历史或发送：
  - 银行卡号（去掉空格/短横线后 13–19 位数字）
  - 美国 SSN
  - 私钥（`-----BEGIN ... PRIVATE KEY-----`）
  - 形如 `password: xxx` / `密码: xxx` 的凭据文本
  - 剪贴板库标记为 `Sensitive` 的内容
  - 密码框 / 安全输入框
- API Key 默认写入系统凭据存储，不写入明文文件；系统凭据存储不可用时请使用 `SMARTPASTE_DIRECT_JEV_KEY`。

## 测试

```bash
cd go
go test ./...
```

覆盖：

- 历史存储、敏感过滤、索引值、图片缩放、Swift 日期格式兼容
- Direct Jev 请求/响应契约、候选去重与排序
- 语言包占位符、外部语言包安装与校验
- 设置读写
- Fyne 软件渲染（3 行面板、中文字体、图片缩略图、搜索过滤、选择）

## 已知限制 / 后续可做

1. 未实现原版的 `⌘V` / `Ctrl+V` 底层拦截（见上文，默认使用 `⌘⇧V` / `Ctrl+Shift+V`）。
2. Windows UI Automation 与 Linux AT-SPI2 代码已通过交叉编译检查，但尚未在真实 Windows/Linux 桌面做运行验证。
3. Linux Wayland 下全局快捷键和焦点读取受桌面环境限制。
4. 剪贴板恢复只覆盖文本和 PNG 图片，不保留富文本、文件等其它格式。
5. 面板不是无边框非激活窗口；打开面板会抢焦点，粘贴前会尝试恢复之前的前台应用。
6. 同一应用内两个 **标签 / 角色 / 应用名完全相同** 的输入框会被当作同一个字段（原版用 AX 元素身份区分）；如果遇到这种情况，可以在 `fieldKey` 中加入窗口位置或平台元素 ID。
7. 未实现开机自启、多显示器精确定位、图片预览大图。
