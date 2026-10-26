import Foundation

struct LanguagePack {
    let code: String
    let name: String
    let strings: [String: String]
    let isBuiltIn: Bool
}

enum LanguageFileError: LocalizedError {
    case unreadable
    case tooLarge
    case invalidFormat
    case missingField(String)
    case emptyTranslations
    case invalidCode
    case reservedCode(String)

    var errorDescription: String? {
        switch self {
        case .unreadable: return L("language.error.unreadable")
        case .tooLarge: return L("language.error.tooLarge")
        case .invalidFormat: return L("language.error.invalidFormat")
        case .missingField(let field): return L("language.error.missingField", field)
        case .emptyTranslations: return L("language.error.emptyTranslations")
        case .invalidCode: return L("language.error.invalidCode")
        case .reservedCode(let code): return L("language.error.reservedCode", code)
        }
    }
}

final class Localization {
    static let shared = Localization()
    static let changed = Notification.Name("SmartPasteLanguageChanged")

    private static let defaultsKey = "smartpaste.language"
    private static let maximumFileSize = 2 * 1024 * 1024
    private static let exampleFileName = "example.json"

    private static let english = LanguagePack(
        code: "en",
        name: "English",
        strings: englishStrings,
        isBuiltIn: true
    )
    private static let chinese = LanguagePack(
        code: "zh-Hans",
        name: "简体中文",
        strings: chineseStrings,
        isBuiltIn: true
    )
    private static let allBuiltIns = [english, chinese]

    let languagesDirectory: URL
    private var externalPacks: [String: LanguagePack]
    private var currentPack: LanguagePack

    private init() {
        let environment = ProcessInfo.processInfo.environment
        let dataDirectory: URL
        if let override = environment["SMARTPASTE_DATA_DIRECTORY"], !override.isEmpty {
            dataDirectory = URL(fileURLWithPath: override, isDirectory: true)
        } else {
            dataDirectory = FileManager.default.homeDirectoryForCurrentUser
                .appendingPathComponent("Library/Application Support/Smart Paste", isDirectory: true)
        }
        languagesDirectory = dataDirectory.appendingPathComponent("Languages", isDirectory: true)
        try? FileManager.default.createDirectory(
            at: languagesDirectory,
            withIntermediateDirectories: true
        )
        Self.writeExampleIfNeeded(in: languagesDirectory)

        let loaded = Self.loadExternalPacks(from: languagesDirectory)
        externalPacks = loaded
        currentPack = Self.resolve(UserDefaults.standard.string(forKey: Self.defaultsKey), in: loaded)
            ?? Self.resolve(Self.systemLanguageCode(in: loaded), in: loaded)
            ?? Self.english
    }

    var currentCode: String { currentPack.code }
    var currentName: String { currentPack.name }

    var availablePacks: [LanguagePack] {
        Self.allBuiltIns + externalPacks.values.sorted {
            $0.name.localizedStandardCompare($1.name) == .orderedAscending
        }
    }

    func pack(for code: String) -> LanguagePack? {
        Self.resolve(code, in: externalPacks)
    }

    func select(_ code: String) {
        guard let pack = pack(for: code) else { return }
        UserDefaults.standard.set(pack.code, forKey: Self.defaultsKey)
        guard pack.code != currentPack.code else { return }
        currentPack = pack
        NotificationCenter.default.post(name: Self.changed, object: nil)
    }

    @discardableResult
    func install(from url: URL) throws -> LanguagePack {
        guard let data = try? Data(contentsOf: url) else {
            throw LanguageFileError.unreadable
        }
        guard data.count <= Self.maximumFileSize else {
            throw LanguageFileError.tooLarge
        }

        let pack = try Self.parsePack(data)
        let destination = languagesDirectory.appendingPathComponent("\(pack.code).json")
        do {
            try data.write(to: destination, options: .atomic)
        } catch {
            throw LanguageFileError.unreadable
        }

        externalPacks = externalPacks.filter {
            $0.key.caseInsensitiveCompare(pack.code) != .orderedSame
        }
        externalPacks[pack.code] = pack

        if currentPack.code.caseInsensitiveCompare(pack.code) == .orderedSame {
            currentPack = pack
            NotificationCenter.default.post(name: Self.changed, object: nil)
        }
        return pack
    }

    func string(_ key: String) -> String {
        currentPack.strings[key] ?? Self.english.strings[key] ?? key
    }

