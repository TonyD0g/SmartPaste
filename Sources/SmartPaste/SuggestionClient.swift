import Foundation

final class SuggestionClient {
    static let noMatchCandidate = "__no_good_match__"

    private let iso = ISO8601DateFormatter()

    private let directEndpoint = URL(string: "https://api.typesafe.ai/v1/systemone")!

    func suggest(
        field: FocusedField,
        values: [CandidateValue],
        completion: @escaping (SuggestionOutcome) -> Void
    ) {
        sendDirect(field: field, values: values, completion: completion)
    }

    func candidateValues(from clips: [Clip]) -> [CandidateValue] {
        var result: [CandidateValue] = []
        var seen = Set<String>()
        var totalLength = 0

        for clip in clips where clip.imageFileName == nil {
            let values = clip.indexedValues ?? MemoryStore.indexValues(from: clip.text)
            for (index, value) in values.enumerated() {
                let key = value.lowercased()
                guard seen.insert(key).inserted else { continue }
                guard result.count < 180, totalLength + value.count <= 24_000 else {
                    return result
                }
                totalLength += value.count
                result.append(CandidateValue(
                    ID: "\(clip.id.uuidString):index:\(index)",
                    text: value,
                    sourceApp: clip.sourceApp,
                    copiedAt: iso.string(from: clip.createdAt)
                ))
            }
        }
        return result
    }

    private func sendDirect(
        field: FocusedField,
        values: [CandidateValue],
        completion: @escaping (SuggestionOutcome) -> Void
    ) {
        guard let apiKey = TypeSafeKeychain.shared.read(), !apiKey.isEmpty else {
            completion(.failure)
            return
        }

        var criteria: [String: String] = [:]
        for value in values {
            criteria[value.text] = value.ID
        }
        criteria[Self.noMatchCandidate] =
            "None of the saved clipboard values is an appropriate answer for this field."

        let question: [String: Any] = [
            "model": "jev-latest",
            "state": "focused_field",
            "label": field.label,
            "role": field.role,
            "application": field.app,
            "page_url": NSNull(),
            "candidates": values.map { value in
                [
                    "id": value.ID,
                    "text": value.text,
                    "sourceApp": value.sourceApp,
                    "copiedAt": value.copiedAt
                ]
            },
            "policy": "Suggest only a value that naturally and specifically answers the "
                + "focused field. Prefer no match over a merely plausible or unrelated value. "
                + "Never transform, combine, or invent text."
        ]

        let body: [String: Any] = [
            "suggestion": [
                "type": "choice",
                "instructions": "Which one saved clipboard value is the best exact value to "
                    + "enter in `focused_field`? Choose its candidate id. Choose the no-match "
                    + "option when none clearly belongs in this field.",
                "criteria": criteria
            ],
            "questions": [question]
        ]

        guard JSONSerialization.isValidJSONObject(body),
              let data = try? JSONSerialization.data(withJSONObject: body) else {
            completion(.failure)
            return
        }

        var request = URLRequest(url: directEndpoint)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
        request.httpBody = data
        request.timeoutInterval = 12

        URLSession.shared.dataTask(with: request) { [weak self] data, response, _ in
            guard let self else { return }
            guard let http = response as? HTTPURLResponse,
                  http.statusCode == 200,
                  let data,
                  let decoded = try? JSONDecoder().decode(DirectJevResponse.self, from: data),
                  let answer = decoded.answers?.answer,
                  let choice = answer.choice,
                  !choice.isEmpty,
                  choice != Self.noMatchCandidate else {
                DispatchQueue.main.async { completion(.noMatch) }
                return
            }

            let probabilities = answer.probabilities ?? [:]
            let ranked = probabilities
                .sorted { $0.value > $1.value }
                .map(\.key)
            let confidence = probabilities[choice] ?? answer.confidence ?? 0

            self.finish(
                id: choice,
                confidence: confidence,
                rankedIds: ranked.isEmpty ? [choice] : ranked,
                values: values,
                completion: completion
            )
        }.resume()
    }

    private func finish(
        id: String,
        confidence: Double,
        rankedIds: [String],
        values: [CandidateValue],
        completion: @escaping (SuggestionOutcome) -> Void
    ) {
        guard id != Self.noMatchCandidate,
              let value = values.first(where: { $0.ID == id }) else {
            DispatchQueue.main.async { completion(.noMatch) }
            return
        }

        let textByID = Dictionary(uniqueKeysWithValues: values.map { ($0.ID, $0.text) })
        var rankedTexts = rankedIds.compactMap { textByID[$0] }
        if !rankedTexts.contains(value.text) {
            rankedTexts.insert(value.text, at: 0)
        }

        let clip = Clip(
            id: UUID(),
            text: value.text,
            sourceApp: value.sourceApp,
            createdAt: Date(),
            imageFileName: nil,
            imagePixelWidth: nil,
            imagePixelHeight: nil,
            indexedValues: []
        )

        DispatchQueue.main.async {
            completion(.match(clip: clip, confidence: confidence, rankedTexts: rankedTexts))
        }
    }
}
