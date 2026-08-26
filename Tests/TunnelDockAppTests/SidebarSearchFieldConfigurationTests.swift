import AppKit
import TestSupport
import TunnelDockAppSupport

enum SidebarSearchFieldConfigurationTests {
    static let all: [TestCase] = [
        TestCase("SidebarSearchFieldConfigurationTests.disablesAutomaticTextAssistance") {
            let values = await MainActor.run {
                let field = NSTextField()
                let editor = NSTextView()
                SidebarSearchFieldConfiguration.configure(field)
                SidebarSearchFieldConfiguration.configure(editor)
                return (
                    field.isAutomaticTextCompletionEnabled,
                    editor.isAutomaticTextCompletionEnabled,
                    editor.isAutomaticTextReplacementEnabled,
                    editor.isAutomaticSpellingCorrectionEnabled
                )
            }

            try expectEqual(values.0, false)
            try expectEqual(values.1, false)
            try expectEqual(values.2, false)
            try expectEqual(values.3, false)
        },
    ]
}
