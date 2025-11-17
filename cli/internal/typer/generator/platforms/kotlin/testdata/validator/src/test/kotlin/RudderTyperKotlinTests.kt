package com.rudderstack.ruddertyper

import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import com.rudderstack.sdk.kotlin.core.Analytics
import com.rudderstack.sdk.kotlin.core.Configuration
import com.rudderstack.sdk.kotlin.core.internals.policies.CountFlushPolicy
import kotlinx.serialization.json.*

class RudderTyperKotlinTests {
    private lateinit var analytics: Analytics
    private lateinit var validations: EventValidationPlugin
    private lateinit var typer: RudderAnalytics

    @BeforeEach
    fun setup() {
        // Initialize real Analytics SDK with minimal configuration
        val config = Configuration(
            writeKey = "test-write-key",
            controlPlaneUrl = "https://localhost:1234",
            dataPlaneUrl = "https://localhost:1234",
            flushPolicies = listOf(
                CountFlushPolicy(1)
            )
        )
        analytics = Analytics(config)
        validations = EventValidationPlugin()
        analytics.add(validations)
        typer = RudderAnalytics(analytics)
    }

    @Test
    fun testIdentify() {
        typer.identify(
            userId = "user-123-abc",
            traits = IdentifyTraits(
                active = true,
                email = "john.doe@example.com"
            )
        )

        typer.identify(
            userId = "user-456-def",
            traits = IdentifyTraits(
                active = false,
                email = "jane.smith@example.com"
            )
        )

        analytics.flush()
        validations.validateCount(2)
        validations.validateNext(
            EventValidation.IdentifyValidation(
                userId = "user-123-abc",
                traits = buildJsonObject {
                    put("active", true)
                    put("email", "john.doe@example.com")
                }
            )
        )
        validations.validateNext(
            EventValidation.IdentifyValidation(
                userId = "user-456-def",
                traits = buildJsonObject {
                    put("active", false)
                    put("email", "jane.smith@example.com")
                }
            )
        )
    }

    @Test
    fun testGroup() {
        typer.group(
            groupId = "company-xyz-789",
            traits = GroupTraits(
                active = true
            )
        )

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(
            EventValidation.GroupValidation(
                groupId = "company-xyz-789",
                traits = buildJsonObject {
                    put("active", true)
                }
            )
        )
    }

    @Test
    fun testScreen() {
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

        typer.screen(
            screenName = "Settings",
            properties = ScreenProperties()
        )

        analytics.flush()
        validations.validateCount(2)
        validations.validateNext(
            EventValidation.ScreenValidation(
                screenName = "Dashboard",
                properties = buildJsonObject {
                    put("profile", buildJsonObject {
                        put("email", "user@example.com")
                        put("first_name", "Alice")
                        put("last_name", "Johnson")
                    })
                    // the following are injected by the SDK
                    put("name", "Dashboard")
                    put("category", "Main Navigation")
                }
            )
        )
        validations.validateNext(
            EventValidation.ScreenValidation(
                screenName = "Settings",
                properties = buildJsonObject {
                    // the following are injected by the SDK
                    put("name", "Settings")
                }
            )
        )
    }

    @Test
    fun testTrackUserSignedUpComprehensive() {
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

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(
            EventValidation.TrackValidation(
                name = "User Signed Up",
                properties = buildJsonObject {
                }
            )
        )
    }

    @Test
    fun testTrackUserSignedUpMinimal() {
        typer.trackUserSignedUp(
            properties = TrackUserSignedUpProperties(
                active = false,
                profile = CustomTypeUserProfile(
                    email = "minimal@example.com",
                    firstName = "Charlie"
                )
            )
        )

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(
            EventValidation.TrackValidation(
                name = "User Signed Up",
                properties = buildJsonObject {}
            )
        )
    }

    @Test
    fun testTrackUserSignedUpDeviceTypes() {
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

        analytics.flush()
        validations.validateCount(4)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackEventWithVariants() {
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

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(EventValidation.TrackValidation(name = "Event With Variants", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackUserSignedUpFeatureConfigs() {
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

        typer.trackUserSignedUp(
            properties = TrackUserSignedUpProperties(
                active = true,
                profile = CustomTypeUserProfile(
                    email = "feature.alpha@example.com",
                    firstName = "Alpha",
                    lastName = "User"
                ),
                featureConfig = CustomTypeFeatureConfig.Default(
                    featureFlag = PropertyFeatureFlag.StringValue("alpha")
                ),
            )
        )

        analytics.flush()
        validations.validateCount(4)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackUserSignedUpEnums() {
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

        analytics.flush()
        validations.validateCount(4)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {

        }))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackUserSignedUpUserAccess() {
        typer.trackUserSignedUp(
            properties = TrackUserSignedUpProperties(
                active = true,
                profile = CustomTypeUserProfile(
                    email = "default.user@example.com",
                    firstName = "Default",
                    lastName = "Case"
                ),
                userAccess = CustomTypeUserAccess.Default(
                    active = true
                ),
            )
        )

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackUserSignedUpAllEnums() {
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

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }

    @Test
    fun testTrackUserSignedUpUnitSerialization() {
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

        analytics.flush()
        validations.validateCount(1)
        validations.validateNext(EventValidation.TrackValidation(name = "User Signed Up", properties = buildJsonObject {}))
    }
}
