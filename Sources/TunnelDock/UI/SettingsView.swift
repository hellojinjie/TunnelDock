import SwiftUI

struct SettingsView: View {
    @Binding var showMenuBar: Bool

    var body: some View {
        Form {
            Toggle("Show in Menu Bar", isOn: $showMenuBar)
        }
        .padding(20)
        .frame(width: 360)
    }
}
