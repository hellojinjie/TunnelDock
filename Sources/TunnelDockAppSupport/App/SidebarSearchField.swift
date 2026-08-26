import AppKit
import SwiftUI

@MainActor
public enum SidebarSearchFieldConfiguration {
    public static func configure(_ textField: NSTextField) {
        textField.isAutomaticTextCompletionEnabled = false
    }

    public static func configure(_ textView: NSTextView) {
        textView.isAutomaticTextCompletionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = false
    }
}

public struct SidebarSearchField: NSViewRepresentable {
    @Binding private var text: String

    public init(text: Binding<String>) {
        _text = text
    }

    public func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    public func makeNSView(context: Context) -> NSTextField {
        let textField = NSTextField()
        textField.placeholderString = "Search"
        textField.font = .systemFont(ofSize: NSFont.smallSystemFontSize)
        textField.isBordered = false
        textField.drawsBackground = false
        textField.focusRingType = .none
        textField.delegate = context.coordinator
        SidebarSearchFieldConfiguration.configure(textField)
        return textField
    }

    public func updateNSView(_ textField: NSTextField, context: Context) {
        context.coordinator.parent = self
        if textField.stringValue != text {
            textField.stringValue = text
        }
    }

    public final class Coordinator: NSObject, NSTextFieldDelegate {
        fileprivate var parent: SidebarSearchField

        fileprivate init(_ parent: SidebarSearchField) {
            self.parent = parent
        }

        public func controlTextDidBeginEditing(_ notification: Notification) {
            guard let textField = notification.object as? NSTextField,
                  let editor = textField.currentEditor() as? NSTextView else {
                return
            }
            SidebarSearchFieldConfiguration.configure(editor)
        }

        public func controlTextDidChange(_ notification: Notification) {
            guard let textField = notification.object as? NSTextField else {
                return
            }
            parent.text = textField.stringValue
        }
    }
}
