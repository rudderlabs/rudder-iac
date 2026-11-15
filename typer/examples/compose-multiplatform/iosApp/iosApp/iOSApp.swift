import SwiftUI
import Shared

@main
struct iOSApp: App {
    // Placeholders. Point them at a data plane you control to see the events arrive.
    private let transport = IosTransport(
        writeKey: "YOUR_WRITE_KEY",
        dataPlaneUrl: "http://localhost:19187",
        controlPlaneUrl: "http://localhost:19187"
    )

    var body: some Scene {
        WindowGroup {
            ComposeView(transport: transport).ignoresSafeArea()
        }
    }
}

struct ComposeView: UIViewControllerRepresentable {
    let transport: RudderTyperTransport

    func makeUIViewController(context: Context) -> UIViewController {
        MainViewControllerKt.MainViewController(transport: transport)
    }

    func updateUIViewController(_ uiViewController: UIViewController, context: Context) {}
}
