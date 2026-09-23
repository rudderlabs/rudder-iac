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

    // MARK: - track (comprehensive)

    func testTrackUserSignedUpComprehensive() {
        typer.trackUserSignedUp(
            properties: TrackUserSignedUpProperties(
                active: true,
                profile: CustomTypeUserProfile(
                    email: "newuser@example.com",
                    firstName: "Bob",
                    lastName: "Williams"
                ),
                age: 28.5,
                arrayOfAny: [
                    .string("string item"),
                    .int(42),
                    .bool(true),
                    .object([
                        "nested": .string("object"),
                        "count": .int(100),
                    ]),
                ],
                contacts: [
                    "contact1@example.com",
                    "contact2@example.com",
                    "support@company.org",
                ],
                deviceType: .mobile,
                propertyOfAny: .object([
                    "custom_field_1": .string("value1"),
                    "custom_field_2": .int(999),
                    "nested_object": .object(["deep_field": .string("deep_value")]),
                ]),
                tags: ["premium", "early-adopter", "beta-tester", "verified"],
                untypedArray: [
                    .double(3.14159),
                    .string("mixed"),
                    .bool(false),
                    .object(["id": .int(123), "name": .string("test")]),
                ],
                untypedField: .object([
                    "arbitrary_key": .string("arbitrary_value"),
                    "number": .double(42.5),
                ])
            )
        )
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(
            name: "User Signed Up",
            properties: [
                "active": true,
                "age": 28.5,
                "array_of_any": [
                    "string item",
                    42,
                    true,
                    ["nested": "object", "count": 100],
                ] as [Any],
                "contacts": ["contact1@example.com", "contact2@example.com", "support@company.org"],
                "device_type": "mobile",
                "profile": [
                    "email": "newuser@example.com",
                    "first_name": "Bob",
                    "last_name": "Williams",
                ],
                "property_of_any": [
                    "custom_field_1": "value1",
                    "custom_field_2": 999,
                    "nested_object": ["deep_field": "deep_value"],
                ] as [String: Any],
                "tags": ["premium", "early-adopter", "beta-tester", "verified"],
                "untyped_array": [
                    3.14159,
                    "mixed",
                    false,
                    ["id": 123, "name": "test"] as [String: Any],
                ] as [Any],
                "untyped_field": [
                    "arbitrary_key": "arbitrary_value",
                    "number": 42.5,
                ] as [String: Any],
            ]
        ))
    }

    // MARK: - track (minimal)

    func testTrackUserSignedUpMinimal() {
        typer.trackUserSignedUp(
            properties: TrackUserSignedUpProperties(
                active: false,
                profile: CustomTypeUserProfile(email: "minimal@example.com", firstName: "Charlie")
            )
        )
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(
            name: "User Signed Up",
            properties: [
                "active": false,
                "profile": ["email": "minimal@example.com", "first_name": "Charlie"],
            ]
        ))
    }

    // MARK: - track (all device types)

    func testTrackUserSignedUpDeviceTypes() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "tablet.user@example.com", firstName: "Diana", lastName: "Martinez"),
            deviceType: .tablet
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "desktop.user@example.com", firstName: "Edward"),
            deviceType: .desktop
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "tv.user@example.com", firstName: "Fiona", lastName: "Chen"),
            age: 45.0,
            deviceType: .smartTv,
            tags: ["smart-home", "entertainment"]
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "iot@example.com", firstName: "George"),
            deviceType: .ioTDevice
        ))
        analytics.flush()

        validations.validateCount(4)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "device_type": "tablet",
            "profile": ["email": "tablet.user@example.com", "first_name": "Diana", "last_name": "Martinez"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "device_type": "desktop",
            "profile": ["email": "desktop.user@example.com", "first_name": "Edward"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "device_type": "smartTV",
            "age": 45.0,
            "profile": ["email": "tv.user@example.com", "first_name": "Fiona", "last_name": "Chen"],
            "tags": ["smart-home", "entertainment"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "device_type": "IoT-Device",
            "profile": ["email": "iot@example.com", "first_name": "George"],
        ]))
    }

    // MARK: - track (variants discriminator)

    func testTrackEventWithVariants() {
        typer.trackEventWithVariants(
            properties: .caseMobile(.init(
                profile: CustomTypeUserProfile(
                    email: "mobile.user@example.com",
                    firstName: "Hannah",
                    lastName: "Smith"
                ),
                tags: ["mobile", "app-user"]
            ))
        )
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(name: "Event With Variants", properties: [
            "device_type": "mobile",
            "profile": [
                "email": "mobile.user@example.com",
                "first_name": "Hannah",
                "last_name": "Smith",
            ],
            "tags": ["mobile", "app-user"],
        ]))
    }

    // MARK: - track (feature config variants)
    //
    // The Swift generator emits an empty `Default` struct (no fields) for the
    // discriminator's fallback case. That is why the fourth "alpha" case here
    // asserts an empty `feature_config` dict — differs from Kotlin where the
    // Default case carries a `featureFlag` field.

    func testTrackUserSignedUpFeatureConfigs() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "feature.enabled@example.com", firstName: "Premium", lastName: "User"),
            featureConfig: .caseTrue(.init(age: 30.0))
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "feature.disabled@example.com", firstName: "Free", lastName: "User"),
            featureConfig: .caseFalse(.init(firstName: "some-name"))
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "feature.beta@example.com", firstName: "Beta", lastName: "Tester"),
            featureConfig: .caseBeta(.init(tags: ["beta-user", "early-access", "experimental"]))
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "feature.alpha@example.com", firstName: "Alpha", lastName: "User"),
            featureConfig: .other(.init())
        ))
        analytics.flush()

        validations.validateCount(4)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "feature_config": ["age": 30.0, "feature_flag": true] as [String: Any],
            "profile": ["email": "feature.enabled@example.com", "first_name": "Premium", "last_name": "User"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "feature_config": ["feature_flag": false, "first_name": "some-name"] as [String: Any],
            "profile": ["email": "feature.disabled@example.com", "first_name": "Free", "last_name": "User"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "feature_config": [
                "feature_flag": "beta",
                "tags": ["beta-user", "early-access", "experimental"],
            ] as [String: Any],
            "profile": ["email": "feature.beta@example.com", "first_name": "Beta", "last_name": "Tester"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "feature_config": [:] as [String: Any],
            "profile": ["email": "feature.alpha@example.com", "first_name": "Alpha", "last_name": "User"],
        ]))
    }

    // MARK: - track (primitive enums)
    //
    // Swift emits `PropertyEnabled` and `PropertyMixedValue` as String-backed
    // enums (rawValue is a String like "true" / "2.5"), so the serialized JSON
    // carries strings — not the booleans / numbers the Kotlin test asserts.

    func testTrackUserSignedUpEnums() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "priority.user@example.com", firstName: "Ivan"),
            priority: .n1
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "enabled.user@example.com", firstName: "Julia"),
            enabled: .`true`
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "rating.user@example.com", firstName: "Kevin"),
            rating: .n45
        ))
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "mixed.user@example.com", firstName: "Laura"),
            mixedValue: .n25
        ))
        analytics.flush()

        validations.validateCount(4)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "priority": 1,
            "profile": ["email": "priority.user@example.com", "first_name": "Ivan"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "enabled": "true",
            "profile": ["email": "enabled.user@example.com", "first_name": "Julia"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "rating": 4.5,
            "profile": ["email": "rating.user@example.com", "first_name": "Kevin"],
        ]))
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "mixed_value": "2.5",
            "profile": ["email": "mixed.user@example.com", "first_name": "Laura"],
        ]))
    }

    // MARK: - track (user access variant)
    //
    // The Swift `CustomTypeUserAccess.Default` struct carries no fields, so
    // the serialized `user_access` is an empty object — unlike Kotlin, whose
    // Default carries `active`.

    func testTrackUserSignedUpUserAccess() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "default.user@example.com", firstName: "Default", lastName: "Case"),
            userAccess: .other(.init())
        ))
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "profile": ["email": "default.user@example.com", "first_name": "Default", "last_name": "Case"],
            "user_access": [:] as [String: Any],
        ]))
    }

    // MARK: - track (combined enums)

    func testTrackUserSignedUpAllEnums() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "all.enums@example.com", firstName: "Michael"),
            deviceType: .mobile,
            enabled: .`false`,
            mixedValue: .active,
            priority: .n3,
            rating: .n5,
            status: .active
        ))
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "device_type": "mobile",
            "enabled": "false",
            "mixed_value": "active",
            "priority": 3,
            "profile": ["email": "all.enums@example.com", "first_name": "Michael"],
            "rating": 5.0,
            "status": "active",
        ]))
    }

    // MARK: - track (empty-object field)

    func testTrackUserSignedUpUnitSerialization() {
        typer.trackUserSignedUp(properties: TrackUserSignedUpProperties(
            active: true,
            profile: CustomTypeUserProfile(email: "unit.test@example.com", firstName: "Unit", lastName: "Test"),
            nestedEmptyObjectNoAdditionalProps: [:],
            tags: ["unit", "test"]
        ))
        analytics.flush()

        validations.validateCount(1)
        validations.validateNext(.track(name: "User Signed Up", properties: [
            "active": true,
            "nested_empty_object_no_additional_props": [:] as [String: Any],
            "profile": ["email": "unit.test@example.com", "first_name": "Unit", "last_name": "Test"],
            "tags": ["unit", "test"],
        ]))
    }
}
