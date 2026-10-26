import AppKit

final class TrayPanel: NSPanel {
    var scrollHandler: ((NSEvent) -> Void)?

    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { false }

    override func scrollWheel(with event: NSEvent) {
        if let scrollHandler {
            scrollHandler(event)
        } else {
            super.scrollWheel(with: event)
        }
    }
}

final class SnappingClipView: NSClipView {
    var rowStep: CGFloat = 0

    override func constrainBoundsRect(_ proposedBounds: NSRect) -> NSRect {
        var rect = super.constrainBoundsRect(proposedBounds)
        guard rowStep > 0 else { return rect }

        let maxY = max(0, (documentView?.frame.height ?? 0) - rect.height)
        let snapped = (rect.origin.y / rowStep).rounded() * rowStep
        rect.origin.y = min(max(snapped, 0), maxY)
        return rect
    }
}