    func format(_ key: String, _ arguments: [CVarArg]) -> String {
        Self.render(string(key), arguments: arguments)
    }

    private static func render(_ template: String, arguments: [CVarArg]) -> String {
        guard template.contains("%") else { return template }

        var output = ""
        var argumentIndex = 0
        var index = template.startIndex

        while index < template.endIndex {
            let character = template[index]
            guard character == "%" else {
                output.append(character)
                index = template.index(after: index)
                continue
            }

            let next = template.index(after: index)
            if next < template.endIndex, template[next] == "%" {
                output.append("%")
                index = template.index(after: next)
                continue
            }

            var consumed = 1
            var isPlaceholder = false
            if next < template.endIndex {
                if template[next] == "@" {
                    isPlaceholder = true
                    consumed = 2
                } else {
                    let after = template.index(after: next)
                    if template[next] == "l",
                       after < template.endIndex,
                       template[after] == "d" {
                        isPlaceholder = true
                        consumed = 3
                    } else if template[next] == "d" {
                        isPlaceholder = true
                        consumed = 2
                    }
                }
            }

            guard isPlaceholder else {
                output.append("%")
                index = template.index(after: index)
                continue
            }

            let end = template.index(index, offsetBy: consumed)
            if argumentIndex < arguments.count {
                let argument = arguments[argumentIndex]
                argumentIndex += 1
                if let text = argument as? String {
                    output += text
                } else if let number = argument as? Int {
                    output += String(number)
                } else {
                    output += String(describing: argument)
                }
            } else {
                output += String(template[index..<end])
            }
            index = end
        }

        return output
    }

    private static func resolve(_ code: String?, in external: [String: LanguagePack]) -> LanguagePack? {
        guard let code, !code.isEmpty else { return nil }
        if let pack = external.first(where: {
            $0.key.caseInsensitiveCompare(code) == .orderedSame
        })?.value {
            return pack
        }
        return allBuiltIns.first {
            $0.code.caseInsensitiveCompare(code) == .orderedSame
        }
    }

    private static func systemLanguageCode(in external: [String: LanguagePack]) -> String {
        for identifier in Locale.preferredLanguages {
            let lower = identifier.lowercased()
            if let builtIn = allBuiltIns.first(where: { lower.hasPrefix($0.code.lowercased()) }) {
                return builtIn.code
            }
            if let externalCode = external.keys.first(where: { lower.hasPrefix($0.lowercased()) }) {
                return externalCode
            }
            if lower.hasPrefix("zh") {
                return chinese.code
            }
        }
        return english.code
    }

    private static func writeExampleIfNeeded(in directory: URL) {
        let url = directory.appendingPathComponent(exampleFileName)
        guard !FileManager.default.fileExists(atPath: url.path) else { return }

        let instructions = "This is an example language pack. Change \"code\" to your "
            + "language code, change \"name\" to the name shown in the Language menu, "
            + "then translate every value in \"strings\". Keep the placeholders %@ and %ld "
            + "unchanged. Load your edited file with Language > Load Language File…, "
            + "or place it in this folder and restart Smart Paste."

        let root: [String: Any] = [
            "_instructions": instructions,
            "code": "example",
            "name": "Example (English Template)",
            "strings": english.strings
        ]

        guard let data = try? JSONSerialization.data(
            withJSONObject: root,
            options: [.prettyPrinted, .sortedKeys]
        ) else { return }
        try? data.write(to: url, options: .atomic)
    }

    private static func loadExternalPacks(from directory: URL) -> [String: LanguagePack] {        guard let urls = try? FileManager.default.contentsOfDirectory(
            at: directory,
            includingPropertiesForKeys: nil,
            options: [.skipsHiddenFiles]
        ) else {
            return [:]
        }

        var packs: [String: LanguagePack] = [:]
        for url in urls where url.pathExtension.lowercased() == "json" {
            guard let data = try? Data(contentsOf: url),
                  data.count <= maximumFileSize,
                  let pack = try? parsePack(data) else { continue }
            packs[pack.code] = pack
        }
        return packs
    }

