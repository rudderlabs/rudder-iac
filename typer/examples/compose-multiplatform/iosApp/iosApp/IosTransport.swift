import Foundation
import RudderStackAnalytics
import Shared

/// Implements the generated Kotlin RudderTyperTransport protocol on top of the RudderStack Swift SDK.
final class IosTransport: RudderTyperTransport {
    private let analytics: Analytics

    init(writeKey: String, dataPlaneUrl: String, controlPlaneUrl: String) {
        analytics = Analytics(configuration: Configuration(
            writeKey: writeKey,
            dataPlaneUrl: dataPlaneUrl,
            controlPlaneUrl: controlPlaneUrl,
            logLevel: .debug
        ))
    }

    func track(name: String, properties: [String: Any], context: [String: Any]) {
        analytics.track(name: name, properties: properties, options: RudderOption(customContext: context))
    }

    func identify(userId: String, traits: [String: Any], context: [String: Any]) {
        analytics.identify(userId: userId, traits: traits, options: RudderOption(customContext: context))
    }

    func group(groupId: String, traits: [String: Any], context: [String: Any]) {
        analytics.group(groupId: groupId, traits: traits, options: RudderOption(customContext: context))
    }

    func screen(screenName: String, category: String, properties: [String: Any], context: [String: Any]) {
        analytics.screen(screenName: screenName, category: category, properties: properties, options: RudderOption(customContext: context))
    }
}
