import Foundation
import Security

final class TypeSafeKeychain {
    static let shared = TypeSafeKeychain()

    private let service: String
    private let account: String

    init() {
        let environment = ProcessInfo.processInfo.environment
        service = environment["SMARTPASTE_KEYCHAIN_SERVICE"] ?? "app.smartpaste.mac.typesafe"
        account = "direct-jev-api-key"
    }

    private var baseQuery: [String: Any] {
        [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]
    }

    func read() -> String? {
        var query = baseQuery
        query[kSecReturnData as String] = true
        query[kSecMatchLimit as String] = kSecMatchLimitOne

        var result: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &result) == errSecSuccess,
              let data = result as? Data,
              let value = String(data: data, encoding: .utf8),
              !value.isEmpty else {
            return nil
        }
        return value
    }

    @discardableResult
    func save(_ value: String) -> Bool {
        guard !value.isEmpty, let data = value.data(using: .utf8) else { return false }

        let attributes: [String: Any] = [
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        ]

        var status = SecItemUpdate(baseQuery as CFDictionary, attributes as CFDictionary)
        if status == errSecItemNotFound {
            var query = baseQuery
            attributes.forEach { query[$0.key] = $0.value }
            status = SecItemAdd(query as CFDictionary, nil)
        }
        return status == errSecSuccess
    }

    @discardableResult
    func remove() -> Bool {
        let status = SecItemDelete(baseQuery as CFDictionary)
        return status == errSecSuccess || status == errSecItemNotFound
    }
}
