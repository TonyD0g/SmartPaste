import AppKit

final class VerticallyCenteredTextFieldCell: NSTextFieldCell {
    override func titleRect(forBounds rect: NSRect) -> NSRect {
        var frame = super.titleRect(forBounds: rect)
        let height = attributedStringValue
            .boundingRect(
                with: NSSize(width: frame.width, height: .greatestFiniteMagnitude),
                options: [.usesLineFragmentOrigin]
            )
            .height
        frame.origin.y = rect.midY - height / 2
        frame.size.height = height
        return frame
    }

    override func drawInterior(withFrame cellFrame: NSRect, in controlView: NSView) {
        super.drawInterior(withFrame: titleRect(forBounds: cellFrame), in: controlView)
    }
}
