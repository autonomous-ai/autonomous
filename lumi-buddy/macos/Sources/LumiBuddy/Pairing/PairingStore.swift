import Foundation
import Security

struct PairingRecord: Codable, Equatable {
    let buddyID: String
    let lampHost: String
    let token: String
    let pairedAt: Date
}

enum PairingStoreError: LocalizedError {
    case keychain(OSStatus)
    case invalidData

    var errorDescription: String? {
        switch self {
        case .keychain(let status): return "Keychain error: \(status)"
        case .invalidData: return "Stored pairing record was unreadable"
        }
    }
}

final class PairingStore {
    private let service = "network.autonomous.ai.lumi-buddy"
    private let account = "default-buddy"

    func save(_ record: PairingRecord) throws {
        let data = try JSONEncoder().encode(record)
        let baseQuery: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(baseQuery as CFDictionary)

        var add = baseQuery
        add[kSecValueData as String] = data
        let status = SecItemAdd(add as CFDictionary, nil)
        guard status == errSecSuccess else { throw PairingStoreError.keychain(status) }
    }

    func load() throws -> PairingRecord? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        if status == errSecItemNotFound { return nil }
        guard status == errSecSuccess else { throw PairingStoreError.keychain(status) }
        guard let data = result as? Data else { throw PairingStoreError.invalidData }
        return try JSONDecoder().decode(PairingRecord.self, from: data)
    }

    func clear() throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        let status = SecItemDelete(query as CFDictionary)
        if status != errSecSuccess && status != errSecItemNotFound {
            throw PairingStoreError.keychain(status)
        }
    }
}
