import AppKit
import Foundation
import QuartzCore

final class TimelineController: NSWindowController, NSCollectionViewDataSource,
    NSCollectionViewDelegateFlowLayout, NSSearchFieldDelegate, NSMenuDelegate,
    NSWindowDelegate {

    private let panel: TrayPanel
    private let collection = NSCollectionView()
    private let flowLayout = NSCollectionViewFlowLayout()
    private let scroll = NSScrollView()
    private let search = NSSearchField()
    private let titleLabel = NSTextField(labelWithString: "Smart Paste")
    private let badge = NSTextField(labelWithString: "BETA")
    private let emptyLabel = NSTextField(labelWithString: "")
    private let separator = NSBox()

    private(set) var results: [Clip] = []
    private var keyMonitor: Any?
    private(set) var selectedIndex = 0
    private var previousApp: NSRunningApplication?
    private var deleteMenuItem: NSMenuItem?
    var rankedTexts: [String] = []
    var onPaste: ((Clip, NSRunningApplication?) -> Void)?

    private static let defaultWidth: CGFloat = 560
    private static let minimumWidth: CGFloat = 440
    private static let widthKey = "smartpaste.timelineWidth"
    private static let rowHeight: CGFloat = 74
    private static let rowGap: CGFloat = 8
    private static let panelHeight: CGFloat = 313

    init() {
        panel = TrayPanel(
            contentRect: NSRect(
                origin: .zero,
                size: NSSize(width: Self.defaultWidth, height: Self.panelHeight)
            ),
            styleMask: [.nonactivatingPanel, .resizable],
            backing: .buffered,
            defer: false
        )
        super.init(window: panel)

        panel.level = NSWindow.Level(rawValue: 101)
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.minSize = NSSize(width: Self.minimumWidth, height: Self.panelHeight)
        panel.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: Self.panelHeight)
        panel.collectionBehavior = [.canJoinAllSpaces, .stationary, .fullScreenAuxiliary]
        panel.delegate = self

        buildContent()
        panel.scrollHandler = { [weak self] event in
            self?.scroll.scrollWheel(with: event)
        }
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(reload),
            name: .clipsChanged,
            object: nil
        )
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
        if let keyMonitor {
            NSEvent.removeMonitor(keyMonitor)
        }
    }

    private func buildContent() {
        let background = NSVisualEffectView()
        background.material = .hudWindow
        background.blendingMode = .behindWindow
        background.state = .active
        background.wantsLayer = true
        background.layer?.cornerRadius = 18
        background.layer?.masksToBounds = true

        let header = NSView()
        header.translatesAutoresizingMaskIntoConstraints = false

        titleLabel.translatesAutoresizingMaskIntoConstraints = false
        titleLabel.font = .systemFont(ofSize: 17, weight: .semibold)

        badge.translatesAutoresizingMaskIntoConstraints = false
        badge.font = .systemFont(ofSize: 9, weight: .semibold)
        badge.textColor = .secondaryLabelColor
        badge.alignment = .center
        badge.wantsLayer = true
        badge.layer?.cornerRadius = 5
        badge.layer?.borderWidth = 1
        badge.layer?.borderColor = NSColor.separatorColor.cgColor
        badge.layer?.backgroundColor = NSColor.labelColor.withAlphaComponent(0.04).cgColor

        search.translatesAutoresizingMaskIntoConstraints = false
        search.placeholderString = L("timeline.search")
        search.sendsSearchStringImmediately = true
        search.sendsWholeSearchString = false
        search.delegate = self
        search.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)

        separator.boxType = .separator
        separator.translatesAutoresizingMaskIntoConstraints = false

        header.addSubview(titleLabel)
        header.addSubview(badge)
        header.addSubview(search)

        NSLayoutConstraint.activate([
            header.heightAnchor.constraint(equalToConstant: 56),

            titleLabel.leadingAnchor.constraint(equalTo: header.leadingAnchor, constant: 18),
            titleLabel.centerYAnchor.constraint(equalTo: header.centerYAnchor),

            badge.leadingAnchor.constraint(equalTo: titleLabel.trailingAnchor, constant: 8),
            badge.centerYAnchor.constraint(equalTo: titleLabel.centerYAnchor),
            badge.widthAnchor.constraint(equalToConstant: 38),
            badge.heightAnchor.constraint(equalToConstant: 17),

            search.trailingAnchor.constraint(equalTo: header.trailingAnchor, constant: -14),
            search.centerYAnchor.constraint(equalTo: header.centerYAnchor),
            search.widthAnchor.constraint(greaterThanOrEqualToConstant: 140),
            search.widthAnchor.constraint(lessThanOrEqualToConstant: 220),
            search.widthAnchor.constraint(equalTo: header.widthAnchor, multiplier: 0.38),
            search.leadingAnchor.constraint(
                greaterThanOrEqualTo: badge.trailingAnchor,
                constant: 12
            )
        ])

        flowLayout.scrollDirection = .vertical
        flowLayout.minimumInteritemSpacing = 0
        flowLayout.minimumLineSpacing = Self.rowGap
        flowLayout.sectionInset = NSEdgeInsets(top: 8, left: 12, bottom: 8, right: 12)

        collection.isSelectable = true
        collection.allowsEmptySelection = true
        collection.backgroundColors = [.clear]
        collection.autoresizingMask = [.width]

        let menu = NSMenu(title: "")
        menu.delegate = self
        let deleteItem = NSMenuItem(
            title: L("timeline.delete"),
            action: #selector(deleteContextItem(_:)),
            keyEquivalent: ""
        )
        deleteItem.target = self
        menu.addItem(deleteItem)
        collection.menu = menu
        deleteMenuItem = deleteItem

        let clipView = SnappingClipView()
        clipView.rowStep = Self.rowHeight + Self.rowGap
        scroll.contentView = clipView
        scroll.translatesAutoresizingMaskIntoConstraints = false
        scroll.drawsBackground = false
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = false
        scroll.autohidesScrollers = true
        scroll.scrollerStyle = .overlay
        scroll.verticalScrollElasticity = .automatic

        emptyLabel.translatesAutoresizingMaskIntoConstraints = false
        emptyLabel.font = .systemFont(ofSize: 13)
        emptyLabel.textColor = .secondaryLabelColor
        emptyLabel.alignment = .center
        emptyLabel.maximumNumberOfLines = 2
        emptyLabel.isHidden = true

        background.addSubview(header)
        background.addSubview(separator)
        background.addSubview(scroll)
        background.addSubview(emptyLabel)

        NSLayoutConstraint.activate([
            header.topAnchor.constraint(equalTo: background.topAnchor),
            header.leadingAnchor.constraint(equalTo: background.leadingAnchor),
            header.trailingAnchor.constraint(equalTo: background.trailingAnchor),

            separator.topAnchor.constraint(equalTo: header.bottomAnchor),
            separator.leadingAnchor.constraint(equalTo: background.leadingAnchor, constant: 16),
            separator.trailingAnchor.constraint(equalTo: background.trailingAnchor, constant: -16),
            separator.heightAnchor.constraint(equalToConstant: 1),

            scroll.topAnchor.constraint(equalTo: separator.bottomAnchor, constant: 2),
            scroll.leadingAnchor.constraint(equalTo: background.leadingAnchor),
            scroll.trailingAnchor.constraint(equalTo: background.trailingAnchor),
            scroll.bottomAnchor.constraint(equalTo: background.bottomAnchor),

            emptyLabel.centerXAnchor.constraint(equalTo: scroll.centerXAnchor),
            emptyLabel.centerYAnchor.constraint(equalTo: scroll.centerYAnchor, constant: 6),
            emptyLabel.widthAnchor.constraint(lessThanOrEqualToConstant: 320)
        ])

        panel.contentView = background
    }

    @objc private func applyLocalization() {
        search.placeholderString = L("timeline.search")
        deleteMenuItem?.title = L("timeline.delete")
        updateEmptyState()
    }

    private func updateEmptyState() {
        emptyLabel.isHidden = !results.isEmpty
        emptyLabel.stringValue = search.stringValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .isEmpty
            ? L("timeline.empty")
            : L("timeline.noResults")
    }

    @objc func reload() {
        let previousID: UUID? = {
            guard selectedIndex >= 0, selectedIndex < results.count else { return nil }
            return results[selectedIndex].id
        }()

        let query = search.stringValue.trimmingCharacters(in: .whitespacesAndNewlines)
        var rankByText: [String: Int] = [:]
        for (index, text) in rankedTexts.enumerated() {
            rankByText[text.lowercased()] = index
        }

        let clips = MemoryStore.shared.clips
        let sorted = clips.enumerated().sorted { lhs, rhs in
            let lRank = rankByText[lhs.element.text.lowercased()]
            let rRank = rankByText[rhs.element.text.lowercased()]
            switch (lRank, rRank) {
            case let (l?, r?):
                return l < r
            case (_?, nil):
                return true
            case (nil, _?):
                return false
            case (nil, nil):
                return lhs.offset < rhs.offset
            }
        }.map(\.element)

        results = sorted.filter { clip in
            query.isEmpty || clip.text.localizedCaseInsensitiveContains(query)
        }

        if collection.bounds.width > 1 {
            collection.reloadData()
        }
        updateEmptyState()
        if let previousID,
           let restored = results.firstIndex(where: { $0.id == previousID }) {
            selectedIndex = restored
        } else {
            selectedIndex = min(selectedIndex, max(results.count - 1, 0))
        }
        if collection.dataSource != nil {
            selectCurrent()
        }
    }

    private func selectCurrent() {
        guard !results.isEmpty else {
            collection.deselectAll(nil)
            return
        }
        selectedIndex = min(max(selectedIndex, 0), results.count - 1)
        let path = IndexPath(item: selectedIndex, section: 0)
        collection.selectItems(at: [path], scrollPosition: .nearestVerticalEdge)

        if let item = collection.item(at: path) as? ClipCollectionItem, !item.isSelected {
            item.isSelected = true
        }
    }

    @objc private func selectCurrentAction() {
        selectCurrent()
    }

    func show() {
        previousApp = NSWorkspace.shared.frontmostApplication

        search.stringValue = ""
        selectedIndex = 0
        reload()

        let mouse = NSEvent.mouseLocation
        let screen = NSScreen.screens.first(where: { $0.frame.contains(mouse) }) ?? NSScreen.main

        if let screen {
            let width = min(restoredWidth(), screen.visibleFrame.width - 48)
            let height = Self.panelHeight
            let finalFrame = NSRect(
                x: screen.frame.midX - width / 2,
                y: screen.visibleFrame.maxY - height - 20,
                width: width,
                height: height
            )
            panel.maxSize = NSSize(
                width: screen.visibleFrame.width,
                height: Self.panelHeight
            )

            let reduceMotion = NSWorkspace.shared.accessibilityDisplayShouldReduceMotion
            if reduceMotion {
                panel.setFrame(finalFrame, display: true)
                panel.alphaValue = 1
            } else {
                panel.setFrame(finalFrame.offsetBy(dx: 0, dy: 18), display: true)
                panel.alphaValue = 0
            }
        }

        panel.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        panel.makeFirstResponder(search)
        installKeyMonitor()

        panel.contentView?.layoutSubtreeIfNeeded()
        if scroll.documentView == nil {
            collection.frame = NSRect(origin: .zero, size: scroll.contentSize)
            scroll.documentView = collection
            collection.collectionViewLayout = flowLayout
            collection.register(
                ClipCollectionItem.self,
                forItemWithIdentifier: ClipCollectionItem.identifier
            )
            collection.dataSource = self
            collection.delegate = self
        }
        updateCollectionWidth()
        collection.reloadData()
        selectCurrent()

        if !NSWorkspace.shared.accessibilityDisplayShouldReduceMotion {
            NSAnimationContext.runAnimationGroup { context in
                context.duration = 0.18
                context.timingFunction = CAMediaTimingFunction(name: .easeOut)
                panel.animator().alphaValue = 1
                if let screen {
                    let width = panel.frame.width
                    let height = panel.frame.height
                    panel.animator().setFrame(
                        NSRect(
                            x: screen.frame.midX - width / 2,
                            y: screen.visibleFrame.maxY - height - 20,
                            width: width,
                            height: height
                        ),
                        display: true
                    )
                }
            }
        }
    }

    func hide() {
        removeKeyMonitor()

        if NSWorkspace.shared.accessibilityDisplayShouldReduceMotion {
            panel.orderOut(nil)
            return
        }

        NSAnimationContext.runAnimationGroup({ context in
            context.duration = 0.12
            context.timingFunction = CAMediaTimingFunction(name: .easeIn)
            panel.animator().alphaValue = 0
        }, completionHandler: { [weak self] in
            self?.panel.orderOut(nil)
            self?.panel.alphaValue = 1
        })
    }

    private func restoredWidth() -> CGFloat {
        let width = UserDefaults.standard.double(forKey: Self.widthKey)
        guard width > 0 else { return Self.defaultWidth }
        return max(width, Self.minimumWidth)
    }

    private func saveWidth() {
        UserDefaults.standard.set(panel.frame.width, forKey: Self.widthKey)
    }

    private func updateCollectionWidth() {
        let width = scroll.contentSize.width
        guard width > 0 else { return }
        if abs(collection.frame.width - width) > 0.5 {
            collection.frame.size.width = width
        }
        collection.collectionViewLayout?.invalidateLayout()
    }

    func windowDidResize(_ notification: Notification) {
        updateCollectionWidth()
    }

    func windowDidEndLiveResize(_ notification: Notification) {
        saveWidth()
    }

    func windowWillResize(_ sender: NSWindow, to frameSize: NSSize) -> NSSize {
        NSSize(width: frameSize.width, height: Self.panelHeight)
    }

    private func installKeyMonitor() {
        removeKeyMonitor()
        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            guard let self else { return event }
            return self.handleKey(event) ? nil : event
        }
    }

    private func removeKeyMonitor() {
        if let keyMonitor {
            NSEvent.removeMonitor(keyMonitor)
            self.keyMonitor = nil
        }
    }

    private func handleKey(_ event: NSEvent) -> Bool {
        switch event.keyCode {
        case 53:
            hide()
            return true
        case 36, 76:
            pasteSelected()
            return true
        case 51, 117:
            if !search.stringValue.isEmpty { return false }
            deleteSelected()
            return true
        case 126, 123:
            if search.stringValue.isEmpty == false, event.keyCode == 123 { return false }
            selectedIndex = max(selectedIndex - 1, 0)
            selectCurrent()
            return true
        case 125, 124:
            if search.stringValue.isEmpty == false, event.keyCode == 124 { return false }
            selectedIndex = min(selectedIndex + 1, max(results.count - 1, 0))
            selectCurrent()
            return true
        default:
            return false
        }
    }

    func pasteSelected() {
        guard selectedIndex >= 0, selectedIndex < results.count else { return }
        let clip = results[selectedIndex]
        panel.orderOut(nil)
        removeKeyMonitor()

        let app = previousApp
        onPaste?(clip, app)
    }

    @objc func deleteSelected() {
        guard selectedIndex >= 0, selectedIndex < results.count else {
            NSSound.beep()
            return
        }
        let clip = results[selectedIndex]
        MemoryStore.shared.remove([clip])
        selectedIndex = min(selectedIndex, max(results.count - 1, 0))
        reload()
    }

    @objc func deleteContextItem(_ sender: NSMenuItem) {
        deleteSelected()
    }

    func controlTextDidChange(_ obj: Notification) {
        selectedIndex = 0
        reload()
    }

    func menuNeedsUpdate(_ menu: NSMenu) {
        let point = panel.convertPoint(fromScreen: NSEvent.mouseLocation)
        let local = collection.convert(point, from: nil)
        if let path = collection.indexPathForItem(at: local) {
            selectedIndex = path.item
            selectCurrent()
        }
        menu.items.first?.isEnabled =
            selectedIndex >= 0 && selectedIndex < results.count
    }

    func numberOfSections(in collectionView: NSCollectionView) -> Int { 1 }

    func collectionView(
        _ collectionView: NSCollectionView,
        numberOfItemsInSection section: Int
    ) -> Int {
        results.count
    }

    func collectionView(
        _ collectionView: NSCollectionView,
        layout collectionViewLayout: NSCollectionViewLayout,
        sizeForItemAt indexPath: IndexPath
    ) -> NSSize {
        let available = collectionView.enclosingScrollView?.contentSize.width
            ?? collectionView.bounds.width
        return NSSize(
            width: max(available - 44, 200),
            height: Self.rowHeight
        )
    }

    func collectionView(
        _ collectionView: NSCollectionView,
        itemForRepresentedObjectAt indexPath: IndexPath
    ) -> NSCollectionViewItem {
        let item = collectionView.makeItem(
            withIdentifier: ClipCollectionItem.identifier,
            for: indexPath
        )
        if let item = item as? ClipCollectionItem, indexPath.item < results.count {
            item.configure(clip: results[indexPath.item])
        }
        return item
    }

    func collectionView(
        _ collectionView: NSCollectionView,
        didSelectItemsAt indexPaths: Set<IndexPath>
    ) {
        guard let path = indexPaths.first else { return }
        selectedIndex = path.item
    }
}
