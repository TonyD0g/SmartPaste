import AppKit
import QuartzCore

final class ClipCollectionItem: NSCollectionViewItem {
    static let identifier = NSUserInterfaceItemIdentifier("ClipCollectionItem")

    private let preview = NSTextField(wrappingLabelWithString: "")
    private let thumbnail = NSImageView()
    private let meta = NSTextField(labelWithString: "")

    override var isSelected: Bool {
        didSet { updateSelection() }
    }

    override func loadView() {
        let container = NSView()
        container.wantsLayer = true
        container.layer?.cornerRadius = 10
        container.layer?.borderWidth = 1
        view = container

        thumbnail.translatesAutoresizingMaskIntoConstraints = false
        thumbnail.imageScaling = .scaleProportionallyDown
        thumbnail.imageAlignment = .alignCenter
        thumbnail.wantsLayer = true
        thumbnail.layer?.cornerRadius = 8
        thumbnail.layer?.masksToBounds = true
        thumbnail.layer?.backgroundColor = NSColor.labelColor.withAlphaComponent(0.06).cgColor

        preview.translatesAutoresizingMaskIntoConstraints = false
        preview.font = .systemFont(ofSize: 13.5)
        preview.maximumNumberOfLines = 2
        preview.lineBreakMode = .byTruncatingTail
        preview.cell?.wraps = true
        preview.setContentCompressionResistancePriority(.defaultLow, for: .vertical)

        meta.translatesAutoresizingMaskIntoConstraints = false
        meta.font = .systemFont(ofSize: 11, weight: .regular)
        meta.textColor = .secondaryLabelColor
        meta.lineBreakMode = .byTruncatingTail

        container.addSubview(thumbnail)
        container.addSubview(preview)
        container.addSubview(meta)

        NSLayoutConstraint.activate([
            thumbnail.leadingAnchor.constraint(equalTo: container.leadingAnchor, constant: 12),
            thumbnail.centerYAnchor.constraint(equalTo: container.centerYAnchor),
            thumbnail.widthAnchor.constraint(equalToConstant: 44),
            thumbnail.heightAnchor.constraint(equalToConstant: 44),

            preview.leadingAnchor.constraint(equalTo: thumbnail.trailingAnchor, constant: 12),
            preview.trailingAnchor.constraint(equalTo: container.trailingAnchor, constant: -12),
            preview.topAnchor.constraint(equalTo: container.topAnchor, constant: 11),
            preview.bottomAnchor.constraint(lessThanOrEqualTo: meta.topAnchor, constant: -2),

            meta.leadingAnchor.constraint(equalTo: preview.leadingAnchor),
            meta.trailingAnchor.constraint(equalTo: preview.trailingAnchor),
            meta.bottomAnchor.constraint(equalTo: container.bottomAnchor, constant: -11)
        ])

        updateSelection()
    }

    func configure(clip: Clip) {
        let isImage = clip.imageFileName != nil
        thumbnail.imageScaling = isImage ? .scaleProportionallyUpOrDown : .scaleProportionallyDown

        if isImage {
            thumbnail.image = MemoryStore.shared.image(for: clip)
            thumbnail.contentTintColor = nil
            preview.stringValue = L("timeline.image")
            preview.textColor = .secondaryLabelColor
        } else {
            let configuration = NSImage.SymbolConfiguration(pointSize: 18, weight: .medium)
            thumbnail.image = NSImage(
                systemSymbolName: symbolName(for: clip.text),
                accessibilityDescription: nil
            )?.withSymbolConfiguration(configuration)
            thumbnail.contentTintColor = .secondaryLabelColor
            preview.stringValue = clip.text.replacingOccurrences(of: "\n", with: " ")
            preview.textColor = .labelColor
        }

        let formatter = RelativeDateTimeFormatter()
        let relative = formatter.localizedString(for: clip.createdAt, relativeTo: Date())
        var details = "\(relative) · \(clip.sourceApp)"
        if isImage, let width = clip.imagePixelWidth, let height = clip.imagePixelHeight {
            details += " · \(width)×\(height)"
        }
        meta.stringValue = details
    }

    private func symbolName(for text: String) -> String {
        let lower = text.lowercased()
        if lower.hasPrefix("http://") || lower.hasPrefix("https://") {
            return "link"
        }
        if lower.contains("@"), !lower.contains(" ") {
            return "envelope"
        }
        return "text.alignleft"
    }

    private func updateSelection() {
        guard let layer = view.layer else { return }

        CATransaction.begin()
        CATransaction.setAnimationDuration(
            NSWorkspace.shared.accessibilityDisplayShouldReduceMotion ? 0 : 0.14
        )
        CATransaction.setAnimationTimingFunction(CAMediaTimingFunction(name: .easeOut))

        layer.backgroundColor = isSelected
            ? NSColor.controlAccentColor.withAlphaComponent(0.10).cgColor
            : NSColor.labelColor.withAlphaComponent(0.03).cgColor
        layer.borderColor = isSelected
            ? NSColor.controlAccentColor.withAlphaComponent(0.55).cgColor
            : NSColor.labelColor.withAlphaComponent(0.08).cgColor
        layer.cornerRadius = 10

        CATransaction.commit()
    }
}
