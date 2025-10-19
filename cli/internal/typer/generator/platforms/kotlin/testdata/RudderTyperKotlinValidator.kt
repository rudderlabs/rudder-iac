import com.rudderstack.ruddertyper.*
import kotlinx.serialization.json.*

fun main() {
    val analytics = com.rudderstack.sdk.kotlin.core.Analytics()
    val typer = RudderAnalytics(analytics)

    println("=== Testing RudderAnalytics Functions ===\n")

    // Test 1: Identify with comprehensive traits
    println("1. Testing identify()...")
    typer.identify(
        userId = "user-123-abc",
        traits = IdentifyTraits(
            active = true,
            email = "john.doe@example.com"
        )
    )

    // Test 2: Identify with minimal data (no userId, minimal traits)
    println("\n2. Testing identify() with optional userId...")
    typer.identify(
        traits = IdentifyTraits(
            active = false,
            email = "jane.smith@example.com"
        )
    )

    // Test 3: Group with traits
    println("\n3. Testing group()...")
    typer.group(
        groupId = "company-xyz-789",
        traits = GroupTraits(
            active = true
        )
    )

    // Test 4: Screen with full properties
    println("\n4. Testing screen() with properties...")
    typer.screen(
        screenName = "Dashboard",
        category = "Main Navigation",
        properties = ScreenProperties(
            profile = CustomTypeUserProfile(
                email = "user@example.com",
                firstName = "Alice",
                lastName = "Johnson"
            )
        )
    )

    // Test 5: Screen without category
    println("\n5. Testing screen() without category...")
    typer.screen(
        screenName = "Settings",
        properties = ScreenProperties(
            profile = null
        )
    )

    // Test 6: Track User Signed Up with all properties
    println("\n6. Testing trackUserSignedUp() with comprehensive data...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            age = 28.5,
            arrayOfAny = buildJsonArray {
                add(JsonPrimitive("string item"))
                add(JsonPrimitive(42))
                add(JsonPrimitive(true))
                add(buildJsonObject {
                    put("nested", "object")
                    put("count", 100)
                })
            },
            contacts = listOf(
                "contact1@example.com",
                "contact2@example.com",
                "support@company.org"
            ),
            deviceType = PropertyDeviceType.MOBILE,
            profile = CustomTypeUserProfile(
                email = "newuser@example.com",
                firstName = "Bob",
                lastName = "Williams"
            ),
            propertyOfAny = buildJsonObject {
                put("custom_field_1", "value1")
                put("custom_field_2", 999)
                put("nested_object", buildJsonObject {
                    put("deep_field", "deep_value")
                })
            },
            tags = listOf("premium", "early-adopter", "beta-tester", "verified"),
            untypedArray = buildJsonArray {
                add(JsonPrimitive(3.14159))
                add(JsonPrimitive("mixed"))
                add(JsonPrimitive(false))
                add(buildJsonObject {
                    put("id", 123)
                    put("name", "test")
                })
            },
            untypedField = buildJsonObject {
                put("arbitrary_key", "arbitrary_value")
                put("number", 42.5)
            }
        )
    )

    // Test 7: Track User Signed Up with minimal required properties
    println("\n7. Testing trackUserSignedUp() with minimal data...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = false,
            profile = CustomTypeUserProfile(
                email = "minimal@example.com",
                firstName = "Charlie"
            )
        )
    )

    // Test 8: Track with different device types
    println("\n8. Testing trackUserSignedUp() with TABLET device...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.TABLET,
            profile = CustomTypeUserProfile(
                email = "tablet.user@example.com",
                firstName = "Diana",
                lastName = "Martinez"
            )
        )
    )

    println("\n9. Testing trackUserSignedUp() with DESKTOP device...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.DESKTOP,
            profile = CustomTypeUserProfile(
                email = "desktop.user@example.com",
                firstName = "Edward"
            )
        )
    )

    println("\n10. Testing trackUserSignedUp() with SMARTTV device...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.SMARTTV,
            age = 45.0,
            profile = CustomTypeUserProfile(
                email = "tv.user@example.com",
                firstName = "Fiona",
                lastName = "Chen"
            ),
            tags = listOf("smart-home", "entertainment")
        )
    )

    println("\n11. Testing trackUserSignedUp() with IOT_DEVICE...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.IOT_DEVICE,
            profile = CustomTypeUserProfile(
                email = "iot@example.com",
                firstName = "George"
            )
        )
    )

    println("\n12. Testing trackEventWithVariants() with different variants...")
    typer.trackEventWithVariants(
        properties = TrackEventWithVariantsProperties.CaseMobile(
            profile = CustomTypeUserProfile(
                email = "mobile.user@example.com",
                firstName = "Hannah",
                lastName = "Smith"
            ),
            tags = listOf("mobile", "app-user"),
        )
    )

    // Test 13: Multi-type discriminator with boolean true (enabled)
    println("\n13. Testing trackUserSignedUp() with feature_config enabled (boolean true)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.enabled@example.com",
                firstName = "Premium",
                lastName = "User"
            ),
            featureConfig = CustomTypeFeatureConfig.Case_True(
                email = "premium@example.com"
            )
        )
    )

    // Test 14: Multi-type discriminator with boolean false (disabled)
    println("\n14. Testing trackUserSignedUp() with feature_config disabled (boolean false)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.disabled@example.com",
                firstName = "Free",
                lastName = "User"
            ),
            featureConfig = CustomTypeFeatureConfig.Case_False(
                status = PropertyStatus.ACTIVE
            )
        )
    )

    // Test 15: Multi-type discriminator with string "beta"
    println("\n15. Testing trackUserSignedUp() with feature_config beta (string)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.beta@example.com",
                firstName = "Beta",
                lastName = "Tester"
            ),
            featureConfig = CustomTypeFeatureConfig.CaseBeta(
                tags = listOf("beta-user", "early-access", "experimental")
            )
        )
    )

    println("\n=== All Tests Completed ===")
}