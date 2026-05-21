import Foundation

enum PairingStatus: Equatable {
    case notPaired
    case paired(buddyID: String, lampHost: String)
}

enum ConnectionStatus: Equatable {
    case disconnected
    case connecting
    case connected
    case error(String)
}

struct CommandRecord {
    let id: String
    let action: String
    let ok: Bool
    let timestamp: Date
}

final class AppState {
    static let shared = AppState()

    private(set) var pairing: PairingStatus = .notPaired { didSet { notify() } }
    private(set) var connection: ConnectionStatus = .disconnected { didSet { notify() } }
    private(set) var discoveredLamps: [LampInfo] = [] { didSet { notify() } }
    private(set) var paused: Bool = false { didSet { notify() } }
    private(set) var lastCommand: CommandRecord? = nil { didSet { notify() } }

    var onChange: (() -> Void)?

    private init() {}

    func setPairing(_ status: PairingStatus) { onMain { self.pairing = status } }
    func setConnection(_ status: ConnectionStatus) { onMain { self.connection = status } }
    func setDiscoveredLamps(_ lamps: [LampInfo]) { onMain { self.discoveredLamps = lamps } }
    func setPaused(_ paused: Bool) { onMain { self.paused = paused } }
    func recordCommand(_ record: CommandRecord) { onMain { self.lastCommand = record } }

    private func notify() {
        // didSet runs on whichever thread the setter ran. setPairing etc. always hop to main first,
        // so onChange always fires on main.
        onChange?()
    }

    private func onMain(_ block: @escaping () -> Void) {
        if Thread.isMainThread { block() }
        else { DispatchQueue.main.async(execute: block) }
    }
}
