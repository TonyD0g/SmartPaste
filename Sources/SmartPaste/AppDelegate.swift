import AppKit
import ApplicationServices
import CoreGraphics
import Foundation
import UniformTypeIdentifiers

private let smartPasteEventTapCallback: CGEventTapCallBack = {
    proxy, type, event, refcon in
    guard let refcon else { return Unmanaged.passUnretained(event) }
    let delegate = Unmanaged<AppDelegate>.fromOpaque(refcon).takeUnretainedValue()

    if type == .tapDisabledByTimeout || type == .tapDisabledByUserInput {
        if let eventTap = delegate.eventTap {
            CGEvent.tapEnable(tap: eventTap, enable: true)
        }
        return Unmanaged.passUnretained(event)
    }

    guard type == .keyDown else {
        return Unmanaged.passUnretained(event)
    }

    let flags = event.flags
    guard flags.contains(.maskCommand),
          !flags.contains(.maskControl),
          !flags.contains(.maskAlternate) else {
        return Unmanaged.passUnretained(event)
    }

    let keyCode = CGKeyCode(event.getIntegerValueField(.keyboardEventKeycode))
    guard keyCode == SmartPasteShortcut.current.keyCode else {
        return Unmanaged.passUnretained(event)
    }

    return delegate.handleSmartPaste() ? nil : Unmanaged.passUnretained(event)
}

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let watcher = ClipboardWatcher()
    private let fields = FieldObserver()
    private let client = SuggestionClient()
    private let timeline = TimelineController()

    private var status: NSStatusItem?
    private var openTimelineMenuItem: NSMenuItem?
    private var timer: Timer?
    private var focusMonitor: Any?
    private var focused: FocusedField?
    private var suggestion: Clip?
    private var isRequesting = false
    private var pasteWhenReady = false
    private var bypassNextPaste = false
    private var generation = 0
    fileprivate var eventTap: CFMachPort?
    fileprivate var eventTapSource: CFRunLoopSource?

    private var configureDirectKeyItem: NSMenuItem?
    private var removeDirectKeyItem: NSMenuItem?
    private var shortcutMenuItem: NSMenuItem?
    private var commandVShortcutItem: NSMenuItem?
    private var commandJShortcutItem: NSMenuItem?
    private var captureMenuItem: NSMenuItem?
    private var clearAllMenuItem: NSMenuItem?
    private var permissionSetup: PermissionSetupWindowController?
    private var welcome: WelcomeWindowController?

    private let pausedKey = "smartpaste.clipboardCapturePaused"
    private let firstLaunchKey = "smartpaste.hasShownFirstLaunchUI"

    func applicationDidFinishLaunching(_ notification: Notification) {
        let trusted = AXIsProcessTrusted()
        let inputAllowed = CGPreflightListenEventAccess()
        log(
            "trusted=\(trusted ? "true" : "false")"
                + " inputAllowed=\(inputAllowed ? "true" : "false")"
                + " frontmost=\(Bundle.main.bundleIdentifier ?? "unknown")"
                + " capturePaused=\(UserDefaults.standard.bool(forKey: pausedKey) ? "true" : "false")\n"
        )

        NSApp.setActivationPolicy(.accessory)
        buildStatusItem()
        buildMenu()

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleLanguageChanged(_:)),
            name: Localization.changed,
            object: nil
        )

        NotificationCenter.default.addObserver(
            self,
            selector: #selector(handleClipsChanged(_:)),
            name: .clipsChanged,
            object: nil
        )

        timeline.onPaste = { [weak self] clip, app in
            app?.activate(options: [.activateIgnoringOtherApps])
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.08) {
                self?.paste(clip, into: nil)
            }
        }

        watcher.start()

        if UserDefaults.standard.bool(forKey: firstLaunchKey) {
            continueLaunch(trusted: trusted, inputAllowed: inputAllowed)
        } else {
            showWelcome()
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        timer?.invalidate()
        timer = nil
        if let focusMonitor {
            NSEvent.removeMonitor(focusMonitor)
        }
        if let eventTap {
            CGEvent.tapEnable(tap: eventTap, enable: false)
        }
        if let eventTapSource {
            CFRunLoopRemoveSource(CFRunLoopGetMain(), eventTapSource, .commonModes)
        }
    }

    func applicationShouldHandleReopen(
        _ sender: NSApplication,
        hasVisibleWindows flag: Bool
    ) -> Bool {
        timeline.show()
        return true
    }

    private func buildStatusItem() {
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        status = item

        if let button = item.button {
            if let image = NSImage(named: "SmartPasteMenuTemplate") {
                image.isTemplate = true
                image.size = NSSize(width: 18, height: 18)
                button.image = image
                button.imagePosition = .imageOnly
            } else {
                button.title = "J"
                button.font = .monospacedSystemFont(ofSize: 15, weight: .semibold)
            }
            button.setAccessibilityLabel("Smart Paste")
        }
    }

    private func buildMenu() {
        let menu = NSMenu(title: "")
        menu.autoenablesItems = false

        let openItem = NSMenuItem(
            title: L("menu.open", SmartPasteShortcut.current.displayKey),
            action: #selector(openTimeline),
            keyEquivalent: ""
        )
        openItem.target = self
        menu.addItem(openItem)
        openTimelineMenuItem = openItem

        let capture = NSMenuItem(
            title: L("menu.pauseCapture"),
            action: #selector(toggleClipboardCapture),
            keyEquivalent: ""
        )
        capture.target = self
        menu.addItem(capture)
        captureMenuItem = capture
        updateCaptureMenu()

        let clear = NSMenuItem(
            title: L("menu.clearHistory"),
            action: #selector(clearAllHistory),
            keyEquivalent: ""
        )
        clear.target = self
        menu.addItem(clear)
        clearAllMenuItem = clear
        clear.isEnabled = !MemoryStore.shared.clips.isEmpty

        menu.addItem(.separator())

        let configure = NSMenuItem(
            title: L("menu.key.configure"),
            action: #selector(configureDirectJevKey),
            keyEquivalent: ""
        )
        configure.target = self
        menu.addItem(configure)
        configureDirectKeyItem = configure

        let removeKey = NSMenuItem(
            title: L("menu.key.remove"),
            action: #selector(removeDirectJevKey),
            keyEquivalent: ""
        )
        removeKey.target = self
        menu.addItem(removeKey)
        removeDirectKeyItem = removeKey

        updateKeyMenu()

        let shortcutTitle = L("menu.shortcut", SmartPasteShortcut.current.displayKey)
        let shortcut = NSMenuItem(title: shortcutTitle, action: nil, keyEquivalent: "")
        let shortcutMenu = NSMenu(title: shortcutTitle)
        shortcutMenuItem = shortcut

        let commandV = NSMenuItem(
            title: L("menu.shortcut.commandV"),
            action: #selector(useCommandVForSmartPaste),
            keyEquivalent: ""
        )
        commandV.target = self
        shortcutMenu.addItem(commandV)
        commandVShortcutItem = commandV

        let commandJ = NSMenuItem(
            title: L("menu.shortcut.commandJ"),
            action: #selector(useCommandJForSmartPaste),
            keyEquivalent: ""
        )
        commandJ.target = self
        shortcutMenu.addItem(commandJ)
        commandJShortcutItem = commandJ

        shortcut.submenu = shortcutMenu
        menu.addItem(shortcut)
        updateShortcutMenu()

        buildLanguageMenuItem(in: menu)

        menu.addItem(.separator())

        let quit = NSMenuItem(
            title: L("menu.quit"),
            action: #selector(NSApplication.terminate(_:)),
            keyEquivalent: ""
        )
        menu.addItem(quit)

        status?.menu = menu
    }

    private func buildLanguageMenuItem(in menu: NSMenu) {
        let language = NSMenuItem(title: L("menu.language"), action: nil, keyEquivalent: "")
        let languageMenu = NSMenu(title: L("menu.language"))
        languageMenu.autoenablesItems = false

        for pack in Localization.shared.availablePacks {
            let item = NSMenuItem(
                title: pack.name,
                action: #selector(selectLanguage(_:)),
                keyEquivalent: ""
            )
            item.target = self
            item.representedObject = pack.code
            item.state = pack.code == Localization.shared.currentCode ? .on : .off
            languageMenu.addItem(item)
        }

        languageMenu.addItem(.separator())

        let load = NSMenuItem(
            title: L("menu.language.load"),
            action: #selector(loadLanguageFile),
            keyEquivalent: ""
        )
        load.target = self
        languageMenu.addItem(load)

        let openFolder = NSMenuItem(
            title: L("menu.language.openFolder"),
            action: #selector(openLanguagesFolder),
            keyEquivalent: ""
        )
        openFolder.target = self
        languageMenu.addItem(openFolder)

        language.submenu = languageMenu
        menu.addItem(language)
    }

    private func updateCaptureMenu() {
        let paused = UserDefaults.standard.bool(forKey: pausedKey)
        captureMenuItem?.title = paused ? L("menu.resumeCapture") : L("menu.pauseCapture")
        captureMenuItem?.state = paused ? .on : .off
    }

    private func updateKeyMenu() {
        let hasKey = TypeSafeKeychain.shared.read() != nil

        configureDirectKeyItem?.title = hasKey
            ? L("menu.key.update")
            : L("menu.key.configure")
        removeDirectKeyItem?.isEnabled = hasKey
    }

    private func updateShortcutMenu() {
        let shortcut = SmartPasteShortcut.current
        shortcutMenuItem?.title = L("menu.shortcut", shortcut.displayKey)
        openTimelineMenuItem?.title = L("menu.open", shortcut.displayKey)
        commandVShortcutItem?.state = shortcut == .commandV ? .on : .off
        commandJShortcutItem?.state = shortcut == .commandJ ? .on : .off
    }

    private func showWelcome() {
        let controller: WelcomeWindowController
        if let existing = welcome {
            controller = existing
        } else {
            controller = WelcomeWindowController()
            welcome = controller
        }

        controller.onContinue = { [weak self] in
            guard let self else { return }
            UserDefaults.standard.set(true, forKey: self.firstLaunchKey)
            self.continueLaunch(
                trusted: AXIsProcessTrusted(),
                inputAllowed: CGPreflightListenEventAccess()
            )
        }

        NSApp.setActivationPolicy(.regular)
        controller.window?.center()
        controller.showWindow(nil)
        controller.window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
    }

    private func continueLaunch(trusted: Bool, inputAllowed: Bool) {
        let preview = ProcessInfo.processInfo.environment["SMARTPASTE_PERMISSION_ONBOARDING_PREVIEW"] == "1"
        if preview {
            showPermissionSetup()
            return
        }

        guard trusted, inputAllowed else {
            showPermissionSetup()
            return
        }

        if timer == nil {
            installKeys()
            installFocusPrefetch()
            startSmartPasteRuntime()
            if TypeSafeKeychain.shared.read() == nil {
                configureDirectJevKey()
            }
        }
        NSApp.setActivationPolicy(.accessory)
    }

    private func showPermissionSetup() {
        let controller: PermissionSetupWindowController
        if let existing = permissionSetup {
            controller = existing
        } else {
            controller = PermissionSetupWindowController()
            permissionSetup = controller
        }

        controller.onPermissionsGranted = { [weak self] in
            self?.continueLaunch(trusted: true, inputAllowed: true)
        }
        controller.begin()
    }

    private func installKeys() {
        guard eventTap == nil else { return }

        let mask = CGEventMask(1 << CGEventType.keyDown.rawValue)
        guard let tap = CGEvent.tapCreate(
            tap: .cgSessionEventTap,
            place: .headInsertEventTap,
            options: .defaultTap,
            eventsOfInterest: mask,
            callback: smartPasteEventTapCallback,
            userInfo: Unmanaged.passUnretained(self).toOpaque()
        ) else {
            log("eventTap=failed\n")
            return
        }

        eventTap = tap
        if let source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0) {
            eventTapSource = source
            CFRunLoopAddSource(CFRunLoopGetMain(), source, .commonModes)
        }
        CGEvent.tapEnable(tap: tap, enable: true)
        log("eventTap=ok\n")
    }

    private func installFocusPrefetch() {
        guard focusMonitor == nil else { return }
        focusMonitor = NSEvent.addGlobalMonitorForEvents(matching: .leftMouseDown) { [weak self] _ in
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.06) {
                self?.observeField()
            }
        }
    }

    private func startSmartPasteRuntime() {
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 0.16, repeats: true) { [weak self] _ in
            self?.observeField()
        }
    }

    private func observeField() {
        guard timeline.window?.isVisible != true else { return }

        guard let current = fields.current() else {
            focused = nil
            suggestion = nil
            isRequesting = false
            pasteWhenReady = false
            timeline.rankedTexts = []
            timeline.reload()
                generation += 1
            return
        }

        if let previous = focused,
           let previousElement = previous.element,
           let currentElement = current.element,
           CFEqual(previousElement, currentElement),
           previous.label == current.label {
            return
        }

        focused = current
        suggestion = nil
        isRequesting = false
        pasteWhenReady = false
        timeline.rankedTexts = []
        timeline.reload()
        generation += 1

        beginSuggestion(for: current, pasteAfterResponse: false)
    }

    private func beginSuggestion(for field: FocusedField, pasteAfterResponse: Bool) {
        focused = field
        suggestion = nil
        timeline.rankedTexts = []
        timeline.reload()
        pasteWhenReady = pasteAfterResponse
        isRequesting = true
        generation += 1
        let currentGeneration = generation

        let values = client.candidateValues(from: MemoryStore.shared.textClips)
        guard !values.isEmpty else {
            apply(.noMatch, generation: currentGeneration, field: field)
            return
        }

        client.suggest(field: field, values: values) { [weak self] outcome in
            guard let self else { return }
            self.apply(outcome, generation: currentGeneration, field: field)
        }

        if pasteAfterResponse {
            DispatchQueue.main.asyncAfter(deadline: .now() + 1.25) { [weak self] in
                guard let self,
                      self.generation == currentGeneration,
                      self.isRequesting else { return }
                self.isRequesting = false
                self.pasteWhenReady = false
                self.fallbackPaste(field: self.focused)
            }
        }
    }

    private func apply(
        _ outcome: SuggestionOutcome,
        generation expectedGeneration: Int,
        field: FocusedField
    ) {
        guard expectedGeneration == generation else { return }

        guard let focus = focused,
              let focusElement = focus.element,
              let capturedElement = field.element,
              CFEqual(focusElement, capturedElement),
              focus.label == field.label else {
            return
        }

        isRequesting = false
        let shouldPaste = pasteWhenReady
        pasteWhenReady = false

        switch outcome {
        case .match(let clip, let confidence, let rankedTexts):
            self.timeline.rankedTexts = rankedTexts
            timeline.reload()

            if confidence >= 0.4 {
                if shouldPaste {
                    paste(clip, into: focused)
                } else {
                    suggestion = clip
                }
            } else if shouldPaste {
                fallbackPaste(field: focused)
            }

        case .noMatch, .failure:
            timeline.rankedTexts = []
            timeline.reload()

            if shouldPaste {
                fallbackPaste(field: focused)
            }
        }
    }

    private func fallbackPaste(field: FocusedField?) {
        if let field,
           !prefersNativePaste(for: field),
           let text = NSPasteboard.general.string(forType: .string),
           insertTextAtSelection(text, into: field) {
            finishPaste()
        } else {
            performNormalPaste()
        }
    }

    private func paste(_ clip: Clip, into field: FocusedField?) {
        if clip.imageFileName == nil,
           let field,
           !prefersNativePaste(for: field),
           insertTextAtSelection(clip.text, into: field) {
            finishPaste()
            return
        }

        let saved = savePasteboard()
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()

        if let image = MemoryStore.shared.image(for: clip) {
            if !pasteboard.writeObjects([image]) {
                restorePasteboard(saved)
                return
            }
        } else if !pasteboard.setString(clip.text, forType: .string) {
            restorePasteboard(saved)
            return
        }

        watcher.lastChange = pasteboard.changeCount
        performNormalPaste()

        let usesAX = field.map { prefersNativePaste(for: $0) } ?? false
        let delay: Double = (clip.imageFileName != nil || usesAX) ? 0.35 : 0.18
        DispatchQueue.main.asyncAfter(deadline: .now() + delay) {
            self.restorePasteboard(saved)
        }

        finishPaste()
    }

    private func finishPaste() {
        suggestion = nil
        isRequesting = false
        pasteWhenReady = false
        generation += 1
    }

    fileprivate func handleSmartPaste() -> Bool {
        guard timeline.window?.isVisible != true else { return false }

        let captured = watcher.captureIfChanged()

        if bypassNextPaste {
            bypassNextPaste = false
            return false
        }

        guard let current = fields.current() else { return false }
        guard !MemoryStore.shared.textClips.isEmpty else { return false }

        if !captured,
           let previous = focused,
           let previousElement = previous.element,
           let currentElement = current.element,
           CFEqual(previousElement, currentElement),
           previous.label == current.label {
            if let suggestion {
                paste(suggestion, into: current)
                return true
            }
            if isRequesting {
                pasteWhenReady = true
                return true
            }
        }

        beginSuggestion(for: current, pasteAfterResponse: true)
        return true
    }

    private func prefersNativePaste(for field: FocusedField) -> Bool {
        guard let element = field.element else { return true }
        if field.role == "AXSecureTextField" {
            return true
        }

        var settable = DarwinBoolean(false)
        if AXUIElementIsAttributeSettable(
            element,
            kAXSelectedTextAttribute as CFString,
            &settable
        ) == .success, settable.boolValue {
            return false
        }
        if AXUIElementIsAttributeSettable(
            element,
            kAXValueAttribute as CFString,
            &settable
        ) == .success, settable.boolValue {
            return false
        }
        return true
    }

    private func insertTextAtSelection(_ text: String, into field: FocusedField) -> Bool {
        guard let element = field.element else { return false }

        let selected = AXUIElementSetAttributeValue(
            element,
            kAXSelectedTextAttribute as CFString,
            text as CFString
        )
        if selected == .success {
            return true
        }

        var current: CFTypeRef?
        if AXUIElementCopyAttributeValue(
            element,
            kAXValueAttribute as CFString,
            &current
        ) == .success,
           let value = current as? String,
           value.isEmpty {
            return AXUIElementSetAttributeValue(
                element,
                kAXValueAttribute as CFString,
                text as CFString
            ) == .success
        }
        return false
    }

    private func performNormalPaste() {
        bypassNextPaste = SmartPasteShortcut.current == .commandV

        let source = CGEventSource(stateID: .combinedSessionState)
        let down = CGEvent(keyboardEventSource: source, virtualKey: 9, keyDown: true)
        down?.flags = .maskCommand
        down?.post(tap: .cghidEventTap)

        let up = CGEvent(keyboardEventSource: source, virtualKey: 9, keyDown: false)
        up?.flags = .maskCommand
        up?.post(tap: .cghidEventTap)
    }

    private func savePasteboard() -> [[NSPasteboard.PasteboardType: Data]] {
        (NSPasteboard.general.pasteboardItems ?? []).map { item in
            var data: [NSPasteboard.PasteboardType: Data] = [:]
            for type in item.types {
                if let value = item.data(forType: type) {
                    data[type] = value
                }
            }
            return data
        }
    }

    private func restorePasteboard(_ saved: [[NSPasteboard.PasteboardType: Data]]) {
        guard !saved.isEmpty else { return }
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        let items = saved.map { entry -> NSPasteboardItem in
            let item = NSPasteboardItem()
            for (type, data) in entry {
                item.setData(data, forType: type)
            }
            return item
        }
        pasteboard.writeObjects(items)
        watcher.lastChange = pasteboard.changeCount
    }

    @objc private func openTimeline() {
        timeline.show()
    }

    @objc private func handleLanguageChanged(_ notification: Notification) {
        DispatchQueue.main.async { [weak self] in
            self?.buildMenu()
        }
    }

    @objc private func selectLanguage(_ sender: NSMenuItem) {
        guard let code = sender.representedObject as? String else { return }
        Localization.shared.select(code)
    }

    @objc private func loadLanguageFile() {
        NSApp.activate(ignoringOtherApps: true)

        let panel = NSOpenPanel()
        panel.title = L("language.load.title")
        panel.message = L("language.load.message")
        panel.allowedContentTypes = [.json]
        panel.allowsMultipleSelection = false
        panel.canChooseDirectories = false
        panel.canChooseFiles = true

        guard panel.runModal() == .OK, let url = panel.url else { return }

        do {
            let pack = try Localization.shared.install(from: url)
            Localization.shared.select(pack.code)
        } catch {
            let alert = NSAlert()
            alert.messageText = L("language.error.title")
            alert.informativeText = error.localizedDescription
            alert.alertStyle = .warning
            alert.addButton(withTitle: L("alert.ok"))
            alert.runModal()
        }
    }

    @objc private func openLanguagesFolder() {
        let directory = Localization.shared.languagesDirectory
        try? FileManager.default.createDirectory(
            at: directory,
            withIntermediateDirectories: true
        )
        NSWorkspace.shared.open(directory)
    }

    @objc private func toggleClipboardCapture() {
        let defaults = UserDefaults.standard
        defaults.set(!defaults.bool(forKey: pausedKey), forKey: pausedKey)
        watcher.lastChange = NSPasteboard.general.changeCount
        updateCaptureMenu()
    }

    @objc private func handleClipsChanged(_ notification: Notification) {
        clearAllMenuItem?.isEnabled = !MemoryStore.shared.clips.isEmpty
    }

    @objc private func clearAllHistory() {
        let clips = MemoryStore.shared.clips
        guard !clips.isEmpty else { return }

        let alert = NSAlert()
        alert.messageText = L("alert.clear.title")
        alert.informativeText = clips.count == 1
            ? L("alert.clear.message.one")
            : L("alert.clear.message.other", clips.count)
        alert.alertStyle = .warning
        alert.addButton(withTitle: L("alert.clear.confirm"))
        alert.addButton(withTitle: L("alert.cancel"))

        guard alert.runModal() == .alertFirstButtonReturn else { return }

        timeline.hide()
        MemoryStore.shared.remove(clips)
        generation += 1
        focused = nil
        suggestion = nil
        pasteWhenReady = false
    }

    @objc private func configureDirectJevKey() {
        NSApp.activate(ignoringOtherApps: true)
        let existing = TypeSafeKeychain.shared.read()

        let alert = NSAlert()
        alert.messageText = existing == nil
            ? L("alert.key.titleConfigure")
            : L("alert.key.titleUpdate")
        alert.informativeText = L("alert.key.message")
        alert.alertStyle = .informational
        alert.addButton(withTitle: L("alert.key.save"))
        alert.addButton(withTitle: L("alert.cancel"))

        let field = NSSecureTextField(frame: NSRect(x: 0, y: 0, width: 300, height: 24))
        field.placeholderString = L("alert.key.placeholder")
        alert.accessoryView = field
        alert.window.initialFirstResponder = field

        guard alert.runModal() == .alertFirstButtonReturn else { return }
        let value = field.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty, TypeSafeKeychain.shared.save(value) else { return }

        updateKeyMenu()
    }

    @objc private func removeDirectJevKey() {
        TypeSafeKeychain.shared.remove()
        updateKeyMenu()
    }

    @objc private func useCommandVForSmartPaste() {
        SmartPasteShortcut.current = .commandV
        updateShortcutMenu()
    }

    @objc private func useCommandJForSmartPaste() {
        SmartPasteShortcut.current = .commandJ
        bypassNextPaste = false
        updateShortcutMenu()
    }

    private func log(_ message: String) {
        let directory = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Logs", isDirectory: true)
        try? FileManager.default.createDirectory(
            at: directory,
            withIntermediateDirectories: true,
            attributes: nil
        )
        let url = directory.appendingPathComponent("Smart Paste.log")

        guard let data = message.data(using: .utf8) else { return }
        if FileManager.default.fileExists(atPath: url.path) {
            if let handle = try? FileHandle(forWritingTo: url) {
                handle.seekToEndOfFile()
                handle.write(data)
                try? handle.close()
            }
        } else {
            try? data.write(to: url, options: .atomic)
        }
    }
}
