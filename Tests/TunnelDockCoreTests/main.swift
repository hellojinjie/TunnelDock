import Darwin
import Foundation
import TestSupport

let filter = CommandLine.arguments.dropFirst().first
let tests = TunnelDefinitionTests.all
    + ForwardSpecificationTests.all
    + SSHConfigScannerTests.all
    + SSHConfigAppenderTests.all
    + SSHIncludeResolverTests.all
    + SSHConfigWatcherTests.all
    + SSHHostResolverTests.all
    + SSHConfigLoaderTests.all
    + TunnelRepositoryTests.all
    + TunnelLogBufferTests.all
    + SSHErrorClassifierTests.all
    + PortAvailabilityCheckerTests.all
    + ControlSocketManagerTests.all
    + SSHProcessControllerTests.all
    + TunnelManagerConnectionTests.all
    + TunnelManagerRecoveryTests.all
let status = await TestRunner.run(tests, filter: filter)
exit(status)
