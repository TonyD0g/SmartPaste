import AppKit
import Foundation

final class MemoryStore {
    static let shared = MemoryStore()

    private(set) var clips: [Clip] = []

    private let queue = DispatchQueue(label: "memory.initializing", qos: .utility)
    private let directory: URL
    private let imagesDirectory: URL
    private let file: URL

    private let maximumInputBytes = 25 * 1024 * 1024
    private let maximumPixelDimension = 12_000
    private let maximumPixelCount = 40_000_000
    private let storedPixelDimension = 4_096

    private init() {
        let environment = ProcessInfo.processInfo.environment
        if let override = environment["SMARTPASTE_DATA_DIRECTORY"], !override.isEmpty {
            directory = URL(fileURLWithPath: override, isDirectory: true)
        } else {
            directory = FileManager.default.homeDirectoryForCurrentUser
                .appendingPathComponent("Library/Application Support/Smart Paste", isDirectory: true)
        }
        imagesDirectory = directory.appendingPathComponent("Images", isDirectory: true)
        file = directory.appendingPathComponent("clips.json")

        try? FileManager.default.createDirectory(
            at: directory,
            withIntermediateDirectories: true,
            attributes: nil
        )
        try? FileManager.default.createDirectory(
            at: imagesDirectory,
            withIntermediateDirectories: true,
            attributes: nil
        )

        if let data = try? Data(contentsOf: file),
           let decoded = try? JSONDecoder().decode([Clip].self, from: data) {
            clips = decoded.filter { clip in
                guard let name = clip.imageFileName else { return true }
                let url = imagesDirectory.appendingPathComponent(name)
                return FileManager.default.fileExists(atPath: url.path)
            }
            if clips.count != decoded.count {
                save()
            }
        }
    }

    var textClips: [Clip] {
        clips.filter { $0.imageFileName == nil }
    }

    func add(text rawText: String, sourceApp: String) {
        let text = rawText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !text.isEmpty, text.count <= 20_000, !Self.looksSensitive(text) else { return }

        if let newest = clips.first, newest.text == text {
            return
        }

        let values = Self.indexValues(from: text)
        let createdAt = Date()
        let clip = Clip(
            id: UUID(),
            text: text,
            sourceApp: sourceApp,
            createdAt: createdAt,
            imageFileName: nil,
            imagePixelWidth: nil,
            imagePixelHeight: nil,
            indexedValues: values
        )

        var inserted = [clip]
        inserted.append(contentsOf: values.filter { $0 != text }.map { value in
            Clip(
                id: UUID(),
                text: value,
                sourceApp: sourceApp,
                createdAt: createdAt,
                imageFileName: nil,
                imagePixelWidth: nil,
                imagePixelHeight: nil,
                indexedValues: [value]
            )
        })

        clips.insert(contentsOf: inserted, at: 0)
        save()
        NotificationCenter.default.post(name: .clipsChanged, object: nil)
    }

