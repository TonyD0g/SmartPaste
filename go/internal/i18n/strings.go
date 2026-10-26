package i18n

func englishPack() Pack {
	return Pack{Code: "en", Name: "English", Strings: englishStrings, BuiltIn: true}
}

func chinesePack() Pack {
	return Pack{Code: chineseCode, Name: "简体中文", Strings: chineseStrings, BuiltIn: true}
}

var englishStrings = map[string]string{
	"app.name":    "Smart Paste",
	"app.tagline": "Paste the right value, anywhere.",

	"menu.open":                "Open Smart Paste (%@)",
	"menu.pauseCapture":        "Pause Clipboard Capture",
	"menu.resumeCapture":       "Resume Clipboard Capture",
	"menu.clearHistory":        "Clear All History",
	"menu.key.configure":       "Configure Direct Jev Key…",
	"menu.key.update":          "Update Direct Jev Key…",
	"menu.key.remove":          "Remove Direct Jev Key",
	"menu.shortcut":            "Keyboard Shortcut (%@)",
	"menu.shortcut.commandV":   "Command+V",
	"menu.shortcut.commandJ":   "Command+J",
	"menu.shortcut.useV":       "Use %@",
	"menu.shortcut.useJ":       "Use %@",
	"menu.shortcut.hint":       "Smart Paste uses a dedicated global shortcut so normal paste keeps working.",
	"menu.language":            "Language",
	"menu.language.load":       "Load Language File…",
	"menu.language.openFolder": "Open Languages Folder",
	"menu.setup":               "Setup & Permissions…",
	"menu.quit":                "Quit Smart Paste",

	"alert.ok":                  "OK",
	"alert.cancel":              "Cancel",
	"alert.clear.title":         "Clear all clipboard history?",
	"alert.clear.message.one":   "This permanently removes the saved clipboard item, including any image. Your current clipboard will stay unchanged.",
	"alert.clear.message.other": "This permanently removes all %ld saved clipboard items, including images. Your current clipboard will stay unchanged.",
	"alert.clear.confirm":       "Clear All History",
	"alert.key.titleConfigure":  "Configure Direct Jev",
	"alert.key.titleUpdate":     "Update Direct Jev Key",
	"alert.key.message":         "Your TypeSafe API key is stored in the system credential store, never in a plain file.",
	"alert.key.save":            "Save",
	"alert.key.placeholder":     "TypeSafe API key",
	"alert.key.envNote":         "Alternatively, set SMARTPASTE_DIRECT_JEV_KEY in the environment.",
	"alert.removeKey.title":     "Remove Direct Jev key?",
	"alert.removeKey.message":   "Smart Paste will stop sending suggestions until a new key is configured.",

	"language.load.title":              "Load Language File",
	"language.load.message":            "Choose a JSON language file.",
	"language.error.title":             "Could Not Load Language",
	"language.error.unreadable":        "The selected file could not be read.",
	"language.error.tooLarge":          "The language file is too large.",
	"language.error.invalidFormat":     "The file is not valid JSON with a language object.",
	"language.error.missingField":      "The language file is missing the \"%@\" field.",
	"language.error.emptyTranslations": "The language file does not contain any translations.",
	"language.error.invalidCode":       "The language code may only contain letters, numbers, dots, dashes, and underscores, and must start with a letter.",
	"language.error.reservedCode":      "The language code \"%@\" is reserved for a built-in language.",

	"timeline.search":    "Search clipboard history",
	"timeline.delete":    "Delete",
	"timeline.empty":     "Nothing here yet. Copy something to get started.",
	"timeline.noResults": "No matching items.",
	"timeline.image":     "Image",

	"time.justNow":    "just now",
	"time.minutesAgo": "%ld min ago",
	"time.hoursAgo":   "%ld h ago",
	"time.daysAgo":    "%ld d ago",

	"welcome.windowTitle": "Welcome to Smart Paste",
	"welcome.title":       "Welcome to Smart Paste",
	"welcome.body":        "Copy anything, then press %@ to paste the right value.\nOpen the tray icon to browse, search and manage your history.",
	"welcome.continue":    "Continue",

	"setup.title":                    "Set Up Smart Paste",
	"setup.subtitle":                 "Smart Paste runs in the tray and works in every app.",
	"setup.shortcut":                 "Copy anything, then press %@ where you want it.",
	"setup.continue":                 "Done",
	"setup.openSettings":             "Open Settings",
	"setup.requestPermissions":       "Request Permissions",
	"setup.refresh":                  "Refresh",
	"setup.note.macos":               "macOS requires Accessibility and Input Monitoring permission for global shortcuts and reading the focused field. After enabling Smart Paste in System Settings, return here.",
	"setup.note.windows":             "Windows needs no extra permission. Apps running as administrator may not expose their focused field to a normal-privilege Smart Paste; run Smart Paste as administrator in that case.",
	"setup.note.linux":               "Linux reads the focused field through AT-SPI2. Enable accessibility in your desktop settings if needed. Global shortcuts require X11; on Wayland use the tray icon.",
	"setup.permission.accessibility": "Accessibility",
	"setup.permission.input":         "Input Monitoring",
	"setup.permission.allowed":       "Allowed",
	"setup.permission.notAllowed":    "Not Allowed",
	"setup.permission.notRequired":   "Not Required",
	"setup.permission.focus":         "Focused field access",
	"setup.permission.clipboard":     "Clipboard access",
	"setup.permission.hotkey":        "Global shortcut",

	"status.clipboard.unavailable": "Clipboard monitoring is unavailable: %@",
	"status.focus.unavailable":     "Focused field access is unavailable; Smart Paste will open the history panel instead.",
	"status.hotkey.failed":         "Could not register %@: %@. Use the tray icon instead.",
	"status.hotkey.active":         "Global shortcut %@ is active.",
	"status.key.missing":           "No Direct Jev key configured. Suggestions are disabled until you add one.",
	"status.wayland":               "Wayland detected: global shortcuts and focused-field access depend on the compositor. The tray icon always works.",
	"status.restartShortcut":       "Restart Smart Paste to apply the new global shortcut.",

	"error.title":     "Smart Paste",
	"error.keyring":   "Could not access the system credential store. Set SMARTPASTE_DIRECT_JEV_KEY instead.",
	"error.clipboard": "Could not start clipboard monitoring: %@",
	"error.save":      "Could not save settings: %@",

	"field.textField": "Text field",
}

