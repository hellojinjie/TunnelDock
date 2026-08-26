// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "TunnelDock",
    platforms: [.macOS(.v13)],
    products: [
        .library(name: "TunnelDockCore", targets: ["TunnelDockCore"]),
        .library(name: "TunnelDockAppSupport", targets: ["TunnelDockAppSupport"]),
        .executable(name: "TunnelDock", targets: ["TunnelDock"]),
        .executable(name: "TunnelDockCoreTests", targets: ["TunnelDockCoreTests"]),
        .executable(name: "TunnelDockAppTests", targets: ["TunnelDockAppTests"]),
    ],
    targets: [
        .target(name: "TunnelDockCore"),
        .target(name: "TunnelDockAppSupport", dependencies: ["TunnelDockCore"]),
        .executableTarget(
            name: "TunnelDock",
            dependencies: ["TunnelDockCore", "TunnelDockAppSupport"]
        ),
        .target(name: "TestSupport", path: "Tests/TestSupport"),
        .executableTarget(
            name: "TunnelDockCoreTests",
            dependencies: ["TunnelDockCore", "TestSupport"],
            path: "Tests/TunnelDockCoreTests"
        ),
        .executableTarget(
            name: "TunnelDockAppTests",
            dependencies: ["TunnelDockAppSupport", "TestSupport"],
            path: "Tests/TunnelDockAppTests"
        ),
    ],
    swiftLanguageModes: [.v6]
)
