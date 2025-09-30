package com.rudderstack.ruddertyper

import com.rudderstack.sdk.kotlin.core.Analytics
import kotlinx.serialization.Serializable
import kotlinx.serialization.SerialName
import kotlinx.serialization.json.Json
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

/** User's email address */
typealias PropertyEmail = CustomTypeEmail

/** User's first name */
typealias PropertyFirstName = String

/** User's last name */
typealias PropertyLastName = String

/** User profile data */
typealias PropertyProfile = CustomTypeUserProfile

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

    /** Type of device */
    @SerialName("device_type")
    val deviceType: PropertyDeviceType? = null,

    /** User profile data */
    @SerialName("profile")
    val profile: PropertyProfile
)

class RudderAnalytics(private val analytics: Analytics) {
    private val json = Json {
        prettyPrint = true
        encodeDefaults = false
    }

    /**
     * Group association event
     */
    fun group(groupId: String, traits: GroupTraits) {
        analytics.group(
            groupId = groupId,
            traits = json.encodeToJsonElement(traits).jsonObject
        )
    }

    /**
     * User identification event
     */
    fun identify(userId: String = "", traits: IdentifyTraits) {
        analytics.identify(
            userId = userId,
            traits = json.encodeToJsonElement(traits).jsonObject
        )
    }

    /**
     * Screen view event
     */
    fun screen(screenName: String, category: String = "", properties: ScreenProperties) {
        analytics.screen(
            screenName = screenName,
            category = category,
            properties = json.encodeToJsonElement(properties).jsonObject
        )
    }

    /**
     * Triggered when a user signs up
     */
    fun trackUserSignedUp(properties: TrackUserSignedUpProperties) {
        analytics.track(
            name = "User Signed Up",
            properties = json.encodeToJsonElement(properties).jsonObject
        )
    }
}
