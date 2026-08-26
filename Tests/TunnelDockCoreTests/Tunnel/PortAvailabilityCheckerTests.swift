import Darwin
import Foundation
import TestSupport
import TunnelDockCore

enum PortAvailabilityCheckerTests {
    static let all: [TestCase] = [
        TestCase("PortAvailabilityCheckerTests.detectsSystemOwnedLoopbackPort") {
            let listener = try LoopbackListener()
            defer { listener.close() }
            let checker = SystemPortAvailabilityChecker()

            let result = await checker.check(LocalEndpoint(address: "127.0.0.1", port: listener.port))

            try expectEqual(result, .occupied(.system))
        },
        TestCase("PortAvailabilityCheckerTests.detectsTunnelDockReservationBeforeSystemCheck") {
            let checker = SystemPortAvailabilityChecker()
            let endpoint = LocalEndpoint(address: "127.0.0.1", port: 8_888)
            try expectEqual(await checker.reserve(endpoint), true)

            try expectEqual(await checker.check(endpoint), .occupied(.tunnelDock))

            await checker.release(endpoint)
            try expectEqual(await checker.isReserved(endpoint), false)
        },
        TestCase("PortAvailabilityCheckerTests.reportsReleasedEphemeralPortAvailable") {
            let listener = try LoopbackListener()
            let endpoint = LocalEndpoint(address: "127.0.0.1", port: listener.port)
            listener.close()
            let checker = SystemPortAvailabilityChecker()

            try expectEqual(await checker.check(endpoint), .available)
        },
        TestCase("PortAvailabilityCheckerTests.listenerProbeObservesLocalTCPHandshake") {
            let listener = try LoopbackListener()
            defer { listener.close() }
            let probe = SystemLocalListenerProbe()

            let listening = await probe.waitUntilListening(
                LocalEndpoint(address: "127.0.0.1", port: listener.port),
                timeout: .milliseconds(300)
            )

            try expectEqual(listening, true)
        },
    ]
}

private final class LoopbackListener: @unchecked Sendable {
    private var descriptor: Int32
    let port: UInt16

    init() throws {
        let socketDescriptor = socket(AF_INET, SOCK_STREAM, 0)
        guard socketDescriptor >= 0 else { throw TestFailure("socket failed") }
        var address = sockaddr_in()
        address.sin_len = UInt8(MemoryLayout<sockaddr_in>.size)
        address.sin_family = sa_family_t(AF_INET)
        address.sin_port = 0
        address.sin_addr = in_addr(s_addr: inet_addr("127.0.0.1"))
        let bindResult = withUnsafePointer(to: &address) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                Darwin.bind(socketDescriptor, $0, socklen_t(MemoryLayout<sockaddr_in>.size))
            }
        }
        guard bindResult == 0, listen(socketDescriptor, 4) == 0 else {
            Darwin.close(socketDescriptor)
            throw TestFailure("bind/listen failed")
        }
        var stored = sockaddr_in()
        var length = socklen_t(MemoryLayout<sockaddr_in>.size)
        let nameResult = withUnsafeMutablePointer(to: &stored) {
            $0.withMemoryRebound(to: sockaddr.self, capacity: 1) {
                getsockname(socketDescriptor, $0, &length)
            }
        }
        guard nameResult == 0 else {
            Darwin.close(socketDescriptor)
            throw TestFailure("getsockname failed")
        }
        descriptor = socketDescriptor
        port = UInt16(bigEndian: stored.sin_port)
    }

    func close() {
        guard descriptor >= 0 else { return }
        Darwin.close(descriptor)
        descriptor = -1
    }

    deinit { close() }
}
