import com.rudderstack.ruddertyper.*
import kotlinx.serialization.json.*

fun main() {
    val analytics = com.rudderstack.sdk.kotlin.core.Analytics()
    val typer = RudderAnalytics(analytics)

    println("=== Testing RudderAnalytics Functions ===\n")
    
    println("Testing identify()...")
    typer.identify(
        userId = "user-123-abc",
        traits = IdentifyTraits(
            active = true,
            email = "john.doe@example.com"
        )
    )
    
    println("\nTesting identify() with optional userId...")
    typer.identify(
        traits = IdentifyTraits(
            active = false,
            email = "jane.smith@example.com"
        )
    )

    println("\nTesting group()...")
    typer.group(
        groupId = "company-xyz-789",
        traits = GroupTraits(
            active = true
        )
    )
    
    println("\nTesting screen() with properties...")
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

    println("\nTesting screen() without category...")
    typer.screen(
        screenName = "Settings",
        properties = ScreenProperties(
            profile = null
        )
    )
    
    println("\nTesting trackUserSignedUp() with comprehensive data...")
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

    println("\nTesting trackUserSignedUp() with minimal data...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = false,
            profile = CustomTypeUserProfile(
                email = "minimal@example.com",
                firstName = "Charlie"
            )
        )
    )
    
    println("\nTesting trackUserSignedUp() with TABLET device...")
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

    println("\nTesting trackUserSignedUp() with DESKTOP device...")
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

    println("\nTesting trackUserSignedUp() with SMARTTV device...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.SMART_TV,
            age = 45.0,
            profile = CustomTypeUserProfile(
                email = "tv.user@example.com",
                firstName = "Fiona",
                lastName = "Chen"
            ),
            tags = listOf("smart-home", "entertainment")
        )
    )

    println("\nTesting trackUserSignedUp() with IOT_DEVICE...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.IO_T_DEVICE,
            profile = CustomTypeUserProfile(
                email = "iot@example.com",
                firstName = "George"
            )
        )
    )

    println("\nTesting trackEventWithVariants() with different variants...")
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

    println("\nTesting trackUserSignedUp() with feature_config enabled (boolean true)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.enabled@example.com",
                firstName = "Premium",
                lastName = "User"
            ),
            featureConfig = CustomTypeFeatureConfig.CaseTrue(
                age = 30.0
            ),
        )
    )

    
    println("\nTesting trackUserSignedUp() with integer enum (priority)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            priority = PropertyPriority._1,
            profile = CustomTypeUserProfile(
                email = "priority.user@example.com",
                firstName = "Ivan"
            )
        )
    )

    
    println("\nTesting trackUserSignedUp() with feature_config disabled (boolean false)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.disabled@example.com",
                firstName = "Free",
                lastName = "User"
            ),
            featureConfig = CustomTypeFeatureConfig.CaseFalse(
                firstName = "some-name"
            ),
        )
    )

    
    println("\nTesting trackUserSignedUp() with boolean enum (enabled)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            enabled = PropertyEnabled.TRUE,
            profile = CustomTypeUserProfile(
                email = "enabled.user@example.com",
                firstName = "Julia"
            )
        )
    )

    
    println("\nTesting trackUserSignedUp() with feature_config beta (string)...")
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
            ),
        )
    )

    
    println("\nTesting trackUserSignedUp() with float enum (rating)...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            rating = PropertyRating._4_5,
            profile = CustomTypeUserProfile(
                email = "rating.user@example.com",
                firstName = "Kevin"
            )
        )
    )

    
    println("\nTesting trackUserSignedUp() with user_access default case...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "default.user@example.com",
                firstName = "Default",
                lastName = "Case"
            ),
            userAccess = CustomTypeUserAccess.Default(
                active = true  // Can be any value - demonstrates default case
            ),
        )
    )

    
    println("\nTesting trackUserSignedUp() with mixed-type enum...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            mixedValue = PropertyMixedValue._2_5,
            profile = CustomTypeUserProfile(
                email = "mixed.user@example.com",
                firstName = "Laura"
            )
        )
    )

    println("\nTesting trackUserSignedUp() with feature_config default case (string 'alpha')...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "feature.alpha@example.com",
                firstName = "Alpha",
                lastName = "User"
            ),
            featureConfig = CustomTypeFeatureConfig.Default(
                featureFlag = PropertyFeatureFlag.StringValue("alpha")  // Not 'beta', true, or false
            ),
        )
    )

    println("\nTesting trackUserSignedUp() with all enum types...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            deviceType = PropertyDeviceType.MOBILE,
            priority = PropertyPriority._3,
            enabled = PropertyEnabled.FALSE,
            rating = PropertyRating._5,
            mixedValue = PropertyMixedValue.ACTIVE,
            status = CustomTypeStatus.ACTIVE,
            profile = CustomTypeUserProfile(
                email = "all.enums@example.com",
                firstName = "Michael"
            )
        )
    )

    println("\nTesting trackUserSignedUp() with Unit serialization...")
    typer.trackUserSignedUp(
        properties = TrackUserSignedUpProperties(
            active = true,
            profile = CustomTypeUserProfile(
                email = "unit.test@example.com",
                firstName = "Unit",
                lastName = "Test"
            ),
            nestedEmptyObjectNoAdditionalProps = Unit,
            tags = listOf("unit", "test")
        )
    )

    println("\n=== All Tests Completed ===")
}