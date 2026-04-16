import XCTest
import RudderStackAnalytics
@testable import RudderTyper

final class RudderTyperTests: XCTestCase {
    var analytics: Analytics!
    var validations: EventValidationPlugin!
    var typer: RudderTyperAnalytics!

    override func setUp() {
        super.setUp()
        let config = Configuration(
            writeKey: "test-write-key",
            dataPlaneUrl: "https://localhost:1234",
            controlPlaneUrl: "https://localhost:1234",
            flushPolicies: [CountFlushPolicy(flushAt: 1)],
            trackApplicationLifecycleEvents: false,
            sessionConfiguration: SessionConfiguration(automaticSessionTracking: false)
        )
        analytics = Analytics(configuration: config)
        validations = EventValidationPlugin()
        analytics.add(plugin: validations)
        typer = RudderTyperAnalytics(analytics: analytics)
    }

    // MARK: - identify

    func testIdentify() {
        typer.identify(
            userId: "user-123-abc",
            traits: IdentifyTraits(email: "john.doe@example.com", active: true)
        )
        typer.identify(
            userId: "user-456-def",
            traits: IdentifyTraits(email: "jane.smith@example.com", active: false)
        )
        analytics.flush()

        validations.validateCount(2)
        validations.validateNext(.identify(
            userId: "user-123-abc",
            traits: ["active": true, "email": "john.doe@example.com"]
        ))
        validations.validateNext(.identify(
            userId: "user-456-def",
            traits: ["active": false, "email": "jane.smith@example.com"]
        ))
    }

    // MARK: - group

    func testGroup() {
        typer.group(
            groupId: "company-xyz-789",
            traits: GroupTraits(active: true)
        )
        analytics.flush()

        validations.validateCount(1)
        // Traits are routed through options.customContext, not onto GroupEvent.traits.
        validations.validateNext(.group(groupId: "company-xyz-789", traits: [:]))
    }

    // MARK: - screen

    func testScreen() {
        typer.screen(
            screenName: "Dashboard",
            properties: ScreenProperties(
                profile: CustomTypeUserProfile(
                    email: "user@example.com",
                    firstName: "Alice",
                    lastName: "Johnson"
                )
            ),
            category: "Main Navigation"
        )
        typer.screen(screenName: "Settings", properties: ScreenProperties())
        analytics.flush()

        validations.validateCount(2)
        validations.validateNext(.screen(
            screenName: "Dashboard",
            properties: [
                "profile": [
                    "email": "user@example.com",
                    "first_name": "Alice",
                    "last_name": "Johnson",
                ],
                "name": "Dashboard",
                "category": "Main Navigation",
            ]
        ))
        validations.validateNext(.screen(
            screenName: "Settings",
            properties: ["name": "Settings"]
        ))
    }
}