    private static func parsePack(_ data: Data) throws -> LanguagePack {
        guard let object = try? JSONSerialization.jsonObject(with: data),
              let root = object as? [String: Any] else {
            throw LanguageFileError.invalidFormat
        }

        guard let code = (root["code"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines), !code.isEmpty else {
            throw LanguageFileError.missingField("code")
        }
        guard code.range(
            of: "^[A-Za-z][A-Za-z0-9._-]{0,31}$",
            options: .regularExpression
        ) != nil else {
            throw LanguageFileError.invalidCode
        }
        guard !allBuiltIns.contains(where: {
            $0.code.caseInsensitiveCompare(code) == .orderedSame
        }) else {
            throw LanguageFileError.reservedCode(code)
        }

        guard let name = (root["name"] as? String)?
            .trimmingCharacters(in: .whitespacesAndNewlines), !name.isEmpty else {
            throw LanguageFileError.missingField("name")
        }
        guard let rawStrings = root["strings"] as? [String: Any] else {
            throw LanguageFileError.missingField("strings")
        }

        var strings: [String: String] = [:]
        for (key, value) in rawStrings {
            if let text = value as? String {
                strings[key] = text
            }
        }
        guard !strings.isEmpty else {
            throw LanguageFileError.emptyTranslations
        }

        return LanguagePack(code: code, name: name, strings: strings, isBuiltIn: false)
    }
}

func L(_ key: String, _ arguments: CVarArg...) -> String {
    Localization.shared.format(key, arguments)
}

extension Localization {
    fileprivate static let englishStrings: [String: String] = [
        "menu.open": "Open Smart Paste Beta (%@)",
        "menu.pauseCapture": "Pause Clipboard Capture",
        "menu.resumeCapture": "Resume Clipboard Capture",
        "menu.clearHistory": "Clear All History",
        "menu.key.configure": "Configure Direct Jev Key…",
        "menu.key.update": "Update Direct Jev Key…",
        "menu.key.remove": "Remove Direct Jev Key",
        "menu.shortcut": "Keyboard Shortcut (%@)",
        "menu.shortcut.commandV": "Command+V",
        "menu.shortcut.commandJ": "Command+J",
        "menu.language": "Language",
        "menu.language.load": "Load Language File…",
        "menu.language.openFolder": "Open Languages Folder",
        "menu.quit": "Quit Smart Paste",

        "alert.ok": "OK",
        "alert.cancel": "Cancel",
        "alert.clear.title": "Clear all clipboard history?",
        "alert.clear.message.one": "This permanently removes the saved clipboard item, "
            + "including any image. Your current Mac clipboard will stay unchanged.",
        "alert.clear.message.other": "This permanently removes all %ld saved clipboard items, "
            + "including images. Your current Mac clipboard will stay unchanged.",
        "alert.clear.confirm": "Clear All History",
        "alert.key.titleConfigure": "Configure Direct Jev",
        "alert.key.titleUpdate": "Update Direct Jev Key",
        "alert.key.message": "Your TypeSafe API key is stored only in this Mac's Keychain.",
        "alert.key.save": "Save",
        "alert.key.placeholder": "TypeSafe API key",

        "language.load.title": "Load Language File",
        "language.load.message": "Choose a JSON language file.",
        "language.error.title": "Could Not Load Language",
        "language.error.unreadable": "The selected file could not be read.",
        "language.error.tooLarge": "The language file is too large.",
        "language.error.invalidFormat": "The file is not valid JSON with a language object.",
        "language.error.missingField": "The language file is missing the \"%@\" field.",
        "language.error.emptyTranslations": "The language file does not contain any translations.",
        "language.error.invalidCode": "The language code may only contain letters, numbers, "
            + "dots, dashes, and underscores, and must start with a letter.",
        "language.error.reservedCode": "The language code \"%@\" is reserved for a built-in language.",

        "timeline.search": "Search clipboard history",
        "timeline.delete": "Delete",
        "timeline.empty": "Nothing here yet. Copy something to get started.",
        "timeline.noResults": "No matching items.",
        "timeline.image": "Image",

        "welcome.windowTitle": "Welcome to Smart Paste",
        "welcome.title": "Welcome to Smart Paste Beta",
        "welcome.body": "Copy anything, then press %@ to paste the right value.\n"
            + "Press %@ to see all suggestions.",
        "welcome.continue": "Continue",

        "permission.windowTitle": "Set Up Smart Paste",
        "permission.title": "Set Up Smart Paste",
        "permission.subtitle": "Two permissions keep Smart Paste working everywhere.",
        "permission.shortcutLine": "Copy anything, then press it where you want it.",
        "permission.accessibility.title": "Accessibility",
        "permission.accessibility.explanation": "Lets Smart Paste read the focused field",
        "permission.accessibility.button": "Open Accessibility Settings",
        "permission.input.title": "Input Monitoring",
        "permission.input.explanation": "Lets Smart Paste notice %@ while you're working in another app.",
        "permission.input.button": "Open Input Monitoring Settings",
        "permission.footer": "Your clipboard history stays on this Mac. "
            + "After enabling Smart Paste in Settings, come back here. "
            + "This page updates automatically.",
        "permission.allowed": "Allowed",
        "permission.notAllowed": "Not Allowed",

        "field.textField": "Text field"
    ]