var chineseStrings = map[string]string{
	"app.name":    "Smart Paste",
	"app.tagline": "在任意位置粘贴最合适的值。",

	"menu.open":                "打开 Smart Paste（%@）",
	"menu.pauseCapture":        "暂停剪贴板捕获",
	"menu.resumeCapture":       "恢复剪贴板捕获",
	"menu.clearHistory":        "清除全部历史记录",
	"menu.key.configure":       "配置 Direct Jev 密钥…",
	"menu.key.update":          "更新 Direct Jev 密钥…",
	"menu.key.remove":          "移除 Direct Jev 密钥",
	"menu.shortcut":            "键盘快捷键（%@）",
	"menu.shortcut.commandV":   "Command+V",
	"menu.shortcut.commandJ":   "Command+J",
	"menu.shortcut.useV":       "使用 %@",
	"menu.shortcut.useJ":       "使用 %@",
	"menu.shortcut.hint":       "Smart Paste 使用独立的全局快捷键，普通粘贴仍然可用。",
	"menu.language":            "语言",
	"menu.language.load":       "加载语言文件…",
	"menu.language.openFolder": "打开语言文件夹",
	"menu.setup":               "设置与权限…",
	"menu.quit":                "退出 Smart Paste",

	"alert.ok":                  "好",
	"alert.cancel":              "取消",
	"alert.clear.title":         "要清除全部剪贴板历史记录吗？",
	"alert.clear.message.one":   "这将永久删除已保存的剪贴板项目（包括图片）。当前剪贴板的内容不会改变。",
	"alert.clear.message.other": "这将永久删除全部 %ld 个已保存的剪贴板项目（包括图片）。当前剪贴板的内容不会改变。",
	"alert.clear.confirm":       "清除全部历史记录",
	"alert.key.titleConfigure":  "配置 Direct Jev",
	"alert.key.titleUpdate":     "更新 Direct Jev 密钥",
	"alert.key.message":         "TypeSafe API 密钥会保存在系统凭据存储中，不会写入明文文件。",
	"alert.key.save":            "保存",
	"alert.key.placeholder":     "TypeSafe API 密钥",
	"alert.key.envNote":         "也可以设置环境变量 SMARTPASTE_DIRECT_JEV_KEY。",
	"alert.removeKey.title":     "要移除 Direct Jev 密钥吗？",
	"alert.removeKey.message":   "在配置新密钥之前，Smart Paste 将停止发送建议请求。",

	"language.load.title":              "加载语言文件",
	"language.load.message":            "选择一个 JSON 语言文件。",
	"language.error.title":             "无法加载语言",
	"language.error.unreadable":        "无法读取所选文件。",
	"language.error.tooLarge":          "语言文件过大。",
	"language.error.invalidFormat":     "该文件不是包含语言对象的有效 JSON。",
	"language.error.missingField":      "语言文件缺少“%@”字段。",
	"language.error.emptyTranslations": "语言文件不包含任何翻译。",
	"language.error.invalidCode":       "语言代码只能包含字母、数字、点、短横线和下划线，并且必须以字母开头。",
	"language.error.reservedCode":      "语言代码“%@”是为内置语言保留的。",

	"timeline.search":    "搜索剪贴板历史",
	"timeline.delete":    "删除",
	"timeline.empty":     "还没有记录，复制点内容试试。",
	"timeline.noResults": "没有匹配的内容。",
	"timeline.image":     "图片",

	"time.justNow":    "刚刚",
	"time.minutesAgo": "%ld 分钟前",
	"time.hoursAgo":   "%ld 小时前",
	"time.daysAgo":    "%ld 天前",

	"welcome.windowTitle": "欢迎使用 Smart Paste",
	"welcome.title":       "欢迎使用 Smart Paste",
	"welcome.body":        "复制任意内容，然后按 %@ 粘贴合适的值。\n打开托盘图标可以浏览、搜索和管理历史记录。",
	"welcome.continue":    "继续",

	"setup.title":                    "设置 Smart Paste",
	"setup.subtitle":                 "Smart Paste 常驻托盘，可在任何应用中使用。",
	"setup.shortcut":                 "复制任意内容，然后在需要的位置按下 %@。",
	"setup.continue":                 "完成",
	"setup.openSettings":             "打开系统设置",
	"setup.requestPermissions":       "请求权限",
	"setup.refresh":                  "刷新",
	"setup.note.macos":               "macOS 需要授予辅助功能和输入监控权限，才能使用全局快捷键并读取当前输入框。在系统设置中启用 Smart Paste 后回到这里。",
	"setup.note.windows":             "Windows 无需额外权限。以管理员身份运行的应用可能不会向普通权限的 Smart Paste 暴露输入框；此时请以管理员身份运行 Smart Paste。",
	"setup.note.linux":               "Linux 通过 AT-SPI2 读取当前输入框，如未生效请在桌面设置中启用无障碍功能。全局快捷键需要 X11；Wayland 下请使用托盘图标。",
	"setup.permission.accessibility": "辅助功能",
	"setup.permission.input":         "输入监控",
	"setup.permission.allowed":       "已允许",
	"setup.permission.notAllowed":    "未允许",
	"setup.permission.notRequired":   "无需",
	"setup.permission.focus":         "输入框读取",
	"setup.permission.clipboard":     "剪贴板访问",
	"setup.permission.hotkey":        "全局快捷键",

	"status.clipboard.unavailable": "剪贴板监听不可用：%@",
	"status.focus.unavailable":     "当前系统无法读取焦点输入框，Smart Paste 将改为打开历史面板。",
	"status.hotkey.failed":         "无法注册 %@：%@。请改用托盘图标。",
	"status.hotkey.active":         "全局快捷键 %@ 已启用。",
	"status.key.missing":           "尚未配置 Direct Jev 密钥，智能建议暂不可用。",
	"status.wayland":               "检测到 Wayland：全局快捷键和输入框读取取决于桌面环境，托盘图标始终可用。",
	"status.restartShortcut":       "重启 Smart Paste 后新的全局快捷键才会生效。",

	"error.title":     "Smart Paste",
	"error.keyring":   "无法访问系统凭据存储。请改为设置环境变量 SMARTPASTE_DIRECT_JEV_KEY。",
	"error.clipboard": "无法启动剪贴板监听：%@",
	"error.save":      "无法保存设置：%@",

	"field.textField": "文本输入框",
}

func init() {
	// Keep both packs in sync with the English source of truth so missing
	// translations fall back predictably.
	for key := range englishStrings {
		if _, ok := chineseStrings[key]; !ok {
			chineseStrings[key] = englishStrings[key]
		}
	}
}
