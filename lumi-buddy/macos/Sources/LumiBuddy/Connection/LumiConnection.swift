import Foundation

final class LumiConnection {
    private let host: String
    private let token: String
    private let dispatcher: CommandDispatcher
    private let reconnect = Reconnect()

    private var session: URLSession?
    private var task: URLSessionWebSocketTask?
    private var loopTask: Task<Void, Never>?
    private var keepAliveTask: Task<Void, Never>?
    private var stopped = false

    init(host: String, token: String, dispatcher: CommandDispatcher) {
        self.host = host
        self.token = token
        self.dispatcher = dispatcher
    }

    func connect() {
        guard loopTask == nil else { return }
        stopped = false
        loopTask = Task { [weak self] in
            await self?.runLoop()
        }
    }

    func disconnect() {
        stopped = true
        keepAliveTask?.cancel()
        keepAliveTask = nil
        task?.cancel(with: .goingAway, reason: nil)
        task = nil
        session?.invalidateAndCancel()
        session = nil
        loopTask?.cancel()
        loopTask = nil
        AppState.shared.setConnection(.disconnected)
    }

    private func runLoop() async {
        while !stopped {
            AppState.shared.setConnection(.connecting)
            do {
                try await runOnce()
            } catch is CancellationError {
                break
            } catch {
                AppState.shared.setConnection(.error(error.localizedDescription))
            }
            if stopped { break }
            let delay = await reconnect.nextDelay()
            try? await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
        }
    }

    private func runOnce() async throws {
        guard let url = URL(string: "ws://\(host)/api/buddy/ws") else {
            throw PairingError.network("invalid host: \(host)")
        }
        var req = URLRequest(url: url)
        req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")

        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        let session = URLSession(configuration: config)
        self.session = session

        let task = session.webSocketTask(with: req)
        self.task = task
        task.resume()

        // Keep-alive ping every 15s. Server should respond pong; if it stops, receive() throws.
        keepAliveTask = Task { [weak self] in
            while let self, !Task.isCancelled {
                try? await Task.sleep(nanoseconds: 15_000_000_000)
                if Task.isCancelled { break }
                self.task?.sendPing(pongReceiveHandler: { _ in })
            }
        }

        await reconnect.reset()
        AppState.shared.setConnection(.connected)

        defer {
            keepAliveTask?.cancel()
            keepAliveTask = nil
        }

        while !Task.isCancelled {
            let message = try await task.receive()
            switch message {
            case .data(let data):
                await handle(data: data)
            case .string(let s):
                if let data = s.data(using: .utf8) { await handle(data: data) }
            @unknown default:
                break
            }
        }
    }

    private func handle(data: Data) async {
        let response = await dispatcher.dispatch(data)
        guard let task else { return }
        guard let str = String(data: response, encoding: .utf8) else { return }
        do {
            try await task.send(.string(str))
        } catch {
            NSLog("LumiConnection send error: \(error.localizedDescription)")
        }
    }
}
