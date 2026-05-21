import Foundation

struct LampInfo: Hashable, Identifiable {
    var id: String { host }
    let name: String      // e.g. "lumi-a1b2"
    let host: String      // resolved hostname, e.g. "lumi-a1b2.local"
    let port: Int         // default 80 (TXT may override later)
    let discoveredAt: Date
}
