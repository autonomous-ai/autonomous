import Foundation

enum PairingError: LocalizedError {
    case invalidCode
    case rejected(String)
    case network(String)
    case malformedResponse

    var errorDescription: String? {
        switch self {
        case .invalidCode: return "Invalid or expired pairing code"
        case .rejected(let s): return "Rejected by lamp: \(s)"
        case .network(let s): return "Network error: \(s)"
        case .malformedResponse: return "Malformed response from lamp"
        }
    }
}

final class PairingManager {
    private let store: PairingStore

    init(store: PairingStore) {
        self.store = store
    }

    func pair(lampHost: String, code: String) async throws -> PairingRecord {
        let host = normalizeHost(lampHost)
        guard let url = URL(string: "http://\(host)/api/buddy/pair/confirm") else {
            throw PairingError.network("invalid host: \(host)")
        }

        let body: [String: Any] = [
            "code": code,
            "name": deviceName(),
            "fingerprint": Self.stableFingerprint(),
            "os_version": ProcessInfo.processInfo.operatingSystemVersionString,
        ]
        let bodyData = try JSONSerialization.data(withJSONObject: body, options: [])

        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = bodyData
        req.timeoutInterval = 10

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await URLSession.shared.data(for: req)
        } catch {
            throw PairingError.network(error.localizedDescription)
        }

        guard let http = response as? HTTPURLResponse else {
            throw PairingError.malformedResponse
        }
        guard (200..<300).contains(http.statusCode) else {
            if http.statusCode == 400 || http.statusCode == 401 || http.statusCode == 410 {
                throw PairingError.invalidCode
            }
            let snippet = String(data: data, encoding: .utf8) ?? "<no body>"
            throw PairingError.rejected("HTTP \(http.statusCode): \(snippet)")
        }

        guard let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            throw PairingError.malformedResponse
        }
        // Accept both raw {token, buddy_id} and lumi envelope {status, data: {...}, message}
        let payload: [String: Any] = (json["data"] as? [String: Any]) ?? json
        guard let token = payload["token"] as? String,
              let buddyID = payload["buddy_id"] as? String else {
            throw PairingError.malformedResponse
        }

        let record = PairingRecord(buddyID: buddyID, lampHost: host, token: token, pairedAt: Date())
        try store.save(record)
        return record
    }

    func unpair() throws {
        try store.clear()
    }

    func current() -> PairingRecord? {
        return try? store.load()
    }

    // MARK: - helpers

    private func deviceName() -> String {
        if let n = Host.current().localizedName, !n.isEmpty { return n }
        return ProcessInfo.processInfo.hostName
    }

    private func normalizeHost(_ raw: String) -> String {
        var h = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if h.hasPrefix("http://") { h.removeFirst("http://".count) }
        if h.hasPrefix("https://") { h.removeFirst("https://".count) }
        if h.hasSuffix("/") { h.removeLast() }
        return h
    }

    private static func stableFingerprint() -> String {
        let key = "buddy.fingerprint"
        if let existing = UserDefaults.standard.string(forKey: key), !existing.isEmpty {
            return existing
        }
        let new = UUID().uuidString
        UserDefaults.standard.set(new, forKey: key)
        return new
    }
}
