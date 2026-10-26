import AppKit

final class WelcomeWindowController: NSWindowController, NSWindowDelegate {
    var onContinue: (() -> Void)?

    init() {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 480, height: 430),
            styleMask: [.titled, .closable, .fullSizeContentView],
            backing: .buffered,
            defer: false
        )
        window.title = L("welcome.windowTitle")
        window.titleVisibility = .hidden
        window.titlebarAppearsTransparent = true
        window.isMovableByWindowBackground = true
        window.isReleasedWhenClosed = false
        window.center()

        super.init(window: window)
        window.delegate = self
        buildPage()
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

    private func buildPage() {
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
        stack.alignment = .centerX
        stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        background.addSubview(stack)

        NSLayoutConstraint.activate([
            stack.centerXAnchor.constraint(equalTo: background.centerXAnchor),
            stack.topAnchor.constraint(equalTo: background.topAnchor, constant: 58),
            stack.leadingAnchor.constraint(
                greaterThanOrEqualTo: background.leadingAnchor,
                constant: 40
            ),
            stack.trailingAnchor.constraint(
                lessThanOrEqualTo: background.trailingAnchor,
                constant: -40
            ),
            stack.bottomAnchor.constraint(
                lessThanOrEqualTo: background.bottomAnchor,
                constant: -30
            )
        ])

        let icon = NSImageView(image: NSApp.applicationIconImage ?? NSImage())
        icon.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            icon.widthAnchor.constraint(equalToConstant: 96),
            icon.heightAnchor.constraint(equalToConstant: 96)
        ])
        stack.addArrangedSubview(icon)

        let title = NSTextField(labelWithString: L("welcome.title"))
        title.font = .systemFont(ofSize: 26, weight: .semibold)
        title.alignment = .center
        stack.addArrangedSubview(title)

        let shortcut = SmartPasteShortcut.current.displayKey
        let body = NSTextField(
            wrappingLabelWithString: L("welcome.body", shortcut, shortcut)
        )
        body.font = .systemFont(ofSize: 13)
        body.textColor = .secondaryLabelColor
        body.alignment = .center
        body.maximumNumberOfLines = 3
        body.translatesAutoresizingMaskIntoConstraints = false
        body.widthAnchor.constraint(equalToConstant: 340).isActive = true
        stack.addArrangedSubview(body)

        let keys = NSStackView()
        keys.orientation = .horizontal
        keys.alignment = .centerY
        keys.spacing = 6
        keys.addArrangedSubview(KeycapView("⌘"))
        keys.addArrangedSubview(KeycapView(String(shortcut.suffix(1))))
        stack.addArrangedSubview(keys)

        let spacer = NSView()
        spacer.translatesAutoresizingMaskIntoConstraints = false
        spacer.heightAnchor.constraint(equalToConstant: 2).isActive = true
        stack.addArrangedSubview(spacer)

        let button = NSButton(
            title: L("welcome.continue"),
            target: self,
            action: #selector(continueSetup)
        )
        button.bezelStyle = .rounded
        button.controlSize = .large
        button.keyEquivalent = "\r"
        button.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            button.widthAnchor.constraint(equalToConstant: 160),
            button.heightAnchor.constraint(equalToConstant: 34)
        ])
        stack.addArrangedSubview(button)
    }

    @objc private func applyLocalization() {
        window?.title = L("welcome.windowTitle")
        buildPage()
    }

    @objc private func continueSetup() {
        close()
        onContinue?()
    }

    func windowShouldClose(_ sender: NSWindow) -> Bool {
        true
    }
}
