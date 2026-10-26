import AppKit
import ApplicationServices
import Foundation

struct Clip: Codable, Identifiable, Equatable {
    let id: UUID
    var text: String
    var sourceApp: String
    var createdAt: Date
    var imageFileName: String?
    var imagePixelWidth: Int?
    var imagePixelHeight: Int?
    var indexedValues: [String]?
}

struct FocusedField {
    var element: AXUIElement?
    var label: String
    var role: String
    var app: String
    var frame: CGRect
    var value: String
}

struct CandidateValue {
    let ID: String
    let text: String
    let sourceApp: String
    let copiedAt: String
}

struct DirectJevResponse: Codable {
    struct Answer: Codable {
        let choice: String?
        let confidence: Double?
        let probabilities: [String: Double]?
    }

    struct Answers: Codable {
        let answer: Answer?
    }

    let answers: Answers?
}

enum SuggestionOutcome {
    case match(clip: Clip, confidence: Double, rankedTexts: [String])
    case noMatch
    case failure
}

enum SmartPasteShortcut: String {
    case commandV
    case commandJ

    private static let key = "smartpaste.smartPasteShortcut"

    static var current: SmartPasteShortcut {
        get {
            guard let raw = UserDefaults.standard.string(forKey: key),
                  let shortcut = SmartPasteShortcut(rawValue: raw) else {
                return .commandV
            }
            return shortcut
        }
        set {
            UserDefaults.standard.set(newValue.rawValue, forKey: key)
        }
    }

    var displayKey: String {
        switch self {
        case .commandV: return "⌘V"
        case .commandJ: return "⌘J"
        }
    }

    var keyCode: CGKeyCode {
        switch self {
        case .commandV: return 9
        case .commandJ: return 38
        }
    }
}

extension Notification.Name {
    static let clipsChanged = Notification.Name("SmartPasteClipsChanged")
}
