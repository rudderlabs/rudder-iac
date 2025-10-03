package com.rudderstack.ruddertyper

import com.rudderstack.sdk.kotlin.core.Analytics
import com.rudderstack.sdk.kotlin.core.internals.models.RudderOptions
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.jsonObject

/** Whether user is active */
typealias CustomTypeActive = Boolean

/** User's age in years */
typealias CustomTypeAge = Double

/** Custom type for email validation */
typealias CustomTypeEmail = String

/** User active status */
typealias PropertyActive = CustomTypeActive

/** User's age */
typealias PropertyAge = CustomTypeAge

/** An array that can contain any type of items */
typealias PropertyArrayOfAny = List<JsonElement>

/** Array of user contacts */
typealias PropertyContacts = List<CustomTypeEmail>

/** User's email address */
typealias PropertyEmail = CustomTypeEmail

/** User's first name */
typealias PropertyFirstName = String

/** User's last name */
typealias PropertyLastName = String

/** An object field with no defined structure */
typealias PropertyObjectProperty = JsonObject

/** User profile data */
typealias PropertyProfile = CustomTypeUserProfile

/** A field that can contain any type of value */
typealias PropertyPropertyOfAny = JsonElement

/** User tags as array of strings */
typealias PropertyTags = List<String>

/** An array with no explicit item type (treated as any) */
typealias PropertyUntypedArray = List<JsonElement>

/** A field with no explicit type (treated as any) */
typealias PropertyUntypedField = JsonElement

/** Type of device */
@Serializable
enum class PropertyDeviceType {
    @SerialName("mobile")
    MOBILE,
    @SerialName("tablet")
    TABLET,
    @SerialName("desktop")
    DESKTOP,
    @SerialName("smartTV")
    SMARTTV,
    @SerialName("IoT-Device")
    IOT_DEVICE
}

/** User profile information */
@Serializable
data class CustomTypeUserProfile(
    /** User's email address */
    @SerialName("email")
    val email: PropertyEmail,

    /** User's first name */
    @SerialName("first_name")
    val firstName: PropertyFirstName,

    /** User's last name */
    @SerialName("last_name")
    val lastName: PropertyLastName? = null
)

/** Group association event */
@Serializable
data class GroupTraits(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive
)

/** User identification event */
@Serializable
data class IdentifyTraits(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive? = null,

    /** User's email address */
    @SerialName("email")
    val email: PropertyEmail
)

/** Page view event */
@Serializable
data class PageProperties(
    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile
)

/** Screen view event */
@Serializable
data class ScreenProperties(
    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile? = null
)

/** Triggered when a user signs up */
@Serializable
data class TrackUserSignedUpProperties(
    /** User active status */
    @SerialName("active")
    val active: PropertyActive,

    /** User's age */
    @SerialName("age")
    val age: PropertyAge? = null,

    /** An array that can contain any type of items */
    @SerialName("array_of_any")
    val arrayOfAny: PropertyArrayOfAny? = null,

    /** Array of user contacts */
    @SerialName("contacts")
    val contacts: PropertyContacts? = null,

    /** Type of device */
    @SerialName("device_type")
    val deviceType: PropertyDeviceType? = null,

    /** An object field with no defined structure */
    @SerialName("object_property")
    val objectProperty: PropertyObjectProperty? = null,

    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile,

    /** A field that can contain any type of value */
    @SerialName("property_of_any")
    val propertyOfAny: PropertyPropertyOfAny? = null,

    /** User tags as array of strings */
    @SerialName("tags")
    val tags: PropertyTags? = null,

    /** An array with no explicit item type (treated as any) */
    @SerialName("untyped_array")
    val untypedArray: PropertyUntypedArray? = null,

    /** A field with no explicit type (treated as any) */
    @SerialName("untyped_field")
    val untypedField: PropertyUntypedField? = null
)

class RudderAnalytics(private val analytics: Analytics) {
    private val json = Json {
        prettyPrint = true
        encodeDefaults = false
    }

    private val context = buildJsonObject {
        put("ruddertyper", buildJsonObject {
            put("platform", "test")
            put("rudderCLIVersion", "1.0.0")
            put("trackingPlanId", "plan_12345")
            put("trackingPlanVersion", 13)
        })
    }

    /**
     * Group association event
     */
    fun group(groupId: String, traits: GroupTraits) {
        analytics.group(
            groupId = groupId,
            traits = json.encodeToJsonElement(traits).jsonObject,
            options = RudderOptions(customContext = context)
        )
    }

    /**
     * User identification event
     */
    fun identify(userId: String = "", traits: IdentifyTraits) {
        analytics.identify(
            userId = userId,
            traits = json.encodeToJsonElement(traits).jsonObject,
            options = RudderOptions(customContext = context)
        )
    }

    /**
     * Screen view event
     */
    fun screen(screenName: String, category: String = "", properties: ScreenProperties) {
        analytics.screen(
            screenName = screenName,
            category = category,
            properties = json.encodeToJsonElement(properties).jsonObject,
            options = RudderOptions(customContext = context)
        )
    }

    /**
     * Triggered when a user signs up
     */
    fun trackUserSignedUp(properties: TrackUserSignedUpProperties) {
        analytics.track(
            name = "User Signed Up",
            properties = json.encodeToJsonElement(properties).jsonObject,
            options = RudderOptions(customContext = context)
        )
    }
}
