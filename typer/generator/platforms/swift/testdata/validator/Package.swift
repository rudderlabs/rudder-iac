// swift-tools-version:5.9
import PackageDescription

let package = Package(
    name: "RudderTyperValidator",
    platforms: [
        .macOS(.v13),
        .iOS(.v13),
    ],
    dependencies: [
        .package(url: "https://github.com/rudderlabs/rudder-sdk-swift.git", exact: "1.2.1"),
    ],
    targets: [
        .target(
            name: "RudderTyper",
            dependencies: [
                .product(name: "RudderStackAnalytics", package: "rudder-sdk-swift"),
            ],
            swiftSettings: [
                // Apps that adopt this (or the Swift 6 language mode with it
                // enabled) need the generated file to import the SDK publicly.
                .enableUpcomingFeature("InternalImportsByDefault"),
            ]
        ),
        .testTarget(
            name: "RudderTyperTests",
            dependencies: [
                "RudderTyper",
                .product(name: "RudderStackAnalytics", package: "rudder-sdk-swift"),
            ]
        ),
    ]
)
