import Darwin
import Foundation
import TestSupport

let filter = CommandLine.arguments.dropFirst().first
let tests = PlaceholderTests.all
    + AppStateTests.all
    + QuickForwardModelTests.all
    + SSHHostDraftTests.all
    + SSHConfigEditorOpenerTests.all
    + MenuBarModelTests.all
    + TunnelOverviewSortTests.all
    + TunnelRowPresentationTests.all
    + AppTerminationCoordinatorTests.all
    + SidebarSearchFieldConfigurationTests.all
let status = await TestRunner.run(tests, filter: filter)
exit(status)
