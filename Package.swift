// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "SmartPaste",
    platforms: [.macOS(.v14)],
    targets: [
        .executableTarget(
            name: "SmartPaste",
            path: "Sources/SmartPaste"
        )
    ]
)