    fileprivate static let chineseStrings: [String: String] = [
        "menu.open": "打开 Smart Paste Beta（%@）",
        "menu.pauseCapture": "暂停剪贴板捕获",
        "menu.resumeCapture": "恢复剪贴板捕获",
        "menu.clearHistory": "清除全部历史记录",
        "menu.key.configure": "配置 Direct Jev 密钥…",
        "menu.key.update": "更新 Direct Jev 密钥…",
        "menu.key.remove": "移除 Direct Jev 密钥",
        "menu.shortcut": "键盘快捷键（%@）",
        "menu.shortcut.commandV": "Command+V",
        "menu.shortcut.commandJ": "Command+J",
        "menu.language": "语言",
        "menu.language.load": "加载语言文件…",
        "menu.language.openFolder": "打开语言文件夹",
        "menu.quit": "退出 Smart Paste",

        "alert.ok": "好",
        "alert.cancel": "取消",
        "alert.clear.title": "要清除全部剪贴板历史记录吗？",
        "alert.clear.message.one": "这将永久删除已保存的剪贴板项目（包括图片）。"
            + "当前 Mac 剪贴板的内容不会改变。",
        "alert.clear.message.other": "这将永久删除全部 %ld 个已保存的剪贴板项目（包括图片）。"
            + "当前 Mac 剪贴板的内容不会改变。",
        "alert.clear.confirm": "清除全部历史记录",
        "alert.key.titleConfigure": "配置 Direct Jev",
        "alert.key.titleUpdate": "更新 Direct Jev 密钥",
        "alert.key.message": "你的 TypeSafe API 密钥仅保存在这台 Mac 的钥匙串中。",
        "alert.key.save": "保存",
        "alert.key.placeholder": "TypeSafe API 密钥",

        "language.load.title": "加载语言文件",
        "language.load.message": "选择一个 JSON 语言文件。",
        "language.error.title": "无法加载语言",
        "language.error.unreadable": "无法读取所选文件。",
        "language.error.tooLarge": "语言文件过大。",
        "language.error.invalidFormat": "该文件不是包含语言对象的有效 JSON。",
        "language.error.missingField": "语言文件缺少“%@”字段。",
        "language.error.emptyTranslations": "语言文件不包含任何翻译。",
        "language.error.invalidCode": "语言代码只能包含字母、数字、点、短横线和下划线，"
            + "并且必须以字母开头。",
        "language.error.reservedCode": "语言代码“%@”是为内置语言保留的。",

        "timeline.search": "搜索剪贴板历史",
        "timeline.delete": "删除",
        "timeline.empty": "还没有记录，复制点内容试试。",
        "timeline.noResults": "没有匹配的内容。",
        "timeline.image": "图片",

        "welcome.windowTitle": "欢迎使用 Smart Paste",
        "welcome.title": "欢迎使用 Smart Paste Beta",
        "welcome.body": "复制任意内容，然后按 %@ 粘贴合适的值。\n按 %@ 查看全部建议。",
        "welcome.continue": "继续",

        "permission.windowTitle": "设置 Smart Paste",
        "permission.title": "设置 Smart Paste",
        "permission.subtitle": "两项权限让智能粘贴在任何地方都能正常工作。",
        "permission.shortcutLine": "复制任意内容，然后在需要的位置按下它。",
        "permission.accessibility.title": "辅助功能",
        "permission.accessibility.explanation": "让 Smart Paste 读取当前聚焦的输入框",
        "permission.accessibility.button": "打开辅助功能设置",
        "permission.input.title": "输入监控",
        "permission.input.explanation": "让 Smart Paste 在你使用其他应用时也能响应 %@。",
        "permission.input.button": "打开输入监控设置",
        "permission.footer": "剪贴板历史只保存在这台 Mac 上。"
            + "在系统设置中启用 Smart Paste 后回到这里，本页面会自动更新。",
        "permission.allowed": "已允许",
        "permission.notAllowed": "未允许",

        "field.textField": "文本输入框"
    ]
}
