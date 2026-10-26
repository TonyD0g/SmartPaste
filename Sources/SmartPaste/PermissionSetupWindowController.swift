import AppKit
import ApplicationServices
import CoreGraphics

final class PermissionSetupWindowController: NSWindowController {
    var onPermissionsGranted: (() -> Void)?

    private let accessibilityStatus = NSTextField(labelWithString: "")
    private let inputStatus = NSTextField(labelWithString: "")
    private var timer: Timer?
    private var hasRequestedAccessibility = false
    private let isPreview =
        ProcessInfo.processInfo.environment["SMARTPASTE_PERMISSION_ONBOARDING_PREVIEW"] == "1"

    private let accessibilityURL = URL(
        string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility"
    )!
    private let inputMonitoringURL = URL(
        string: "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent"
    )!

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 580, height: 480),
            styleMask: [.titled, .closable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.title = L("permission.windowTitle")
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.isMovableByWindowBackground = true
        window.isReleasedWhenClosed = false
        window.center()

        super.init(window: window)
        buildContent()
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(applyLocalization),
            name: Localization.changed,
            object: nil
        )
    }

    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    deinit {
        timer?.invalidate()
    }

    private func buildContent() {
        guard let content = window?.contentView else { return }
        content.subviews.forEach { $0.removeFromSuperview() }

        let background = NSVisualEffectView()
        background.material = .underWindowBackground
        background.blendingMode = .behindWindow
        background.state = .active
        background.translatesAutoresizingMaskIntoConstraints = false
        content.addSubview(background)

        NSLayoutConstraint.activate([
            background.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            background.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            background.topAnchor.constraint(equalTo: content.topAnchor),
            background.bottomAnchor.constraint(equalTo: content.bottomAnchor)
        ])

        let stack = NSStackView()
        stack.orientation = .vertical
        stack.alignment = .leading
        stack.spacing = 16
        stack.translatesAutoresizingMaskIntoConstraints = false
        background.addSubview(stack)

        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: background.leadingAnchor, constant: 28),
            stack.trailingAnchor.constraint(equalTo: background.trailingAnchor, constant: -28),
            stack.topAnchor.constraint(equalTo: background.topAnchor, constant: 54),
            stack.bottomAnchor.constraint(
                lessThanOrEqualTo: background.bottomAnchor,
                constant: -24
            )
        ])

        let header = makeHeader()
        stack.addArrangedSubview(header)
        header.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true

        let shortcutCard = makeShortcutCard()
        stack.addArrangedSubview(shortcutCard)
        shortcutCard.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true

        let shortcut = SmartPasteShortcut.current.displayKey
        let accessibilityCard = makePermissionCard(
            symbol: "accessibility",
            title: L("permission.accessibility.title"),
            explanation: L("permission.accessibility.explanation"),
            status: accessibilityStatus,
            buttonTitle: L("permission.accessibility.button"),
            action: #selector(openAccessibilitySettings)
        )
        stack.addArrangedSubview(accessibilityCard)
        accessibilityCard.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true

        let inputCard = makePermissionCard(
            symbol: "keyboard",
            title: L("permission.input.title"),
            explanation: L("permission.input.explanation", shortcut),
            status: inputStatus,
            buttonTitle: L("permission.input.button"),
            action: #selector(openInputMonitoringSettings)
        )
        stack.addArrangedSubview(inputCard)
        inputCard.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true

        let footer = NSTextField(
            wrappingLabelWithString: L("permission.footer")
        )
        footer.font = .systemFont(ofSize: 11)
        footer.textColor = .secondaryLabelColor
        footer.maximumNumberOfLines = 3
        stack.addArrangedSubview(footer)
        footer.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true
    }

    private func makeHeader() -> NSView {
        let header = NSStackView()
        header.orientation = .horizontal
        header.alignment = .centerY
        header.spacing = 16

        let icon = NSImageView(image: NSApp.applicationIconImage ?? NSImage())
        icon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 56),
            icon.heightAnchor.constraint(equalToConstant: 56)
        ])
        header.addArrangedSubview(icon)

        let titles = NSStackView()
        titles.orientation = .vertical
        titles.alignment = .leading
        titles.spacing = 4

        let title = NSTextField(labelWithString: L("permission.title"))
        title.font = .systemFont(ofSize: 24, weight: .semibold)
        titles.addArrangedSubview(title)

        let subtitle = NSTextField(labelWithString: L("permission.subtitle"))
        subtitle.font = .systemFont(ofSize: 12)
        subtitle.textColor = .secondaryLabelColor
        titles.addArrangedSubview(subtitle)

        header.addArrangedSubview(titles)
        return header
    }

    private func makeShortcutCard() -> NSView {
        let card = CardView()

        let row = NSStackView()
        row.orientation = .horizontal
        row.alignment = .centerY
        row.spacing = 8
        row.translatesAutoresizingMaskIntoConstraints = false

        let shortcut = SmartPasteShortcut.current.displayKey
        row.addArrangedSubview(KeycapView("⌘"))
        row.addArrangedSubview(KeycapView(String(shortcut.suffix(1))))

        let label = NSTextField(labelWithString: L("permission.shortcutLine"))
        label.font = .systemFont(ofSize: 13, weight: .medium)
        label.textColor = .secondaryLabelColor
        row.addArrangedSubview(label)

        card.addSubview(row)
        NSLayoutConstraint.activate([
            row.centerXAnchor.constraint(equalTo: card.centerXAnchor),
            row.centerYAnchor.constraint(equalTo: card.centerYAnchor),
            row.topAnchor.constraint(greaterThanOrEqualTo: card.topAnchor, constant: 12),
            row.bottomAnchor.constraint(lessThanOrEqualTo: card.bottomAnchor, constant: -12),
            card.heightAnchor.constraint(greaterThanOrEqualToConstant: 64)
        ])

        return card
    }

    private func makePermissionCard(
        symbol: String,
        title: String,
        explanation: String,
        status: NSTextField,
        buttonTitle: String,
        action: Selector
    ) -> NSView {
        let card = CardView()

        let content = NSStackView()
        content.orientation = .vertical
        content.alignment = .leading
        content.spacing = 9
        content.translatesAutoresizingMaskIntoConstraints = false
        card.addSubview(content)

        NSLayoutConstraint.activate([
            content.leadingAnchor.constraint(equalTo: card.leadingAnchor, constant: 14),
            content.trailingAnchor.constraint(equalTo: card.trailingAnchor, constant: -14),
            content.topAnchor.constraint(equalTo: card.topAnchor, constant: 14),
            content.bottomAnchor.constraint(equalTo: card.bottomAnchor, constant: -14)
        ])

        let header = NSStackView()
        header.orientation = .horizontal
        header.alignment = .centerY
        header.spacing = 8

        let configuration = NSImage.SymbolConfiguration(pointSize: 16, weight: .medium)
        let icon = NSImageView(
            image: NSImage(systemSymbolName: symbol, accessibilityDescription: nil)?
                .withSymbolConfiguration(configuration) ?? NSImage()
        )
        icon.contentTintColor = .secondaryLabelColor
        icon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 20),
            icon.heightAnchor.constraint(equalToConstant: 20)
        ])
        header.addArrangedSubview(icon)

        let titleLabel = NSTextField(labelWithString: title)
        titleLabel.font = .systemFont(ofSize: 15, weight: .semibold)
        header.addArrangedSubview(titleLabel)

        let spacer = NSView()
        spacer.setContentHuggingPriority(.defaultLow, for: .horizontal)
        header.addArrangedSubview(spacer)

        status.font = .systemFont(ofSize: 12, weight: .semibold)
        status.alignment = .right
        header.addArrangedSubview(status)

        content.addArrangedSubview(header)
        header.widthAnchor.constraint(equalTo: content.widthAnchor).isActive = true

        let explanationLabel = NSTextField(wrappingLabelWithString: explanation)
        explanationLabel.font = .systemFont(ofSize: 12.5)
        explanationLabel.textColor = .secondaryLabelColor
        explanationLabel.maximumNumberOfLines = 2
        content.addArrangedSubview(explanationLabel)
        explanationLabel.widthAnchor.constraint(equalTo: content.widthAnchor).isActive = true

        let button = NSButton(title: buttonTitle, target: self, action: action)
        button.bezelStyle = .rounded
        button.controlSize = .small
        content.addArrangedSubview(button)

        return card
    }

    func begin() {
        NSApp.setActivationPolicy(.regular)
        showWindow(nil)
        window?.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)

        refresh()

        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 0.8, repeats: true) { [weak self] _ in
            self?.refresh()
        }

        guard !isPreview else { return }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.7) { [weak self] in
            self?.requestAccessibilityIfNeeded()
        }
    }

    private func requestAccessibilityIfNeeded() {
        guard !hasRequestedAccessibility else { return }
        hasRequestedAccessibility = true
        let options = ["AXTrustedCheckOptionPrompt": true] as CFDictionary
        _ = AXIsProcessTrustedWithOptions(options)
        _ = CGRequestListenEventAccess()
    }

    @objc private func refresh() {
        let trusted = AXIsProcessTrusted()
        let listening = CGPreflightListenEventAccess()
        styleStatus(accessibilityStatus, allowed: trusted)
        styleStatus(inputStatus, allowed: listening)

        if !isPreview, trusted, listening {
            timer?.invalidate()
            timer = nil
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.45) { [weak self] in
                self?.window?.close()
                self?.onPermissionsGranted?()
            }
        }
    }

    private func styleStatus(_ label: NSTextField, allowed: Bool) {
        let text = allowed ? L("permission.allowed") : L("permission.notAllowed")
        label.stringValue = allowed ? "✓ \(text)" : "○ \(text)"
        label.textColor = allowed ? .systemGreen : .secondaryLabelColor
    }

    @objc private func applyLocalization() {
        window?.title = L("permission.windowTitle")
        buildContent()
        refresh()
    }

    @objc private func openAccessibilitySettings() {
        NSWorkspace.shared.open(accessibilityURL)
    }

    @objc private func openInputMonitoringSettings() {
        NSWorkspace.shared.open(inputMonitoringURL)
    }
}
