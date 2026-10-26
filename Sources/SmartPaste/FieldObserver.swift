import AppKit
import ApplicationServices
import Foundation

final class FieldObserver {
    private static let editableRoles: Set<String> = [
        kAXTextFieldRole as String,
        kAXTextAreaRole as String,
        kAXComboBoxRole as String
    ]

    private static let sensitiveWords = [
        "password", "passcode", "secure", "credit card", "card number",
        "cvv", "cvc", "security code", "one-time", "one time", "otp",
        "social security", "ssn"
    ]

    func current() -> FocusedField? {
        guard let element = focusedElement() else { return nil }

        let role = stringAttribute(element, kAXRoleAttribute) ?? ""
        guard Self.editableRoles.contains(role) else { return nil }

        let subrole = stringAttribute(element, kAXSubroleAttribute) ?? ""
        let value = stringAttribute(element, kAXValueAttribute) ?? ""

        var pieces: [String] = []
        for attribute in [
            kAXTitleAttribute,
            kAXDescriptionAttribute,
            kAXHelpAttribute,
            kAXPlaceholderValueAttribute
        ] {
            if let piece = stringAttribute(element, attribute),
               !piece.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                pieces.append(piece)
            }
        }
        if !subrole.isEmpty {
            pieces.append(subrole.lowercased())
        }

        var label = pieces.joined(separator: " · ")
        if label.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            label = L("field.textField")
        }

        let haystack = "\(role) \(label) \(value) \(subrole)".lowercased()
        if Self.sensitiveWords.contains(where: { haystack.contains($0) }) {
            return nil
        }

        guard let origin = pointAttribute(element, kAXPositionAttribute),
              let size = sizeAttribute(element, kAXSizeAttribute) else {
            return nil
        }

        let app = NSWorkspace.shared.frontmostApplication?.localizedName ?? "Unknown"

        return FocusedField(
            element: element,
            label: label,
            role: role,
            app: app,
            frame: CGRect(origin: origin, size: size),
            value: value
        )
    }

    private func focusedElement() -> AXUIElement? {
        let system = AXUIElementCreateSystemWide()
        if let element = attribute(system, kAXFocusedUIElementAttribute) {
            return (element as! AXUIElement)
        }

        guard let application = NSWorkspace.shared.frontmostApplication else { return nil }
        let appElement = AXUIElementCreateApplication(application.processIdentifier)
        if let element = attribute(appElement, kAXFocusedUIElementAttribute) {
            return (element as! AXUIElement)
        }
        return nil
    }

    private func attribute(_ element: AXUIElement, _ name: String) -> CFTypeRef? {
        var value: CFTypeRef?
        let result = AXUIElementCopyAttributeValue(element, name as CFString, &value)
        guard result == .success else { return nil }
        return value
    }

    private func stringAttribute(_ element: AXUIElement, _ name: String) -> String? {
        attribute(element, name) as? String
    }

    private func pointAttribute(_ element: AXUIElement, _ name: String) -> CGPoint? {
        guard let value = attribute(element, name), CFGetTypeID(value) == AXValueGetTypeID() else {
            return nil
        }
        var point = CGPoint.zero
        guard AXValueGetValue(value as! AXValue, .cgPoint, &point) else { return nil }
        return point
    }

    private func sizeAttribute(_ element: AXUIElement, _ name: String) -> CGSize? {
        guard let value = attribute(element, name), CFGetTypeID(value) == AXValueGetTypeID() else {
            return nil
        }
        var size = CGSize.zero
        guard AXValueGetValue(value as! AXValue, .cgSize, &size) else { return nil }
        return size
    }
}