    func add(imageData: Data, sourceApp: String) {
        guard !imageData.isEmpty, imageData.count <= maximumInputBytes else { return }
        guard let image = NSImage(data: imageData) else { return }

        let representations = image.representations
        guard !representations.isEmpty else { return }

        var best = representations[0]
        var bestArea = max(best.pixelsWide, 0) * max(best.pixelsHigh, 0)
        for rep in representations.dropFirst() {
            let area = max(rep.pixelsWide, 0) * max(rep.pixelsHigh, 0)
            if area > bestArea {
                best = rep
                bestArea = area
            }
        }

        let width = best.pixelsWide
        let height = best.pixelsHigh
        guard width >= 1, height >= 1,
              width <= maximumPixelDimension, height <= maximumPixelDimension,
              width * height <= maximumPixelCount else { return }

        let factor = min(Double(storedPixelDimension) / Double(max(width, height)), 1.0)
        let scaledWidth = max(Int(factor * Double(width)), 1)
        let scaledHeight = max(Int(factor * Double(height)), 1)

        guard let bitmap = NSBitmapImageRep(
            bitmapDataPlanes: nil,
            pixelsWide: scaledWidth,
            pixelsHigh: scaledHeight,
            bitsPerSample: 8,
            samplesPerPixel: 4,
            hasAlpha: true,
            isPlanar: false,
            colorSpaceName: .deviceRGB,
            bytesPerRow: 0,
            bitsPerPixel: 0
        ) else { return }

        bitmap.size = NSSize(width: scaledWidth, height: scaledHeight)

        NSGraphicsContext.saveGraphicsState()
        if let context = NSGraphicsContext(bitmapImageRep: bitmap) {
            NSGraphicsContext.current = context
            image.draw(
                in: NSRect(x: 0, y: 0, width: scaledWidth, height: scaledHeight),
                from: .zero,
                operation: .copy,
                fraction: 1.0
            )
        }
        NSGraphicsContext.restoreGraphicsState()

        guard let png = bitmap.representation(using: .png, properties: [:]),
              png.count <= maximumInputBytes else { return }

        let name = UUID().uuidString.lowercased() + ".png"
        let destination = imagesDirectory.appendingPathComponent(name)
        do {
            try png.write(to: destination, options: .atomic)
        } catch {
            return
        }

        let clip = Clip(
            id: UUID(),
            text: "",
            sourceApp: sourceApp,
            createdAt: Date(),
            imageFileName: name,
            imagePixelWidth: scaledWidth,
            imagePixelHeight: scaledHeight,
            indexedValues: []
        )

        clips.insert(clip, at: 0)
        save()
        NotificationCenter.default.post(name: .clipsChanged, object: nil)
    }

    func image(for clip: Clip) -> NSImage? {
        guard let name = clip.imageFileName else { return nil }
        let url = imagesDirectory.appendingPathComponent(name)
        guard url.lastPathComponent == name else { return nil }
        return NSImage(contentsOf: url)
    }

    func remove(_ clipsToRemove: [Clip]) {
        guard !clipsToRemove.isEmpty else { return }
        let ids = Set(clipsToRemove.map(\.id))
        clips.removeAll { ids.contains($0.id) }

        for clip in clipsToRemove {
            guard let name = clip.imageFileName else { continue }
            let url = imagesDirectory.appendingPathComponent(name)
            guard url.lastPathComponent == name else { continue }
            try? FileManager.default.removeItem(at: url)
        }

        save()
        NotificationCenter.default.post(name: .clipsChanged, object: nil)
    }

    private func save() {
        let snapshot = clips
        let destination = file
        queue.async {
            guard let data = try? JSONEncoder().encode(snapshot) else { return }
            try? data.write(to: destination, options: .atomic)
        }
    }

    static func looksSensitive(_ text: String) -> Bool {
        let compact = text
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: "-", with: "")
        if compact.range(of: "^\\d{13,19}$", options: .regularExpression) != nil {
            return true
        }

        let lower = text.lowercased()
        if lower.range(of: "\\b\\d{3}-\\d{2}-\\d{4}\\b", options: .regularExpression) != nil {
            return true
        }
        return lower.contains("begin private key")
    }

    static func indexValues(from text: String) -> [String] {
        var values: [String] = []
        var seen = Set<String>()
        var totalLength = 0

        func add(_ raw: String) {
            let value = raw.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !value.isEmpty, value.count <= 500, !looksSensitive(value) else { return }
            guard seen.insert(value.lowercased()).inserted else { return }
            guard values.count < 180, totalLength + value.count <= 24_000 else { return }
            totalLength += value.count
            values.append(value)
        }

        if text.count < 501 {
            add(text)
        }

        let patterns = [
            "https?://[^\\s]+",
            "[A-Z0-9._%+-]+@[A-Z0-9.-]+\\.[A-Z]{2,}",
            "(?:\\+?\\d[\\d ().-]{7,}\\d)"
        ]

        let source = text as NSString
        for pattern in patterns {
            guard let expression = try? NSRegularExpression(
                pattern: pattern,
                options: [.caseInsensitive]
            ) else { continue }
            let range = NSRange(location: 0, length: source.length)
            let matches = expression.matches(in: text, options: [], range: range)
            for match in matches.prefix(20) {
                add(source.substring(with: match.range))
            }
        }

        let lines = text
            .components(separatedBy: .newlines)
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
        for line in lines.prefix(30) {
            add(line)
        }

        return values
    }
}
