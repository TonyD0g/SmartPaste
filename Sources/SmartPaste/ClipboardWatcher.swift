import AppKit
import Foundation

final class ClipboardWatcher {
    private let pasteboard = NSPasteboard.general
    var lastChange: Int
    private var timer: Timer?

    private static let pausedKey = "smartpaste.clipboardCapturePaused"

    init() {
        lastChange = pasteboard.changeCount
    }

    deinit {
        timer?.invalidate()
    }

    func start() {
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 0.6, repeats: true) { [weak self] _ in
            self?.captureIfChanged()
        }
    }

    @discardableResult
    func captureIfChanged() -> Bool {
        let changeCount = pasteboard.changeCount
        guard changeCount != lastChange else { return false }
        lastChange = changeCount

        guard !UserDefaults.standard.bool(forKey: Self.pausedKey) else { return false }

        let sourceApp = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"

        if let png = pasteboard.data(forType: .png) {
            MemoryStore.shared.add(imageData: png, sourceApp: sourceApp)
            return true
        }
        if let tiff = pasteboard.data(forType: .tiff) {
            MemoryStore.shared.add(imageData: tiff, sourceApp: sourceApp)
            return true
        }
        if let image = NSImage(pasteboard: pasteboard),
           let tiff = image.tiffRepresentation {
            MemoryStore.shared.add(imageData: tiff, sourceApp: sourceApp)
            return true
        }
        if let string = pasteboard.string(forType: .string) {
            MemoryStore.shared.add(text: string, sourceApp: sourceApp)
            return true
        }
        return false
    }
}
